package store

import (
	"fmt"
	"net/url"

	res "github.com/jirenius/go-res"
)

// QueryHandler is a res.Service handler for get requests that fetches the
// results from an underlying QueryStore. It also listens to changes in the
// QueryStore, and sends events if the change affects the query.
type QueryHandler struct {
	// A QueryStore from where to fetch the resource references.
	QueryStore QueryStore
	// QueryRequestHandler transforms an external query request and path param
	// values into a query that can be handled by the QueryStore. The QueryHandler will rerespond with a query resource.
	//
	// A non-empty normalized query string based on the url.Values must be returned.
	// See: https://resgate.io/docs/specification/res-service-protocol/#get-request
	//
	// Must not be set if RequestHandler is set.
	QueryRequestHandler func(rname string, pathParams map[string]string, q url.Values) (url.Values, string, error)
	// RequestHandler transforms an external request and path param values into
	// a query that can be handled by the QueryStore.
	//
	// Must not be set if QueryRequestHandler is set.
	RequestHandler func(rname string, pathParams map[string]string) (url.Values, error)
	// Transformer transforms the internal query results and events into an external
	// resource, and events for the external resource.
	Transformer QueryTransformer
	// AffectedResources is called on query change, and should return a list of
	// resources affected by the change.
	//
	// This is only required if the resource contains path parameters.
	//
	// Example
	//
	// If a the handler listens for requests on: library.books.$firstLetter
	// A change of a book's name from "Alpha" to "Beta" should have
	// AffectedResources return the following:
	//
	// 	[]string{"library.books.a", "library.books.b"}
	AffectedResources func(res.Pattern, QueryChange) []string
}

var _ res.Option = QueryHandler{}

type queryHandler struct {
	s       *res.Service
	pattern res.Pattern
	typ     res.ResourceType
	qs      QueryStore
	qrh     func(string, map[string]string, url.Values) (url.Values, string, error)
	rh      func(string, map[string]string) (url.Values, error)
	trans   QueryTransformer
	ar      func(res.Pattern, QueryChange) []string
	isQuery bool
}

// WithQueryStore returns a new QueryHandler value with QueryStore set to qstore.
func (qh QueryHandler) WithQueryStore(qstore QueryStore) QueryHandler {
	qh.QueryStore = qstore
	return qh
}

// WithQueryRequestHandler returns a new QueryHandler value with QueryRequestHandler set to f.
func (qh QueryHandler) WithQueryRequestHandler(f func(rname string, pathParams map[string]string, q url.Values) (url.Values, string, error)) QueryHandler {
	qh.QueryRequestHandler = f
	return qh
}

// WithRequestHandler returns a new QueryHandler value with RequestHandler set to f.
func (qh QueryHandler) WithRequestHandler(f func(rname string, pathParams map[string]string) (url.Values, error)) QueryHandler {
	qh.RequestHandler = f
	return qh
}

// WithTransformer returns a new QueryHandler value with Tranformer set to transformer.
func (qh QueryHandler) WithTransformer(transformer QueryTransformer) QueryHandler {
	qh.Transformer = transformer
	return qh
}

// WithAffectedResources returns a new QueryHandler value with IDToRID set to f.
func (qh QueryHandler) WithAffectedResources(f func(res.Pattern, QueryChange) []string) QueryHandler {
	qh.AffectedResources = f
	return qh
}

// SetOption is to implement the res.Option interface
func (qh QueryHandler) SetOption(h *res.Handler) {
	if qh.QueryStore == nil {
		panic("no QueryStore is set")
	}
	if qh.QueryRequestHandler != nil && qh.RequestHandler != nil {
		panic("both RequestHandler and QueryRequestHandler is set")
	}
	o := queryHandler{
		qs:      qh.QueryStore,
		qrh:     qh.QueryRequestHandler,
		rh:      qh.RequestHandler,
		trans:   qh.Transformer,
		ar:      qh.AffectedResources,
		isQuery: qh.QueryRequestHandler != nil,
	}
	h.Option(res.OnRegister(o.onRegister))
	// Set conditional handler methods depending if we have
	// an ordinary resource or a query resource.
	if o.isQuery {
		h.Option(res.GetResource(o.getQueryResource))
		o.qs.OnQueryChange(o.queryChangeHandler)
	} else {
		h.Option(res.GetResource(o.getResource))
		o.qs.OnQueryChange(o.changeHandler)
	}
}

func (o *queryHandler) onRegister(s *res.Service, p res.Pattern, h res.Handler) {
	if res.Pattern(p).IndexWildcard() >= 0 {
		if o.ar == nil {
			panic("QueryHandler requires an AffectedResources callback when handling resources with tags or wildcards: " + p)
		}
	} else {
		o.ar = nil
	}
	if h.Type == res.TypeUnset {
		panic("no Type is set")
	}
	if h.Type != res.TypeModel && h.Type != res.TypeCollection {
		panic("Type must be set to TypeModel or TypeCollection")
	}
	o.s = s
	o.pattern = p
	o.typ = h.Type
}

