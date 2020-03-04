package restest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	res "github.com/jirenius/go-res"
	nats "github.com/nats-io/nats.go"
)

// Msg represent a message sent to NATS.
type Msg struct {
	*nats.Msg
	c *MockConn
}

// ParallelMsgs holds multiple requests in undetermined order.
type ParallelMsgs struct {
	c    *MockConn
	msgs []*Msg
}

// Equals asserts that the message has the expected subject and payload.
func (m *Msg) Equals(subject string, payload interface{}) *Msg {
	m.AssertSubject(subject)
	m.AssertPayload(payload)
	return m
}

// Payload unmarshals the message data into an empty interface.
// Panics if the data is not valid json.
func (m *Msg) Payload() interface{} {
	var p interface{}
	if len(m.Data) > 0 {
		err := json.Unmarshal(m.Data, &p)
		if err != nil {
			panic("test: error unmarshaling msg data: " + err.Error())
		}
	}
	return p
}

// AssertSubject asserts that the message has the expected subject.
func (m *Msg) AssertSubject(subject string) *Msg {
	AssertEqualJSON(m.c.t, "subject", subject, m.Subject)
	return m
}

// AssertPayload asserts that the message has the expected payload.
func (m *Msg) AssertPayload(payload interface{}) *Msg {
	var err error
	pj, err := json.Marshal(payload)
	if err != nil {
		panic("test: error marshaling assertion payload: " + err.Error())
	}

	var p interface{}
	err = json.Unmarshal(pj, &p)
	if err != nil {
		panic("test: error unmarshaling assertion payload: " + err.Error())
	}

	if !reflect.DeepEqual(p, m.Payload()) {
		m.c.t.Fatalf("expected message payload to be:\n%s\nbut got:\n%s", pj, m.Data)
	}
	return m
}

// AssertRawPayload asserts that the message has the expected payload bytes.
func (m *Msg) AssertRawPayload(payload []byte) *Msg {
	if !bytes.Equal(payload, m.Data) {
		m.c.t.Fatalf("expected message payload to be:\n%s\nbut got:\n%s", payload, m.Data)
	}
	return m
}

// AssertResult asserts that the response has the expected result.
//
// Only valid for call and auth requests.
func (m *Msg) AssertResult(result interface{}) *Msg {
	m.AssertNoPath("error")
	mr := m.PathPayload("result")
	AssertEqualJSON(m.c.t, "response result", mr, result)
	return m
}

// AssertResource asserts that the response is a resource response matching rid.
//
// Only valid for call and auth requests.
func (m *Msg) AssertResource(rid string) *Msg {
	m.AssertNoPath("error")
	m.AssertNoPath("result")
	mr := m.PathPayload("resource")
	AssertEqualJSON(m.c.t, "response resource", mr, res.Ref(rid))
	return m
}

// AssertError asserts that the response has the expected error.
func (m *Msg) AssertError(rerr *res.Error) *Msg {
	// Assert it is an error
	m.AssertNoPath("result")
	me := m.PathPayload("error")
	AssertEqualJSON(m.c.t, "response error", me, rerr)
	return m
}

// AssertErrorCode asserts that the response has the expected error code.
func (m *Msg) AssertErrorCode(code string) *Msg {
	// Assert it is not a successful result
	m.AssertNoPath("result")
	c := m.PathPayload("error.code")

	// Assert the code is a string
	s, ok := c.(string)
	if !ok {
		m.c.t.Fatalf("expected error code to be a string, but got type:\n%T", c)
	}

	if s != code {
		m.c.t.Fatalf("expected response error code to be:\n%#v\nbut got:\n%#v", code, c)
	}
	return m
}

// AssertPathPayload asserts that a the message payload at a given dot-separated
// path in a nested object has the expected payload.
func (m *Msg) AssertPathPayload(path string, payload interface{}) *Msg {
	pp := m.PathPayload(path)
	AssertEqualJSON(m.c.t, fmt.Sprintf("message payload of path  %#v", path), pp, payload)
	return m
}

// AssertPathType asserts that a the message payload at a given dot-separated
// path in a nested object has the same type as typ.
func (m *Msg) AssertPathType(path string, typ interface{}) *Msg {
	pp := m.PathPayload(path)

	ppt := reflect.TypeOf(pp)
	pt := reflect.TypeOf(typ)

	if ppt != pt {
		m.c.t.Fatalf("expected message payload of path %#v to be of type \"%s\", but got \"%s\"", path, pt, ppt)
	}
	return m
}

