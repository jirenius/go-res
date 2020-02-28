package resbadger

import (
	"bytes"
	"net/url"

	res "github.com/jirenius/go-res"
)

// QueryCollection represents a collection of indexed Models that may be queried.
type QueryCollection struct {
	// BadgerDB middleware
	BadgerDB BadgerDB
	// IndexSet defines a set of indexes to be used with query requests.
	IndexSet *IndexSet
	// QueryCallback takes a query request and returns an IndexQuery used for searching.
	QueryCallback QueryCallback
}

// QueryCallback is called for each query request.
// It returns an index query and a normalized query string, or an error.
//
// If the normalized query string is empty, the initial query string is used as normalized query.
type QueryCallback func(idxs *IndexSet, rname string, params map[string]string, query url.Values) (*IndexQuery, string, error)

type queryCollection struct {
	BadgerDB
	idxs          *IndexSet
	queryCallback QueryCallback
	pattern       string
	s             *res.Service
}

// WithIndexSet returns a new QueryCollection value with the IndexSet set to idxs.
func (o QueryCollection) WithIndexSet(idxs *IndexSet) QueryCollection {
	o.IndexSet = idxs
	return o
}

// WithQueryCallback returns a new QueryCollection value with the QueryCallback set to callback.
func (o QueryCollection) WithQueryCallback(callback QueryCallback) QueryCollection {
	o.QueryCallback = callback
	return o
}

// SetOption sets the res handler options,
// and implements the res.Option interface.
func (o QueryCollection) SetOption(hs *res.Handler) {
	// var err error

	if o.BadgerDB.DB == nil {
		panic("middleware: no badger DB set")
	}

	if o.IndexSet == nil {
		panic("resbadger: no indexes set")
	}

	qc := queryCollection{
		BadgerDB:      o.BadgerDB,
		idxs:          o.IndexSet,
		queryCallback: o.QueryCallback,
	}

	res.Collection.SetOption(hs)
	res.GetResource(qc.getQueryCollection).SetOption(hs)
	res.OnRegister(qc.onRegister).SetOption(hs)
	o.IndexSet.Listen(qc.onIndexUpdate)
}

func (qc *queryCollection) onRegister(service *res.Service, pattern res.Pattern, rh res.Handler) {
	qc.s = service
	qc.pattern = string(pattern)
}

// onIndexUpdate is a handler for changes to the indexes used
// by the query collection.
func (qc *queryCollection) onIndexUpdate(r res.Resource, before, after interface{}) {
	qcr, err := r.Service().Resource(qc.pattern)
	if err != nil {
		panic(err)
	}
	qcr.QueryEvent(func(qreq res.QueryRequest) {
		// Nil means end of query event.
		if qreq == nil {
			return
		}

		iq, _, err := qc.queryCallback(qc.idxs, qcr.ResourceName(), qcr.PathParams(), qreq.ParseQuery())
		if err != nil {
			qreq.Error(res.InternalError(err))
			return
		}
		var beforeKey, afterKey []byte
		if before != nil {
			beforeKey = iq.Index.Key(before)
		}
		if after != nil {
			afterKey = iq.Index.Key(after)
		}
		// No event if no change to the index
		if bytes.Equal(beforeKey, afterKey) {
			return
		}
		wasMatch := before != nil && bytes.HasPrefix(beforeKey, iq.KeyPrefix)
		isMatch := after != nil && bytes.HasPrefix(afterKey, iq.KeyPrefix)
		if iq.FilterKeys != nil {
			if wasMatch {
				wasMatch = iq.FilterKeys(beforeKey)
			}
			if isMatch {
				isMatch = iq.FilterKeys(afterKey)
			}
		}
		if wasMatch || isMatch {
			collection, err := iq.FetchCollection(qc.DB)
			if err != nil {
				qreq.Error(res.ToError(err))
			}
			qreq.Collection(collection)
		}
	})
}

// getQueryCollection is a get handler for a query request.
func (qc *queryCollection) getQueryCollection(r res.GetRequest) {
	iq, normalizedQuery, err := qc.queryCallback(qc.idxs, r.ResourceName(), r.PathParams(), r.ParseQuery())
	if err != nil {
		r.Error(res.ToError(err))
		return
	}

	collection, err := iq.FetchCollection(qc.DB)
	if err != nil {
		r.Error(res.ToError(err))
		return
	}

	// Get normalized query, or default to the initial query.
	if normalizedQuery == "" {
		normalizedQuery = r.Query()
	}
	r.QueryCollection(collection, normalizedQuery)
}