func (o *queryHandler) getResource(r res.GetRequest) {
	var err error
	var q url.Values
	if o.rh != nil {
		q, err = o.rh(r.ResourceName(), r.PathParams())
		if err != nil {
			r.Error(err)
			return
		}
	}

	result, err := o.getResult(q)
	if err != nil {
		r.Error(err)
		return
	}

	switch o.typ {
	case res.TypeModel:
		r.Model(result)
	case res.TypeCollection:
		r.Collection(result)
	default:
		panic("invalid type")
	}
}

func (o *queryHandler) getQueryResource(r res.GetRequest) {
	q, norm, err := o.qrh(r.ResourceName(), r.PathParams(), r.ParseQuery())
	if err != nil {
		r.Error(err)
		return
	}

	result, err := o.getResult(q)
	if err != nil {
		r.Error(err)
		return
	}

	if norm == "" {
		panic("QueryResourceHandler returned an empty normalized query string for resource: " + r.ResourceName())
	}

	switch o.typ {
	case res.TypeModel:
		r.QueryModel(result, norm)
	case res.TypeCollection:
		r.QueryCollection(result, norm)
	default:
		panic("invalid type")
	}

}

func (o *queryHandler) queryChangeHandler(qc QueryChange) {
	if o.ar != nil {
		qrids := o.ar(o.pattern, qc)
		for _, qrid := range qrids {
			o.queryEvent(qrid, qc)
		}
	} else {
		o.queryEvent(string(o.pattern), qc)
	}
}

func (o *queryHandler) changeHandler(qc QueryChange) {
	if o.ar != nil {
		rids := o.ar(o.pattern, qc)
		for _, rid := range rids {
			if err := o.resourceEvent(rid, qc); err != nil {
				o.errorf("QueryHandler encountered error generating events for resource %s: %s", rid, err)
				return
			}
		}
	} else {
		if err := o.resourceEvent(string(o.pattern), qc); err != nil {
			o.errorf("QueryHandler encountered error generating events for resource %s: %s", o.pattern, err)
		}
	}
}

func (o *queryHandler) resourceEvent(rid string, qc QueryChange) error {
	r, err := o.s.Resource(rid)
	if err != nil {
		return fmt.Errorf("error getting resource: %s", err)
	}

	var q url.Values
	if o.rh != nil {
		q, err = o.rh(r.ResourceName(), r.PathParams())
		if err != nil {
			return fmt.Errorf("error calling RequestHandler: %s", err)
		}
	}

	// Get events. We assume Events will will by itself determine if
	// the query is affected or not, by returning a no events.
	evs, reset, err := qc.Events(q)
	if err != nil {
		return fmt.Errorf("error getting events: %s", err)
	}

	if reset {
		r.ResetEvent()
		return nil
	}

	if len(evs) == 0 {
		return nil
	}

	if o.trans != nil {
		evs, err = o.trans.TransformEvents(evs)
		if err != nil {
			return fmt.Errorf("error transforming events: %s", err)
		}
	}

	for _, ev := range evs {
		switch ev.Name {
		case "add":
			r.AddEvent(ev.Value, ev.Idx)
		case "remove":
			r.RemoveEvent(ev.Idx)
		case "change":
			r.ChangeEvent(ev.Changed)
		default:
			return fmt.Errorf("invalid event name: %s", ev.Name)
		}
	}
	return nil
}

func (o *queryHandler) queryEvent(qrid string, qc QueryChange) {
	qcr, err := o.s.Resource(qrid)
	if err != nil {
		panic(err)
	}
	qcr.QueryEvent(func(qreq res.QueryRequest) {
		// Nil means end of query event.
		if qreq == nil {
			return
		}

		q, _, err := o.qrh(qreq.ResourceName(), qreq.PathParams(), qreq.ParseQuery())
		if err != nil {
			qreq.Error(err)
			return
		}

		// Get events for the query
		evs, reset, err := qc.Events(q)
		if err != nil {
			qreq.Error(err)
			return
		}

		// Handle reset
		if reset {
			result, err := o.getResult(q)
			if err != nil {
				qreq.Error(err)
				return
			}
			switch o.typ {
			case res.TypeModel:
				qreq.Model(result)
			case res.TypeCollection:
				qreq.Collection(result)
			default:
				panic("invalid type")
			}
			return
		}

		if len(evs) == 0 {
			return
		}

		if o.trans != nil {
			evs, err = o.trans.TransformEvents(evs)
			if err != nil {
				panic(err)
			}
		}

		for _, ev := range evs {
			switch ev.Name {
			case "add":
				qreq.AddEvent(ev.Value, ev.Idx)
			case "remove":
				qreq.RemoveEvent(ev.Idx)
			case "change":
				qreq.ChangeEvent(ev.Changed)
			default:
				panic("invalid event name: " + ev.Name)
			}
		}
	})
}

func (o *queryHandler) getResult(q url.Values) (interface{}, error) {
	result, err := o.qs.Query(q)
	if err != nil {
		return nil, err
	}

	// Transform results
	if o.trans != nil {
		result, err = o.trans.TransformResult(result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (o *queryHandler) errorf(format string, v ...interface{}) {
	l := o.s.Logger()
	if l != nil {
		l.Errorf(format, v...)
	}
}