// AssertModel asserts that a the result is a model response.
//
// Only valid for get requests and query requests.
func (m *Msg) AssertModel(model interface{}) *Msg {
	m.AssertNoPath("error")
	mr := m.PathPayload("result.model")
	AssertEqualJSON(m.c.t, "result model", mr, model)
	return m
}

// AssertCollection asserts that a the result is a collection response.
//
// Only valid for get requests and query requests.
func (m *Msg) AssertCollection(collection interface{}) *Msg {
	m.AssertNoPath("error")
	mr := m.PathPayload("result.collection")
	AssertEqualJSON(m.c.t, "result collection", mr, collection)
	return m
}

// AssertEvents asserts that a the result is an events response.
//
// Only valid for query requests.
func (m *Msg) AssertEvents(events ...Event) *Msg {
	m.AssertNoPath("error")
	_ = m.PathPayload("result")
	m.AssertNoPath("result.collection")
	m.AssertNoPath("result.model")
	var evs []interface{}
	// It is valid not to have events property set
	if pp, ok := m.HasPath("result.events"); ok {
		if pp != nil {
			// If set, it must be an array of items
			if evs, ok = pp.([]interface{}); !ok {
				m.c.t.Fatalf("expected message result events to be an array of events, but got %#v", pp)
			}
		}
	}
	// Quick exit on no events
	if len(evs) == 0 && len(events) == 0 {
		return m
	}
	type eventItem struct {
		Event string `json:"event"`
		Data  Event  `json:"data"`
	}
	evItems := make([]eventItem, len(events))
	for i, ev := range events {
		evItems[i] = eventItem{ev.Name, ev}
	}
	AssertEqualJSON(m.c.t, "result events", evs, evItems)
	return m
}

// AssertQuery asserts that the result query matches query.
//
// Only valid for get requests.
func (m *Msg) AssertQuery(query string) *Msg {
	m.AssertNoPath("error")
	mr := m.PathPayload("result.query")
	AssertEqualJSON(m.c.t, "result query", mr, query)
	return m
}

// AssertAccess asserts that the access response matches the get and call values.
//
// Only valid for access requests.
func (m *Msg) AssertAccess(get bool, call string) *Msg {
	m.AssertNoPath("error")
	r, ok := m.PathPayload("result").(map[string]interface{})
	if !ok {
		m.c.t.Fatalf("expected result payload to be an object with access values")
	}
	var v interface{}
	// Assert get value
	if v, ok = r["get"]; !ok {
		v = false
	}
	AssertEqualJSON(m.c.t, "get value", v, get)
	// Assert call value
	if v, ok = r["call"]; !ok {
		v = ""
	}
	AssertEqualJSON(m.c.t, "call value", v, call)
	return m
}

// AssertEventName asserts that the message is an event matching the name for
// the given resource ID.
func (m *Msg) AssertEventName(rid string, name string) *Msg {
	m.AssertSubject("event." + rid + "." + name)
	return m
}

// AssertEvent asserts that the message is a matching event for the given
// resource ID.
func (m *Msg) AssertEvent(rid string, ev Event) *Msg {
	m.AssertEventName(rid, ev.Name)
	m.AssertPayload(ev)
	return m
}

// AssertQueryEvent asserts that the message is a query event for the given
// resource ID, setting the event subject.
//
// The subject may be nil, and will then not be set.
func (m *Msg) AssertQueryEvent(rid string, subject *string) *Msg {
	m.AssertEventName(rid, "query")
	s, ok := m.PathPayload("subject").(string)
	if !ok {
		m.c.t.Errorf("expected query event payload contain subject string, but got %#v", m.Payload())
	}
	if subject != nil {
		*subject = s
	}
	return m
}

// AssertChangeEvent asserts that the message is a change event for the given
// resource ID, matching the change values.
//
// The values parameter should marshal into a JSON object.
func (m *Msg) AssertChangeEvent(rid string, values interface{}) *Msg {
	return m.AssertEvent(rid, Event{Name: "change", Changed: values})
}

// AssertAddEvent asserts that the message is an add event for the given
// resource ID, matching value and idx.
func (m *Msg) AssertAddEvent(rid string, value interface{}, idx int) *Msg {
	return m.AssertEvent(rid, Event{Name: "add", Value: value, Idx: idx})
}

