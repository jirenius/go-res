package restest

import (
	"encoding/json"

	"github.com/jirenius/go-res/store"
)

// Event represents an event.
type Event struct {
	// Name of the event.
	Name string

	// Index position where the resource is added or removed from the query
	// result.
	//
	// Only valid for "add" and "remove" events.
	Idx int

	// ID of resource being added or removed from the query result.
	//
	// Only valid for "add" events.
	Value interface{}

	// Changed property values for the model emitting the event.
	//
	// Only valid for "change" events, and should marshal into a json object
	// with changed key/value properties.
	Changed interface{}

	// Payload of a custom event.
	Payload interface{}
}

// MarshalJSON marshals the event into json.
func (ev Event) MarshalJSON() ([]byte, error) {
	switch ev.Name {
	case "change":
		return json.Marshal(struct {
			Values interface{} `json:"values"`
		}{ev.Changed})
	case "add":
		return json.Marshal(struct {
			Value interface{} `json:"value"`
			Idx   int         `json:"idx"`
		}{ev.Value, ev.Idx})
	case "remove":
		return json.Marshal(struct {
			Idx int `json:"idx"`
		}{ev.Idx})
	case "delete":
		fallthrough
	case "create":
		fallthrough
	case "reaccess":
		return []byte("null"), nil
	default:
		return json.Marshal(ev.Payload)
	}
}

// ToResultEvents creates a slice of store result events from a slice of events.
func ToResultEvents(evs []Event) []store.ResultEvent {
	if evs == nil {
		return nil
	}
	revs := make([]store.ResultEvent, len(evs))

	for i, ev := range evs {
		var changed map[string]interface{}
		if ev.Changed != nil {
			dta, err := json.Marshal(ev.Changed)
			if err != nil {
				panic("failed to marshal changed value to a json object: " + err.Error())
			}
			err = json.Unmarshal(dta, &changed)
			if err != nil {
				panic("failed to unmarshal changed alues to a map[string]interface{}: " + err.Error())
			}
		}
		revs[i] = store.ResultEvent{
			Name:    ev.Name,
			Idx:     ev.Idx,
			Value:   ev.Value,
			Changed: changed,
		}
	}
	return revs
}
