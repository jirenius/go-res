package store

import (
	"errors"
	"net/url"
)

// Predefined errors
var (
	ErrNotFound  = errors.New("resource not found")
	ErrDuplicate = errors.New("duplicate resource")
)

// Store is a CRUD interface for storing resources of a specific type. The
// resources are identified by a unique ID string.
//
// Read will return a read transaction once there are no open write transaction
// for the same resource. The transaction will last until Close is called.
//
// Write will return a write transaction once there are no open read or write
// transactions for the same resource. The transaction will last until Close is
// called.
//
// If the Store implementation does not support the caller generating its own ID
// for resource creation, the implementation's Write method may accept an empty
// ID string. In such case, any call to WriteTxn.Value, WriteTxn.Update, and
// WriteTxn.Delete must return ErrNotFound (or an error that wraps ErrNotFound),
// until WriteTxn.Create has been called. After Create is called, ID method
// should return the new ID. If the Store implementation does not support
// generating new IDs, a call to WriteTxn.Create with an empty ID should return
// an error.
//
// OnChange registers a callback that is called whenever a resource has been
// modified. The parameters describes which resource has been modified, and the
// value before and after modification. If the before-value is nil, the resource
// was created. If the after-value is nil, the resource was deleted.
//
// The value type returned by ReadTxn.Value, and passed to the OnChange
// callback, is determined by the Store, and must remain the same for all calls.
// The Store should enforce that the same type is also passed as values to the
// WriteTxn.Create and WriteTxn.Update methods.
type Store interface {
	Read(id string) ReadTxn
	Write(id string) WriteTxn
	OnChange(func(id string, before, after interface{}))
}

// ReadTxn represents a read transaction.
//
// ID returns the ID string of the resource.
//
// Any call to Value should return ErrNotFound (or an error that wraps
// ErrNotFound), if a resource with the provided ID does not exist in the store.
//
// Exists returns true if the value exists, or false on read error or if the
// resource does not exist.
//
// Close will return an error if it has already been called.
type ReadTxn interface {
	ID() string
	Close() error
	Exists() bool
	Value() (interface{}, error)
}

// WriteTxn represents a write transaction.
//
// Any call to Update or Delete should return ErrNotFound (or an error that
// wraps ErrNotFound) if the value does not exist.
//
// A call to Create should return ErrDuplicate (or an error that wraps
// ErrDuplicate) if a resource with the same ID already exists in the store, or
// if a unique index is violated.
//
// If any of the methods, Update, Delete, or Create, results in a change, any
// callback registered with Store.OnChange must be called and completed before
// returning from the method.
//
// If a call to Create results in a new resource, the Store OnChange callback
// should be triggered with the before-value set to nil.
//
// If a call to Delete results in the deletion of the resource, the Store
// OnChange callback should be triggered with the after value set to nil.
type WriteTxn interface {
	ReadTxn
	Create(interface{}) error
	Update(interface{}) error
	Delete() error
}

// QueryStore is an interface for quering the resource in a store.
//
// Query returns a result based on the provided query values. The result type is
// determined by the QueryStore implementation, and must remain the same for all
// calls regardless of query values. If error is non-nil the result interface{}
// is nil.
//
// OnQueryChange registers a callback that is called whenever a change to a
// reasource has occurred that may affect the results returned by Query.
type QueryStore interface {
	Query(query url.Values) (interface{}, error)
	OnQueryChange(func(QueryChange))
}

// QueryChange represents a change to a resource that may affects queries.
//
// ID returns the ID of the changed resource triggering the event.
//
// Before returns the resource value before the change. The value type is
// defined by the underlying store. If the resource was created, Before will
// return nil.
//
// After returns the resource value after the change. The value type is defined
// by the underlying store. If the resource was deleted, After will return nil.
//
// AffectsQuery returns true if a given query might be affected by the change,
// otherwise false.
//
// Events returns a list of events that describes mutations of the results for a
// given query. The ResultEvent.Value field should be set for both "add" and
// "remove" events. In case a resource moved position, the "remove" event should
// come prior to the "add" event. The QueryStore implementation may return zero
// or nil events, even if the query may be affected by the change. The eturned
// reset flag must then be set to true. The method must be called before the
// OnQueryChange callback returns.
type QueryChange interface {
	ID() string
	Before() interface{}
	After() interface{}
	AffectsQuery(q url.Values) bool
	Events(q url.Values) (events []ResultEvent, reset bool)
}

// ResultEvent represents an event on a query result.
//
// See: https://resgate.io/docs/specification/res-service-protocol/#events
type ResultEvent struct {
	// Name of the event.
	Name string

	// Index position where the resource is added or removed from the query
	// result. * Only valid for "add" and "remove" events.
	Idx int

	// ID of resource being added or removed from the query result. * Only valid
	// for "add" and "remove" events.
	Value interface{}

	// Changed property values for the model emitting the event. * Only valid
	// for "change" events.
	Changed map[string]interface{}
}

// Transformer is an interface with methods to transform a stored resource into
// a resource served by the service.
//
// RIDToID transforms an external resource ID to the internal ID, used by the
// store. An empty ID will be interpreted as resource not found.
//
// IDToRID transforms an internal ID, used by the store, to an external resource
// ID. An empty RID will be interpreted as resource not found.
//
// Transform transforms an internal value, persisted in the store, to an
// external resource to send to the requesting client.
type Transformer interface {
	RIDToID(rid string, pathParams map[string]string) string
	IDToRID(id string, v interface{}) string
	Transform(v interface{}) (interface{}, error)
}

// QueryTransformer is an interface with methods to transform and validate an
// incoming query so that it can be passed to a QueryStore. And transforming the
// results so that it can be returned as an external resource.
//
// TransformResults transforms a query result into an external resource to send
// to the requesting client.
//
// TransformEvents transform events, as returned from QueryChange.Events into
// events for the external resource.
type QueryTransformer interface {
	TransformResult(v interface{}) (interface{}, error)
	TransformEvents(events []ResultEvent) ([]ResultEvent, error)
}
