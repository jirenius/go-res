package badgerstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"

	"github.com/dgraph-io/badger"
	"github.com/jirenius/go-res/logger"
	"github.com/jirenius/go-res/store"
	"github.com/jirenius/taskqueue"
)

// QueryStore allows for querying resources in a Store.
//
// It implements the res.QueryStore interface.
//
// A QueryStore should be created using NewQueryStore.
type QueryStore struct {
	st            *Store
	onQueryChange []func(store.QueryChange)
	tq            *taskqueue.TaskQueue
	log           logger.Logger
	idxs          map[string]Index
	iq            func(qs *QueryStore, q url.Values) (*IndexQuery, error)
}

var _ store.QueryStore = &QueryStore{}

type queryChange struct {
	qs     *QueryStore
	id     string
	before interface{}
	after  interface{}
}

const taskCapacity = 256

// NewQueryStore creates a new QueryStore and initializes it.
//
// The type of typ will be used as value. If the type supports both the
// encoding.BinaryMarshaler and the encoding.BinaryUnmarshaler, those method
// will be used for marshaling the values. Otherwise, encoding/json will be
// used for marshaling.
//
// The index query callback, iq, will be called on queries to transform a set of
// url.Values to an *IndexQuery and a normalized query string. In case the query
// callback returns an error, both the IndexQuery value and the normalized query
// string will be ignored.
func NewQueryStore(st *Store, iq func(qs *QueryStore, q url.Values) (*IndexQuery, error)) *QueryStore {
	qs := QueryStore{
		st: st,
		tq: taskqueue.NewTaskQueue(taskCapacity),
		iq: iq,
	}
	st.OnChange(qs.handleChange)
	return &qs
}

// AddIndex adds an index to the query store.
func (qs *QueryStore) AddIndex(idx Index) *QueryStore {
	if qs.idxs == nil {
		qs.idxs = make(map[string]Index)
	}
	if _, ok := qs.idxs[idx.Name]; ok {
		panic(`index "` + idx.Name + `" already exists"`)
	}
	qs.idxs[idx.Name] = idx
	return qs
}

// Index returns the named index.
// Panics if the index does not exist.
func (qs *QueryStore) Index(name string) Index {
	idx, ok := qs.idxs[name]
	if !ok {
		panic(`index "` + name + `" does not exist"`)
	}
	return idx
}

// RebuildIndexes drops current index entries and creates new ones.
func (qs *QueryStore) RebuildIndexes() error {
	// Quick exit in case no index exists
	if len(qs.idxs) == 0 {
		return nil
	}

	// Drop existing index entries
	for _, idx := range qs.idxs {
		err := qs.st.DB.DropPrefix(idx.getQuery(nil))
		if err != nil {
			return err
		}
	}

	// Create new index entries in a single transaction
	return qs.st.DB.Update(func(txn *badger.Txn) error {
		t := reflect.TypeOf(qs.st.Type())
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(qs.st.prefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			// Load item and unmarshal it
			item := it.Item()
			v := reflect.New(t)
			err := item.Value(func(dta []byte) error {
				return json.Unmarshal(dta, v.Interface())
			})
			if err != nil {
				return err
			}
			// Loop through indexes and generate a new entry per index
			for _, idx := range qs.idxs {
				rname := item.KeyCopy(nil)[len(prefix):]
				k := idx.getKey(rname, idx.Key(v.Elem().Interface()))
				if err := txn.Set(k, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// SetLogger sets the logger.
func (qs *QueryStore) SetLogger(l logger.Logger) *QueryStore {
	qs.log = l
	return qs
}

// Query performs a query towards the Store. If error is non-nil the result is
// nil.
func (qs *QueryStore) Query(q url.Values) (interface{}, error) {
	iq, err := qs.iq(qs, q)
	if err != nil {
		return nil, err
	}
	result, err := iq.FetchCollection(qs.st.DB)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// OnQueryChange adds a listener callback that is called whenever a value change
// may have affected the results of the queries.
func (qs *QueryStore) OnQueryChange(cb func(store.QueryChange)) {
	qs.onQueryChange = append(qs.onQueryChange, cb)
}

// Flush waits for the indexing queue to be cleared.
func (qs *QueryStore) Flush() {
	qs.tq.Flush()
}

func (qs *QueryStore) handleChange(id string, before, after interface{}) {
	qs.tq.Do(func() {
		err := qs.updateIndex(id, before, after)
		if err != nil {
			if qs.log != nil {
				qs.log.Errorf("Error updating index: %s", err)
			}
		}
	})
}

func (qs *QueryStore) updateIndex(id string, before, after interface{}) error {
	updated := false
	errmsg := ""
	err := qs.st.DB.Update(func(txn *badger.Txn) error {
		rname := []byte(id)
		// Update index entries
		for _, idx := range qs.idxs {
			var beforeKey, afterKey []byte
			if before != nil {
				beforeKey = idx.Key(before)
			}
			if after != nil {
				afterKey = idx.Key(after)
			}

			// Do nothing if key hasn't change; before and after is equal
			if bytes.Equal(beforeKey, afterKey) {
				continue
			}

			// Delete old index entry
			if len(beforeKey) > 0 {
				k := idx.getKey(rname, beforeKey)
				if err := txn.Delete(k); err != nil {
					errmsg += "\n\terror deleting index key " + string(k) + ": " + err.Error()
				}
			}
			// Set new index entry
			if len(afterKey) > 0 {
				k := idx.getKey(rname, afterKey)
				if err := txn.Set(k, nil); err != nil {
					errmsg += "\n\terror setting index key " + string(k) + ": " + err.Error()
				}
			}

			updated = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if errmsg != "" {
		return errors.New("failed to update resource [" + id + "] index:" + errmsg)
	}
	if updated {
		qc := store.QueryChange(queryChange{
			qs:     qs,
			id:     id,
			before: before,
			after:  after,
		})
		for _, cb := range qs.onQueryChange {
			cb(qc)
		}
	}
	return nil
}

func (qc queryChange) ID() string {
	return qc.id
}

func (qc queryChange) Before() interface{} {
	return qc.before
}

func (qc queryChange) After() interface{} {
	return qc.after
}

func (qc queryChange) Events(q url.Values) ([]store.ResultEvent, bool, error) {
	// [TODO] Fetch the results using the query, and compare the results with
	// the Before and After values to determine which events the change results
	// in.
	affected, err := qc.affectsQuery(q)
	if err != nil {
		return nil, false, err
	}

	return nil, affected, nil
}

func (qc queryChange) affectsQuery(q url.Values) (bool, error) {
	iq, err := qc.qs.iq(qc.qs, q)
	if err != nil {
		return false, err
	}
	var beforeKey, afterKey []byte
	if qc.before != nil {
		beforeKey = iq.Index.Key(qc.before)
	}
	if qc.after != nil {
		afterKey = iq.Index.Key(qc.after)
	}
	// Not affected if no change to the index
	if bytes.Equal(beforeKey, afterKey) {
		return false, nil
	}
	wasMatch := qc.before != nil && bytes.HasPrefix(beforeKey, iq.KeyPrefix)
	isMatch := qc.after != nil && bytes.HasPrefix(afterKey, iq.KeyPrefix)
	if iq.FilterKeys != nil {
		if wasMatch {
			wasMatch = iq.FilterKeys(beforeKey)
		}
		if isMatch {
			isMatch = iq.FilterKeys(afterKey)
		}
	}
	return wasMatch || isMatch, nil
}
