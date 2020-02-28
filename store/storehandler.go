package store

import (
	"encoding/json"
	"errors"

	res "github.com/jirenius/go-res"
)

// Handler is a res.Service handler that fetches resources,
// and listens for updates from a Store.
type Handler struct {
	Store       Store
	Transformer Transformer
}

var _ res.Option = Handler{}

type storeHandler struct {
	s     *res.Service
	p     res.Pattern
	st    Store
	typ   res.ResourceType
	trans Transformer
}

var errInvalidResourceType = res.InternalError(errors.New("invalid store resource type"))

// WithStore returns a new Handler value with Store set to store.
func (sh Handler) WithStore(store Store) Handler {
	sh.Store = store
	return sh
}

// WithTransformer returns a new Handler value with Transformer set to transformer.
func (sh Handler) WithTransformer(transformer Transformer) Handler {
	sh.Transformer = transformer
	return sh
}

// SetOption is to implement the res.Option interface
func (sh Handler) SetOption(h *res.Handler) {
	if sh.Store == nil {
		panic("no Store is set")
	}
	o := storeHandler{
		st:    sh.Store,
		trans: sh.Transformer,
	}
	h.Option(
		res.GetResource(o.getResource),
		res.OnRegister(o.onRegister),
	)
	o.st.OnChange(o.changeHandler)
}

func (o *storeHandler) onRegister(s *res.Service, p res.Pattern, h res.Handler) {
	if h.Type == res.TypeUnset {
		panic("no Type is set")
	}
	if h.Type != res.TypeModel && h.Type != res.TypeCollection {
		panic("Type must be set to TypeModel or TypeCollection")
	}
	o.s = s
	o.p = p
	o.typ = h.Type
}

func (o *storeHandler) getResource(r res.GetRequest) {
	id := r.ResourceName()
	if o.trans != nil {
		id = o.trans.RIDToID(id, r.PathParams())
		if id == "" {
			r.NotFound()
			return
		}
	}

	txn := o.st.Read(id)
	defer txn.Close()

	v, err := txn.Value()
	if err != nil {
		r.Error(res.ToError(err))
		return
	}
	if o.trans != nil {
		v, err = o.trans.Transform(id, v)
		if err != nil {
			r.Error(res.ToError(err))
			return
		}
	}
	switch o.typ {
	case res.TypeModel:
		r.Model(v)
	case res.TypeCollection:
		r.Collection(v)
	default:
		r.Error(errInvalidResourceType)
	}
}

func (o *storeHandler) changeHandler(id string, before, after interface{}) {
	var err error
	rid := id
	if o.trans != nil {
		if before != nil {
			before, err = o.trans.Transform(id, before)
			if err != nil {
				before = nil
			}
		}
		if after != nil {
			after, err = o.trans.Transform(id, after)
			if err != nil {
				after = nil
			}
		}
		if after != nil {
			rid = o.trans.IDToRID(id, after, o.p)
		} else if before != nil {
			rid = o.trans.IDToRID(id, before, o.p)
		}
		if rid == "" {
			return
		}
	}

	r, err := o.s.Resource(rid)
	if err != nil {
		o.s.Logger().Errorf("error getting resource %s: %s", rid, err)
		return
	}

	if before == nil {
		// Assert that both values are not null. Shouldn't happen.
		if after == nil {
			return
		}
		r.CreateEvent(after)
		return
	}

	if after == nil {
		r.DeleteEvent()
		return
	}

	switch o.typ {
	case res.TypeModel:
		if err := modelDiff(r, before, after); err != nil {
			o.s.Logger().Errorf("diff failed for model %s: %s", rid, err)
		}
	case res.TypeCollection:
		if err := o.collectionDiff(r, before, after); err != nil {
			o.s.Logger().Errorf("diff failed for collection %s: %s", rid, err)
		}
	default:
		o.s.Logger().Errorf("invalid resource type")
	}
}

