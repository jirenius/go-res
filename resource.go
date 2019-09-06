package res

import (
	"net/url"

	nats "github.com/nats-io/go-nats"
)

// Resource represents a resource
type Resource interface {
	// Service returns the service instance
	Service() *Service

	// Resource returns the resource name.
	ResourceName() string

	// ResourceType returns the resource type.
	ResourceType() ResourceType

	// PathParams returns parameters that are derived from the resource name.
	PathParams() map[string]string

	// PathParam returns the key placeholder parameter value derived from the resource name.
	PathParam(string) string

	// Query returns the query part of the resource ID without the question mark separator.
	Query() string

	// Group which the resource shares worker goroutine with.
	// Will be the resource name of no specific group was set.
	Group() string

	// ParseQuery parses the query and returns the corresponding values.
	// It silently discards malformed value pairs.
	// To check errors use url.ParseQuery(Query()).
	ParseQuery() url.Values

	// Value gets the resource value as provided from the Get resource handlers.
	// If it fails to get the resource value, or no get handler is
	// defined, it returns a nil interface and a *Error type error.
	Value() (interface{}, error)

	// RequireValue gets the resource value as provided from the Get resource handlers.
	// Panics if it fails to get the resource value, or no get handler is defined.
	RequireValue() interface{}

	// Event sends a custom event on the resource.
	// Will panic if the event is one of the pre-defined or reserved events,
	// "change", "delete", "add", "remove", "patch", "reaccess", "unsubscribe", or "query".
	// For pre-defined events, the matching method, ChangeEvent, AddEvent,
	// RemoveEvent, CreateEvent, DeleteEvent, or ReaccessEvent should be used instead.
	//
	// This is to ensure compliance with the specifications:
	// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#events
	Event(event string, payload interface{})

	// ChangeEvents sends a change event with properties that has been changed
	// and their new values.
	// If props is empty, no event is sent.
	// Panics if the resource is not a Model.
	// The values must be serializable into JSON primitives, resource references,
	// or a delete action objects.
	// See the protocol specification for more information:
	//    https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#model-change-event
	ChangeEvent(props map[string]interface{})

	// AddEvent sends an add event, adding the value at index idx.
	// Panics if the resource is not a Collection, or if idx is less than 0.
	// The value must be serializable into a JSON primitive or resource reference.
	// See the protocol specification for more information:
	//    https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#collection-add-event
	AddEvent(value interface{}, idx int)

	// RemoveEvent sends a remove event, removing the value at index idx.
	// Panics if the resource is not a Collection, or if idx is less than 0.
	// See the protocol specification for more information:
	//    https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#collection-remove-event
	RemoveEvent(idx int)

	// ReaccessEvent sends a reaccess event to signal that the resource's access permissions has changed.
	// It will invalidate any previous access response sent for the resource.
	// See the protocol specification for more information:
	//    https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#reaccess-event
	ReaccessEvent()

	// QueryEvent sends a query event to signal that the query resource's underlying data has been modified.
	// See the protocol specification for more information:
	//    https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#query-event
	QueryEvent(func(QueryRequest))

	// CreateEvent sends a create event, to signal the resource has been created, with
	// value being the resource value.
	CreateEvent(value interface{})

	// DeleteEvent sends a delete event, to signal the resource has been deleted.
	DeleteEvent()
}

// resource is the internal implementation of the Resource interface
type resource struct {
	rname      string
	pathParams map[string]string
	query      string
	group      string
	inGet      bool
	h          Handler
	s          *Service
}

// Service returns the service instance
func (r *resource) Service() *Service {
	return r.s
}

// ResourceName returns the resource name.
func (r *resource) ResourceName() string {
	return r.rname
}

// ResourceType returns the resource type.
func (r *resource) ResourceType() ResourceType {
	return r.h.Type
}

// PathParams returns parameters that are derived from the resource name.
func (r *resource) PathParams() map[string]string {
	return r.pathParams
}

// PathParam returns the parameter derived from the resource name for the key placeholder.
func (r *resource) PathParam(key string) string {
	return r.pathParams[key]
}

// Query returns the query part of the resource ID without the question mark separator.
func (r *resource) Query() string {
	return r.query
}

// Group returns the group which the resource shares the worker goroutine with.
// Will be the resource name of no specific group was set.
func (r *resource) Group() string {
	return r.group
}

// ParseQuery parses the query and returns the corresponding values.
// It silently discards malformed value pairs.
// To check errors use url.ParseQuery.
func (r *resource) ParseQuery() url.Values {
	v, _ := url.ParseQuery(r.query)
	return v
}

