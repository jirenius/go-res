package resbadger

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/dgraph-io/badger"
	res "github.com/jirenius/go-res"
)

// QueryCollection represents a collection of indexed Models that may be queried.
type QueryCollection struct {
	// BadgerDB middleware
	BadgerDB BadgerDB
	// Indexes defines a set of indexes to be used with query requests.
	Indexes *Indexes
	// QueryCallback takes a query request and returns an IndexQuery used for searching.
	QueryCallback QueryCallback
}

// QueryCallback is called for each query request
type QueryCallback func(idxs *Indexes, rname string, params map[string]string, query url.Values) (*IndexQuery, error)

type queryCollection struct {
	BadgerDB
	indexes       *Indexes
	queryCallback func(idxs *Indexes, rname string, params map[string]string, query url.Values) (*IndexQuery, error)
	pattern       string
	s             *res.Service
}

// Max initial buffer size for results, and default size for limit set to -1.
var resultBufSize = 256

// WithIndexes returns a new QueryCollection value with the Indexes set to idx.
func (o QueryCollection) WithIndexes(idxs *Indexes) QueryCollection {
	o.Indexes = idxs
	return o
}

// WithQueryCallback returns a new QueryCollection value with the QueryCallback set to callback.
func (o QueryCollection) WithQueryCallback(callback func(idxs *Indexes, rname string, params map[string]string, query url.Values) (*IndexQuery, error)) QueryCollection {
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

	if o.Indexes == nil {
		panic("resbadger: no indexes set")
	}

	qc := queryCollection{
		BadgerDB:      o.BadgerDB,
		indexes:       o.Indexes,
		queryCallback: o.QueryCallback,
	}

	res.Collection.SetOption(hs)
	res.GetResource(qc.getQueryCollection).SetOption(hs)
	res.OnRegister(qc.onRegister).SetOption(hs)
	o.Indexes.Listen(qc.onIndexUpdate)
}

func (qc *queryCollection) onRegister(service *res.Service, pattern string) {
	qc.s = service
	qc.pattern = pattern
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

		iq, err := qc.queryCallback(qc.indexes, r.ResourceName(), r.PathParams(), qreq.ParseQuery())
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
			collection, err := qc.fetchCollection(iq)
			if err != nil {
				qreq.Error(res.ToError(err))
			}
			qreq.Collection(collection)
		}
	})
}

// getQueryCollection is a get handler for a query request.
func (qc *queryCollection) getQueryCollection(r res.GetRequest) {
	iq, err := qc.queryCallback(qc.indexes, r.ResourceName(), r.PathParams(), r.ParseQuery())
	if err != nil {
		r.Error(res.ToError(err))
		return
	}

	collection, err := qc.fetchCollection(iq)
	if err != nil {
		r.Error(res.ToError(err))
		return
	}

	// Get normalized query, or default to the initial query.
	normalizedQuery := iq.NormalizedQuery
	if normalizedQuery == "" {
		normalizedQuery = r.Query()
	}
	r.QueryCollection(collection, normalizedQuery)
}

// fetchCollection fetches a collection that matches the IndexQuery iq.
func (qc *queryCollection) fetchCollection(iq *IndexQuery) ([]res.Ref, error) {
	offset := iq.Offset
	limit := iq.Limit

	// Quick exit if we are fetching zero items
	if limit == 0 {
		return nil, nil
	}

	// Prepare a slice to store the results in
	buf := resultBufSize
	if limit > 0 && limit < resultBufSize {
		buf = limit
	}
	result := make([]res.Ref, 0, buf)

	queryPrefix := iq.Index.getQuery(iq.KeyPrefix)
	qplen := len(queryPrefix)

	filter := iq.FilterKeys
	namelen := len(iq.Index.Name) + 1

	if err := qc.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(queryPrefix); it.ValidForPrefix(queryPrefix); it.Next() {
			k := it.Item().Key()
			idx := bytes.LastIndexByte(k, ridSeparator)
			if idx < 0 {
				return fmt.Errorf("index entry [%s] is invalid", k)
			}
			// Validate that a query with ?-mark isn't mistaken for a hit
			// when matching the ? separator for the resource ID.
			if qplen > idx {
				continue
			}

			// If we have a key filter, validate against it
			if filter != nil {
				if !filter(k[namelen:idx]) {
					continue
				}
			}

			// Skip until we reach the offset we are searching from
			if offset > 0 {
				offset--
				continue
			}

			// Add resource ID reference to result
			result = append(result, res.Ref(k[idx+1:]))

			limit--
			if limit == 0 {
				return nil
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}
