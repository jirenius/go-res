package res

import "net/url"

// Resource represents a resource
type Resource struct {
	// Resource model or collection as returned from the get handler.
	// Value is nil on get or access request, or if no GetModel or GetCollection handler is set.
	Value interface{}

	// Resource name. The name is the resource ID without the query.
	ResourceName string

	// Path parameters parsed from the resource name
	PathParams map[string]string

	// RawQuery part of the resource ID without the question mark separator.
	RawQuery string `json:"query"`

	s *Service
	h *Handlers
}

// Query parses RawQuery and returns the corresponding values.
// It silently discards malformed value pairs.
// To check errors use url.ParseQuery.
func (r *Resource) Query() url.Values {
	v, _ := url.ParseQuery(r.RawQuery)
	return v
}

// Event sends a custom event on the resource.
// Will panic if the event is one of the pre-defined or reserved events,
// "change", "add", "remove", "reaccess", or "unsubscribe".
// For pre-defined events, the matching method, ChangeEvent, AddEvent,
// RemoveEvent, or ReaccessEvent should be used instead.
//
// This is to ensure compliance with the specifications:
// https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#events
func (r *Resource) Event(event string, payload interface{}) {
	switch event {
	case "change":
		panic("res: use ChangeEvent to send change events")
	case "add":
		panic("res: use AddEvent to send add events")
	case "remove":
		panic("res: use RemoveEvent to send remove events")
	case "reaccess":
		panic("res: use ReaccessEvent to send a reaccess event")
	case "unsubscribe":
		panic(`res: "unsubscribe" is a reserved event name`)
	}

	r.s.send("event."+r.ResourceName+"."+event, payload)
}

// ChangeEvent sends a change event.
// If ev is empty, no event is sent.
// Panics if the resource is not a Model.
func (r *Resource) ChangeEvent(ev map[string]interface{}) {
	if r.h.typ != rtypeModel {
		panic("res: change event only allowed on Models")
	}
	if len(ev) == 0 {
		return
	}
	r.s.send("event."+r.ResourceName+".change", ev)
}

// AddEvent sends an add event, adding the value v at index idx.
// Panics if the resource is not a Collection.
func (r *Resource) AddEvent(v interface{}, idx int) {
	if r.h.typ != rtypeCollection {
		panic("res: add event only allowed on Collections")
	}
	r.s.send("event."+r.ResourceName+".add", addEvent{Value: v, Idx: idx})
}

// RemoveEvent sends an remove event, removing the value at index idx.
// Panics if the resource is not a Collection.
func (r *Resource) RemoveEvent(idx int) {
	if r.h.typ != rtypeCollection {
		panic("res: remove event only allowed on Collections")
	}
	r.s.send("event."+r.ResourceName+".remove", removeEvent{Idx: idx})
}

// ReaccessEvent sends an reaccess event.
func (r *Resource) ReaccessEvent() {
	r.s.send("event."+r.ResourceName+".reaccess", nil)
}