// AssertRemoveEvent asserts that the message is a remove event for the given
// resource ID, matching idx.
func (m *Msg) AssertRemoveEvent(rid string, idx int) *Msg {
	return m.AssertEvent(rid, Event{Name: "remove", Idx: idx})
}

// AssertReaccessEvent asserts that the message is a reaccess event for the given
// resource ID, with no payload.
func (m *Msg) AssertReaccessEvent(rid string) *Msg {
	return m.AssertEvent(rid, Event{Name: "reaccess"})
}

// AssertCreateEvent asserts that the message is a create event for the given
// resource ID, with no payload.
func (m *Msg) AssertCreateEvent(rid string) *Msg {
	return m.AssertEvent(rid, Event{Name: "create"})
}

// AssertDeleteEvent asserts that the message is a delete event for the given
// resource ID, with no payload.
func (m *Msg) AssertDeleteEvent(rid string) *Msg {
	return m.AssertEvent(rid, Event{Name: "delete"})
}

// AssertCustomEvent asserts that the message is a custom event for the given
// resource ID, with matching payload.
func (m *Msg) AssertCustomEvent(rid string, event string, payload interface{}) *Msg {
	return m.AssertEvent(rid, Event{Name: event, Payload: payload})
}

// AssertTokenEvent asserts that the message is a connection token event for the given
// connection ID, cid, with matching token.
func (m *Msg) AssertTokenEvent(cid string, token interface{}) *Msg {
	m.AssertSubject("conn." + cid + ".token")
	m.AssertPayload(struct {
		Token interface{} `json:"token"`
	}{token})
	return m
}

// AssertSystemReset asserts that the message is a system reset event, matching
// the resources and access.
func (m *Msg) AssertSystemReset(resources []string, access []string) *Msg {
	m.AssertSubject("system.reset")
	m.AssertPayload(struct {
		Resources []string `json:"resources,omitempty"`
		Access    []string `json:"access,omitempty"`
	}{resources, access})
	return m
}

// PathPayload returns the message payload at a given dot-separated path in a
// nested object. It gives a fatal error if the path doesn't exist.
func (m *Msg) PathPayload(path string) interface{} {
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(m.Payload())
	for _, part := range parts {
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		typ := v.Type()
		if typ.Kind() != reflect.Map {
			m.c.t.Fatalf("expected to find path %#v, but part %#v is of type %s\n%#v", path, part, typ, v.Interface())
		}
		if typ.Key().Kind() != reflect.String {
			panic("test: key of part " + part + " of path " + path + " is not of type string")
		}
		v = v.MapIndex(reflect.ValueOf(part))
		if !v.IsValid() {
			m.c.t.Fatalf("expected to find path %#v, but missing map key %#v", path, part)
		}
	}

	return v.Interface()
}

// HasPath checks if a a given dot-separated path in a nested object exists. If
// it does, it returns the path value and true, otherwise nil and false.
func (m *Msg) HasPath(path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(m.Payload())
	for _, part := range parts {
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		typ := v.Type()
		if typ.Kind() != reflect.Map {
			return nil, false
		}
		if typ.Key().Kind() != reflect.String {
			panic("test: key of part " + part + " of path " + path + " is not of type string")
		}
		v = v.MapIndex(reflect.ValueOf(part))
		if !v.IsValid() {
			return nil, false
		}
	}

	return v.Interface(), true
}

// AssertNoPath asserts that a the message payload doesn't have a value at a
// given dot-separated path in a nested object.
func (m *Msg) AssertNoPath(path string) *Msg {
	if m.Payload() == nil {
		return m
	}
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(m.Payload())
	for _, part := range parts {
		typ := v.Type()
		if typ.Kind() != reflect.Map {
			return m
		}
		if typ.Key().Kind() != reflect.String {
			panic("test: key of part " + part + " of path " + path + " is not of type string")
		}
		v = v.MapIndex(reflect.ValueOf(part))
		if !v.IsValid() {
			return m
		}
	}

	m.c.t.Fatalf("expected not to find path %#v, but found the value:\n%#v", path, v.Interface())
	return m
}

// GetMsg returns a published message based on subject.
func (pm ParallelMsgs) GetMsg(subject string) *Msg {
	for _, m := range pm.msgs {
		if m.Subject == subject {
			return m
		}
	}

	pm.c.t.Fatalf("expected parallel messages to contain subject %#v, but found none", subject)
	return nil
}
