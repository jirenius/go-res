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

func (o *storeHandler) onRegister(s *res.Service, _ string, h res.Handler) {
	if h.Type == res.TypeUnset {
		panic("no Type is set")
	}
	if h.Type != res.TypeModel && h.Type != res.TypeCollection {
		panic("Type must be set to TypeModel or TypeCollection")
	}
	o.s = s
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
		v, err = o.trans.Transform(v)
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
			before, err = o.trans.Transform(before)
			if err != nil {
				before = nil
			}
		}
		if after != nil {
			after, err = o.trans.Transform(after)
			if err != nil {
				after = nil
			}
		}
		if after != nil {
			rid = o.trans.IDToRID(id, after)
		} else if before != nil {
			rid = o.trans.IDToRID(id, before)
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

	var beforeDta, afterDta []byte
	if beforeDta, err = json.Marshal(before); err != nil {
		o.s.Logger().Errorf("error marshaling resource %s: %s", rid, err)
	}
	if afterDta, err = json.Marshal(after); err != nil {
		o.s.Logger().Errorf("error marshaling resource %s: %s", rid, err)
	}

	var beforeMap, afterMap map[string]Value
	if err = json.Unmarshal(beforeDta, &beforeMap); err != nil {
		o.s.Logger().Errorf("error unmarshaling resource %s to value map: %s", rid, err)
	}
	if err = json.Unmarshal(afterDta, &afterMap); err != nil {
		o.s.Logger().Errorf("error unmarshaling resource %s to value map: %s", rid, err)
	}

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
}
