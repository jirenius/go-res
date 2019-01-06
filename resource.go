package res

import (
	"net/url"
)

// Resource represents a resource
type Resource interface {
	// Service returns the service instance
	Service() *Service

	/// Resource returns the resource name.
	ResourceName() string

	// PathParams returns parameters that are derived from the resource name.
	PathParams() map[string]string

	// Query returns the query part of the resource ID without the question mark separator.
	Query() string

	// ParseQuery parses the query and returns the corresponding values.
	// It silently discards malformed value pairs.
	// To check errors use url.ParseQuery(Query()).
	ParseQuery() url.Values

	// Value gets the resource value as provided from the GetModel or
	// GetCollection resource handlers.
	// If it fails to get the resource value, or no get handler is
	// defined, it returns a nil interface and a *Error type error.
	Value() (interface{}, error)

	// Event sends a custom event on the resource.
	// Will panic if the event is one of the pre-defined or reserved events,
	// "change", "add", "remove", "reaccess", or "unsubscribe".
	// For pre-defined events, the matching method, ChangeEvent, AddEvent,
	// RemoveEvent, or ReaccessEvent should be used instead.
	//
	// See the protocol specification for more information:
	// https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#events
	Event(event string, payload interface{})

	// ChangeEvents sends a change event with properties that has been changed
	// and their new values.
	// If props is empty, no event is sent.
	// Panics if the resource is not a Model.
	// The values must be serializable into JSON primitives, resource references,
	// or a delete action objects.
	// See the protocol specification for more information:
	//    https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#model-change-event
	ChangeEvent(props map[string]interface{})

	// AddEvent sends an add event, adding the value at index idx.
	// Panics if the resource is not a Collection, or if idx is less than 0.
	// The value must be serializable into a JSON primitive or resource reference.
	// See the protocol specification for more information:
	//    https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#collection-add-event
	AddEvent(value interface{}, idx int)

	// RemoveEvent sends a remove event, removing the value at index idx.
	// Panics if the resource is not a Collection, or if idx is less than 0.
	// See the protocol specification for more information:
	//    https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#collection-remove-event
	RemoveEvent(idx int)

	// ReaccessEvent sends a reaccess event to signal that the resource's access permissions has changed.
	// It will invalidate any previous access response sent for the resource.
	// See the protocol specification for more information:
	//    https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#reaccess-event
	ReaccessEvent()
}

// resource is the internal implementation of the Resource interface
type resource struct {
	rname      string
	pathParams map[string]string
	query      string
	s          *Service
	hs         *regHandler
}

// Service returns the service instance
func (r *resource) Service() *Service {
	return r.s
}

// ResourceName returns the resource name.
func (r *resource) ResourceName() string {
	return r.rname
}

// PathParams returns parameters that are derived from the resource name.
func (r *resource) PathParams() map[string]string {
	return r.pathParams
}

// Query returns the query part of the resource ID without the question mark separator.
func (r *resource) Query() string {
	return r.query
}

// ParseQuery parses the query and returns the corresponding values.
// It silently discards malformed value pairs.
// To check errors use url.ParseQuery.
func (r *resource) ParseQuery() url.Values {
	v, _ := url.ParseQuery(r.query)
	return v
}

// Value gets the resource value as provided from the GetModel or
// GetCollection resource handlers.
// If it fails to get the resource value, or no get handler is
// defined, it returns a nil interface and a *Error type error.
func (r *resource) Value() (interface{}, error) {
	panic("not implemented")
	return nil, ErrNotFound

	// switch r.hs.typ {

	// default:
	// 	return nil, ErrNotFound
	// }

	// req := &Request{}
	// r.hs.GetModel()
}

func isValidEvent(ev string) bool {
	for _, r := range ev {
		if r == '?' {
			return false
		}
		if r < 33 || r > 126 || r == '?' || r == '*' || r == '>' || r == '.' {
			return false
		}
	}

	return true
}

// Event sends a custom event on the resource.
// Will panic if the event is one of the pre-defined or reserved events,
// "change", "delete", "add", "remove", "patch", "reaccess", or "unsubscribe".
// For pre-defined events, the matching method, ChangeEvent, AddEvent,
// RemoveEvent, or ReaccessEvent should be used instead.
//
// This is to ensure compliance with the specifications:
// https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#events
func (r *resource) Event(event string, payload interface{}) {
	switch event {
	case "change":
		panic("res: use ChangeEvent to send change events")
	case "delete":
		panic(`res: "delete" is a reserved event name`)
	case "add":
		panic("res: use AddEvent to send add events")
	case "remove":
		panic("res: use RemoveEvent to send remove events")
	case "patch":
		panic(`res: "patch" is a reserved event name`)
	case "reaccess":
		panic("res: use ReaccessEvent to send a reaccess event")
	case "unsubscribe":
		panic(`res: "unsubscribe" is a reserved event name`)
	}

	if !isValidEvent(event) {
		panic(`res: invalid event name`)
	}

	r.s.event("event."+r.rname+"."+event, payload)
}

// ChangeEvent sends a change event.
// If ev is empty, no event is sent.
// Panics if the resource is not a Model.
func (r *resource) ChangeEvent(ev map[string]interface{}) {
	if r.hs.typ != rtypeModel {
		panic("res: change event only allowed on Models")
	}
	if len(ev) == 0 {
		return
	}
	r.s.event("event."+r.rname+".change", ev)
}

// AddEvent sends an add event, adding the value v at index idx.
// Panics if the resource is not a Collection.
func (r *resource) AddEvent(v interface{}, idx int) {
	if r.hs.typ != rtypeCollection {
		panic("res: add event only allowed on Collections")
	}
	if idx < 0 {
		panic("res: add event idx less than zero")
	}
	r.s.event("event."+r.rname+".add", addEvent{Value: v, Idx: idx})
}

// RemoveEvent sends an remove event, removing the value at index idx.
// Panics if the resource is not a Collection.
func (r *resource) RemoveEvent(idx int) {
	if r.hs.typ != rtypeCollection {
		panic("res: remove event only allowed on Collections")
	}
	if idx < 0 {
		panic("res: remove event idx less than zero")
	}
	r.s.event("event."+r.rname+".remove", removeEvent{Idx: idx})
}

// ReaccessEvent sends a reaccess event.
func (r *resource) ReaccessEvent() {
	r.s.rawEvent("event."+r.rname+".reaccess", nil)
}
