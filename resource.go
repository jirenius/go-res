package res

import (
	"net/url"
)

// Resource represents a resource
type Resource struct {
	rname      string
	pathParams map[string]string
	query      string
	s          *Service
	hs         *Handlers
}

// Service returns the service instance
func (r *Resource) Service() *Service {
	return r.s
}

// ResourceName returns the resource name.
func (r *Resource) ResourceName() string {
	return r.rname
}

// PathParams returns parameters that are derived from the resource name.
func (r *Resource) PathParams() map[string]string {
	return r.pathParams
}

// Query returns the query part of the resource ID without the question mark separator.
func (r *Resource) Query() string {
	return r.query
}

// ParseQuery parses the query and returns the corresponding values.
// It silently discards malformed value pairs.
// To check errors use url.ParseQuery.
func (r *Resource) ParseQuery() url.Values {
	v, _ := url.ParseQuery(r.query)
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

	r.s.event("event."+r.rname+"."+event, payload)
}

// ChangeEvent sends a change event.
// If ev is empty, no event is sent.
// Panics if the resource is not a Model.
func (r *Resource) ChangeEvent(ev map[string]interface{}) {
	if r.hs.rtype != rtypeModel {
		panic("res: change event only allowed on Models")
	}
	if len(ev) == 0 {
		return
	}
	r.s.event("event."+r.rname+".change", ev)
}

// AddEvent sends an add event, adding the value v at index idx.
// Panics if the resource is not a Collection.
func (r *Resource) AddEvent(v interface{}, idx int) {
	if r.hs.rtype != rtypeCollection {
		panic("res: add event only allowed on Collections")
	}
	r.s.event("event."+r.rname+".add", addEvent{Value: v, Idx: idx})
}

// RemoveEvent sends an remove event, removing the value at index idx.
// Panics if the resource is not a Collection.
func (r *Resource) RemoveEvent(idx int) {
	if r.hs.rtype != rtypeCollection {
		panic("res: remove event only allowed on Collections")
	}
	r.s.event("event."+r.rname+".remove", removeEvent{Idx: idx})
}

// ReaccessEvent sends a reaccess event.
func (r *Resource) ReaccessEvent() {
	r.s.rawEvent("event."+r.rname+".reaccess", nil)
}