// Value gets the resource value as provided from the Get resource handlers.
// If it fails to get the resource value, or no get handler is
// defined, it returns a nil interface and a *Error type error.
// Panics if called from within a Get handler.
func (r *resource) Value() (interface{}, error) {
	// Panic if the getRequest is called within Get handler.
	if r.inGet {
		panic("Value() called from within get handler")
	}

	gr := &getRequest{resource: r}
	gr.executeHandler()
	return gr.value, gr.err
}

// RequireValue uses Value to gets the resource value, provided from the Get resource handler.
// It panics if the underlying call to Value return an error.
func (r *resource) RequireValue() interface{} {
	i, err := r.Value()
	if err != nil {
		panic(err)
	}
	return i
}

// Event sends a custom event on the resource.
// Will panic if the event is one of the pre-defined or reserved events,
// "change", "delete", "add", "remove", "patch", "reaccess", "unsubscribe", or "query".
// For pre-defined events, the matching method, ChangeEvent, AddEvent,
// RemoveEvent, CreateEvent, DeleteEvent, or ReaccessEvent should be used instead.
//
// This is to ensure compliance with the specifications:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#events
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
	case "query":
		panic(`res: "query" is a reserved event name`)
	}

	if !isValidPart(event) {
		panic(`res: invalid event name`)
	}

	r.s.event("event."+r.rname+"."+event, payload)
}

// ChangeEvent sends a change event.
// If ev is empty, no event is sent.
// Panics if the resource is not a Model.
func (r *resource) ChangeEvent(ev map[string]interface{}) {
	if r.h.Type == TypeCollection {
		panic("res: change event not allowed on Collections")
	}
	if len(ev) == 0 {
		return
	}
	if r.h.ApplyChange != nil {
		rev, err := r.h.ApplyChange(r, ev)
		if err != nil {
			panic(err)
		}
		if len(rev) == 0 {
			return
		}
	}
	r.s.event("event."+r.rname+".change", changeEvent{Values: ev})
}

// AddEvent sends an add event, adding the value v at index idx.
// Panics if the resource is not a Collection.
func (r *resource) AddEvent(v interface{}, idx int) {
	if r.h.Type == TypeModel {
		panic("res: add event not allowed on Models")
	}
	if idx < 0 {
		panic("res: add event idx less than zero")
	}
	if r.h.ApplyAdd != nil {
		err := r.h.ApplyAdd(r, v, idx)
		if err != nil {
			panic(err)
		}
	}
	r.s.event("event."+r.rname+".add", addEvent{Value: v, Idx: idx})
}

// RemoveEvent sends an remove event, removing the value at index idx.
// Panics if the resource is not a Collection.
func (r *resource) RemoveEvent(idx int) {
	if r.h.Type == TypeModel {
		panic("res: remove event not allowed on Models")
	}
	if idx < 0 {
		panic("res: remove event idx less than zero")
	}
	if r.h.ApplyRemove != nil {
		_, err := r.h.ApplyRemove(r, idx)
		if err != nil {
			panic(err)
		}
	}
	r.s.event("event."+r.rname+".remove", removeEvent{Idx: idx})
}

// ReaccessEvent sends a reaccess event.
func (r *resource) ReaccessEvent() {
	r.s.rawEvent("event."+r.rname+".reaccess", nil)
}

// QueryEvent sends a query event on the resource, calling the
// provided callback on any query request.
// The last call to the callback will always be with nil, indicating
// that the query event duration has expired.
func (r *resource) QueryEvent(cb func(QueryRequest)) {
	qsubj := nats.NewInbox()
	ch := make(chan *nats.Msg, queryEventChannelSize)
	sub, err := r.s.nc.ChanSubscribe(qsubj, ch)
	if err != nil {
		cb(nil)
		r.s.Logf("Failed to subscribe to query event: %s", err)
		return
	}

	qe := &queryEvent{
		r:   *r,
		sub: sub,
		ch:  ch,
		cb:  cb,
	}

	r.s.event("event."+r.rname+".query", resQueryEvent{Subject: qsubj})

	go qe.startQueryListener()

	r.s.queryTQ.Add(qe)
}

// CreateEvent sends a create event for the resource.
func (r *resource) CreateEvent(value interface{}) {
	if r.h.ApplyCreate != nil {
		err := r.h.ApplyCreate(r, value)
		if err != nil {
			panic(err)
		}
	}
	r.s.rawEvent("event."+r.rname+".create", nil)
}

// DeleteEvent sends a delete event.
func (r *resource) DeleteEvent() {
	if r.h.ApplyDelete != nil {
		_, err := r.h.ApplyDelete(r)
		if err != nil {
			panic(err)
		}
	}
	r.s.rawEvent("event."+r.rname+".delete", nil)
}

func isValidPart(p string) bool {
	if p == "" {
		return false
	}
	for _, r := range p {
		if r == '?' {
			return false
		}
		if r < 33 || r > 126 || r == '?' || r == '*' || r == '>' || r == '.' {
			return false
		}
	}
	return true
}