// modelDiff produces change event by comparing before and after value, as they
// look when marshaled into json.
func modelDiff(r res.Resource, before, after interface{}) error {
	var beforeMap, afterMap map[string]Value
	var ok bool

	// Convert before and after value to map[string]Value
	if beforeMap, ok = before.(map[string]Value); !ok {
		beforeDta, err := json.Marshal(before)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(beforeDta, &beforeMap); err != nil {
			return err
		}
	}
	if afterMap, ok = after.(map[string]Value); !ok {
		afterDta, err := json.Marshal(after)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(afterDta, &afterMap); err != nil {
			return err
		}
	}

	// Generate change event
	ch := make(map[string]interface{}, len(afterMap))
	for k := range beforeMap {
		if _, ok := afterMap[k]; !ok {
			ch[k] = DeleteValue
		}
	}
	for k, v := range afterMap {
		ov, ok := beforeMap[k]
		if !ok || !v.Equal(ov) {
			ch[k] = v
		}
	}

	r.ChangeEvent(ch)
	return nil
}

// collectionDiff produces remove and add events by comparing before and after
// value, as they look when marshaled into json.
func (o *storeHandler) collectionDiff(r res.Resource, before, after interface{}) error {
	var a, b []Value
	var ok bool

	// Convert before and after value to []Value
	if a, ok = before.([]Value); !ok {
		beforeDta, err := json.Marshal(before)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(beforeDta, &a); err != nil {
			return err
		}
	}
	if b, ok = after.([]Value); !ok {
		afterDta, err := json.Marshal(after)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(afterDta, &b); err != nil {
			return err
		}
	}

	// Generate remove/add events
	var i, j int
	// Do a LCS matrix calculation
	// https://en.wikipedia.org/wiki/Longest_common_subsequence_problem
	s := 0
	m := len(a)
	n := len(b)

	// Trim of matches at the start and end
	for s < m && s < n && a[s].Equal(b[s]) {
		s++
	}

	if s == m && s == n {
		return nil
	}

	for s < m && s < n && a[m-1].Equal(b[n-1]) {
		m--
		n--
	}

	var aa, bb []Value
	if s > 0 || m < len(a) {
		aa = a[s:m]
		m = m - s
	} else {
		aa = a
	}
	if s > 0 || n < len(b) {
		bb = b[s:n]
		n = n - s
	} else {
		bb = b
	}

	// Create matrix and initialize it
	w := m + 1
	c := make([]int, w*(n+1))

	for i = 0; i < m; i++ {
		for j = 0; j < n; j++ {
			if aa[i].Equal(bb[j]) {
				c[(i+1)+w*(j+1)] = c[i+w*j] + 1
			} else {
				v1 := c[(i+1)+w*j]
				v2 := c[i+w*(j+1)]
				if v2 > v1 {
					c[(i+1)+w*(j+1)] = v2
				} else {
					c[(i+1)+w*(j+1)] = v1
				}
			}
		}
	}

	idx := m + s
	i = m
	j = n
	rems := 0

	var adds [][3]int
	addCount := n - c[w*(n+1)-1]
	if addCount > 0 {
		adds = make([][3]int, 0, addCount)
	}
Loop:
	for {
		m = i - 1
		n = j - 1
		switch {
		case i > 0 && j > 0 && aa[m].Equal(bb[n]):
			idx--
			i--
			j--
		case j > 0 && (i == 0 || c[i+w*n] >= c[m+w*j]):
			adds = append(adds, [3]int{n, idx, rems})
			j--
		case i > 0 && (j == 0 || c[i+w*n] < c[m+w*j]):
			idx--
			r.RemoveEvent(idx)
			rems++
			i--
		default:
			break Loop
		}
	}

	// Do the adds
	l := len(adds) - 1
	for i := l; i >= 0; i-- {
		add := adds[i]
		r.AddEvent(bb[add[0]], add[1]-rems+add[2]+l-i)
	}

	return nil
}
