package store

import (
	"net/url"

	"github.com/jirenius/go-res"
)

// Predefined errors
var (
	ErrNotFound  = res.ErrNotFound
	ErrDuplicate = &res.Error{Code: res.CodeInvalidParams, Message: "Duplicate resource"}
)

// Store is a CRUD interface for storing resources of a specific type. The
// resources are identified by a unique ID string.
//
// The value type returned by ReadTxn.Value, and passed to the OnChange
// callback, is determined by the Store, and remains the same for all calls. The
// Store expects that the same type is also passed as values to the
// WriteTxn.Create and WriteTxn.Update methods.
type Store interface {
	// Read will return a read transaction once there are no open write
	// transaction for the same resource. The transaction will last until Close
	// is called.
	Read(id string) ReadTxn

	// Write will return a write transaction once there are no open read or
	// write transactions for the same resource. The transaction will last until
	// Close is called.
	//
	// If the Store implementation does not support the caller generating its
	// own ID for resource creation, the implementation's Write method may
	// accept an empty ID string. In such case, any call to WriteTxn.Value,
	// WriteTxn.Update, and WriteTxn.Delete returns ErrNotFound (or an error
	// that wraps ErrNotFound), until WriteTxn.Create has been called. After
	// Create is called, the ID method should returns the new ID. If the Store
	// implementation does not support generating new IDs, a call to
	// WriteTxn.Create with an empty ID returns an error.
	Write(id string) WriteTxn

	// OnChange registers a callback that is called whenever a resource has been
	// modified. The callback parameters describes the ID of the modified
	// resource, and the value before and after modification.
	//
	// If the before-value is nil, the resource was created. If the after-value
	// is nil, the resource was deleted.
	OnChange(func(id string, before, after interface{}))
}

// ReadTxn represents a read transaction.
type ReadTxn interface {
	// ID returns the ID string of the resource.
	ID() string

	// Close closes the transaction, rendering it unusable for any subsequent
	// calls. Close will return an error if it has already been called.
	Close() error

	// Exists returns true if the value exists, or false on read error or if the
	// resource does not exist.
	Exists() bool

	// Value returns the stored value. Value returns ErrNotFound (or an error
	// that wraps ErrNotFound), if a resource with the provided ID does not
	// exist in the store.
	Value() (interface{}, error)
}

// WriteTxn represents a write transaction.
type WriteTxn interface {
	ReadTxn

	// Create adds a new value to the store.
	//
	// If a resource with the same ID already exists in the store, or if a
	// unique index is violated, Create returns ErrDuplicate (or an error that
	// wraps ErrDuplicate).
	//
	// If the value is successfully created, the Store OnChange callbacks will
	// be triggered on the calling goroutine with the before-value set to nil.
	Create(interface{}) error

	// Update replaces an existing value in the store.
	//
	// If the value does not exist, Update returns ErrNotFound (or an error that
	// wraps ErrNotFound).
	//
	// If the value is successfully updated, the Store OnChange callbacks will
	// be triggered on the calling goroutine.
	Update(interface{}) error

	// Delete deletes an existing value from the store.
	//
	// If the value does not exist, Delete returns ErrNotFound (or an error that
	// wraps ErrNotFound).
	//
	// If the value is successfully deleted, the Store OnChange callbacks will
	// be triggered on the calling goroutine with the after-value set to nil.
	Delete() error
}

// QueryStore is an interface for quering the resource in a store.
type QueryStore interface {
	// Query returns a result based on the provided query values.
	//
	// The result type is determined by the QueryStore implementation, and must
	// remain the same for all calls regardless of query values. If error is
	// non-nil the returned interface{} is nil.
	Query(query url.Values) (interface{}, error)

	// OnQueryChange registers a callback that is called whenever a change to a
	// reasource has occurred that may affect the results returned by Query.
	OnQueryChange(func(QueryChange))
}

// QueryChange represents a change to a resource that may affects queries.
type QueryChange interface {
	// ID returns the ID of the changed resource triggering the event.
	ID() string

	// Before returns the resource value before the change. The value type is
	// defined by the underlying store. If the resource was created, Before will
	// return nil.
	Before() interface{}

	// After returns the resource value after the change. The value type is
	// defined by the underlying store. If the resource was deleted, After will
	// return nil.
	After() interface{}

	// Events returns a list of events that describes mutations of the results,
	// caused by the change, for a given query.
	//
	// If the query result is a collection, where the change caused a value to
	// move position, the "remove" event should come prior to the "add" event.
	//
	// The QueryStore implementation may return zero or nil events, even if the
	// query may be affected by the change, but must then have the returned
	// reset flag set to true.
	Events(q url.Values) (events []ResultEvent, reset bool, err error)
}

// ResultEvent represents an event on a query result.
//
// See: https://resgate.io/docs/specification/res-service-protocol/#events
type ResultEvent struct {
	// Name of the event.
	Name string

	// Index position where the resource is added or removed from the query
	// result.
	//
	// Only valid for "add" and "remove" events.
	Idx int

	// ID of resource being added or removed from the query result.
	//
	// Only valid for "add" and "remove" events.
	Value interface{}

	// Changed property values for the model emitting the event.
	//
	// Only valid for "change" events.
	Changed map[string]interface{}
}

// Transformer is an interface with methods to transform a stored resource into
// a resource served by the service.
type Transformer interface {
	// RIDToID transforms an external resource ID to the internal ID, used by
	// the store. An empty ID will be interpreted as resource not found.
	RIDToID(rid string, pathParams map[string]string) string

	// IDToRID transforms an internal ID, used by the store, to an external
	// resource ID. Pattern is the full pattern for the resource ID.
	//
	// An empty RID will be interpreted as resource not found.
	IDToRID(id string, v interface{}, pattern res.Pattern) string

	// Transform transforms an internal value, persisted in the store, to an
	// external resource to send to the requesting client.
	Transform(id string, v interface{}) (interface{}, error)
}

// QueryTransformer is an interface with methods to transform and validate an
// incoming query so that it can be passed to a QueryStore. And transforming the
// results so that it can be returned as an external resource.
type QueryTransformer interface {
	// TransformResults transforms a query result into an external resource to
	// send to the requesting client.
	TransformResult(v interface{}) (interface{}, error)

	// TransformEvents transform events, as returned from QueryChange.Events
	// into events for the external resource.
	TransformEvents(events []ResultEvent) ([]ResultEvent, error)
}
