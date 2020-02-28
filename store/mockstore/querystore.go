package mockstore

import (
	"net/url"

	"github.com/jirenius/go-res/store"
)

// QueryStore mocks a query store.
//
// It implements the store.QueryStore interface.
type QueryStore struct {
	// OnQueryChangeCallbacks contains callbacks added with OnQueryChange.
	OnQueryChangeCallbacks []func(store.QueryChange)

	// OnQuery handles calls to Query.
	OnQuery func(q url.Values) (interface{}, error)
}

// Assert *QueryStore implements the store.QueryStore interface.
var _ store.QueryStore = &QueryStore{}

// QueryChange mocks a change in a resource that affects the query.
//
// It implements the store.QueryChange interface.
type QueryChange struct {
	IDValue        string
	BeforeValue    interface{}
	AfterValue     interface{}
	OnAffectsQuery func(q url.Values) bool
	OnEvents       func(q url.Values) ([]store.ResultEvent, bool, error)
}

// Assert QueryChange implements the store.QueryChange interface.
var _ store.QueryChange = QueryChange{}

// NewQueryStore creates a new QueryStore and initializes it.
func NewQueryStore(cb func(q url.Values) (interface{}, error)) *QueryStore {
	return &QueryStore{
		OnQuery: cb,
	}
}

// Query returns a collection of references to store ID's matching
// the query. If error is non-nil the reference slice is nil.
func (qs *QueryStore) Query(q url.Values) (interface{}, error) {
	return qs.OnQuery(q)
}

// OnQueryChange adds a listener callback that is triggered using the
// TriggerQueryChange.
func (qs *QueryStore) OnQueryChange(cb func(store.QueryChange)) {
	qs.OnQueryChangeCallbacks = append(qs.OnQueryChangeCallbacks, cb)
}

// TriggerQueryChange call all OnQueryChange listeners with the QueryChange.
func (qs *QueryStore) TriggerQueryChange(qc QueryChange) {
	for _, cb := range qs.OnQueryChangeCallbacks {
		cb(qc)
	}
}

// ID returns the IDValue string.
func (qc QueryChange) ID() string {
	return qc.IDValue
}

// Before returns the BeforeValue.
func (qc QueryChange) Before() interface{} {
	return qc.BeforeValue
}

// After returns the AfterValue.
func (qc QueryChange) After() interface{} {
	return qc.AfterValue
}

// Events calls the OnEvents callback, or returns nil and false if OnEvents is
// nil.
func (qc QueryChange) Events(q url.Values) ([]store.ResultEvent, bool, error) {
	if qc.OnEvents == nil {
		return nil, false, nil
	}
	return qc.OnEvents(q)
}
