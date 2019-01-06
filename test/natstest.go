package test

import (
	"encoding/json"
	"os"
	"reflect"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	nats "github.com/nats-io/go-nats"
)

const timeoutSeconds = 1

// TestConn mocks a client connection to a NATS server.
type TestConn struct {
	closed bool

	reqs chan *Msg
	mu   sync.Mutex
	ch   chan *nats.Msg
	subs map[string]struct{}

	rcount int
}

// Msg represent a message sent to NATS
type Msg struct {
	Subject    string
	RawPayload []byte
	Payload    interface{}
	c          *TestConn
}

// ParallelMsgs holds multiple requests in undetermined order
type ParallelMsgs []*Msg

// NewTestConn creates a new TestConn instance
func NewTestConn() *TestConn {
	return &TestConn{
		subs: make(map[string]struct{}),
		reqs: make(chan *Msg, 256),
	}
}

// Publish publishes the data argument to the given subject
func (c *TestConn) Publish(subj string, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var p interface{}
	if len(payload) > 0 {
		err := json.Unmarshal(payload, &p)
		if err != nil {
			panic("test: error unmarshaling message payload: " + err.Error())
		}
	}

	m := &Msg{
		Subject:    subj,
		RawPayload: payload,
		Payload:    p,
		c:          c,
	}

	c.reqs <- m

	return nil
}

// ChanSubscribe subscribes to messages matching the subject pattern.
func (c *TestConn) ChanSubscribe(subj string, ch chan *nats.Msg) (*nats.Subscription, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ch == nil {
		c.ch = ch
	} else if c.ch != ch {
		panic("test: subscription with different receiving channels")
	}

	if _, ok := c.subs[subj]; ok {
		panic("test: subscription for " + subj + " already exists")
	}

	c.subs[subj] = struct{}{}

	return &nats.Subscription{}, nil
}

// Close will close the connection to the server.
func (c *TestConn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	close(c.reqs)
	c.reqs = nil
	c.closed = true
}

// Request mocks a request from NATS
func (c *TestConn) Request(subj string, payload interface{}) string {
	data, err := json.Marshal(payload)
	if err != nil {
		panic("test: error marshaling request: " + err.Error())
	}
	return c.RequestRaw(subj, data)
}

// RequestRaw mocks a raw byte request from NATS and returns
// the reply inbox used.
func (c *TestConn) RequestRaw(subj string, data []byte) string {
	inbox := nats.NewInbox()
	msg := nats.Msg{
		Subject: subj,
		Reply:   inbox,
		Data:    data,
		Sub:     nil,
	}
	c.ch <- &msg
	return inbox
}

// IsClosed tests if the client connection has been closed.
func (c *TestConn) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// AssertSubscription asserts that the given subjects is subscribed to with the channel
func (c *TestConn) AssertSubscription(t *testing.T, subj string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.subs[subj]
	if !ok {
		t.Fatalf("expected subscription for %#v, but found none", subj)
	}
}

// AssertNoSubscription asserts that there is no subscription for the given subject
func (c *TestConn) AssertNoSubscription(t *testing.T, subj string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.subs[subj]
	if ok {
		t.Fatalf("expected no subscription for %#v, but found one", subj)
	}
}

// GetMsg gets a pending message that is published to NATS.
// If no message is received within a set amount of time,
// it will log it as a fatal error.
func (c *TestConn) GetMsg(t *testing.T) *Msg {
	select {
	case r := <-c.reqs:
		return r
	case <-time.After(timeoutSeconds * time.Second):
		if t == nil {
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			panic("expected a message but found none")
		} else {
			t.Fatal("expected a message but found none")
		}
	}
	return nil
}

// AssertEqual expects that a equals b for the named value,
// and returns true if it is, otherwise logs an error and returns false.
func AssertEqual(t *testing.T, name string, result, expected interface{}) bool {
	aa, aj := jsonMap(t, result)
	bb, bj := jsonMap(t, expected)

	if !reflect.DeepEqual(aa, bb) {
		t.Errorf("expected %s to be:\n%s\nbut got:\n%s", name, bj, aj)
		return false
	}

	return true
}

// AssertNoError expects that err is nil, otherwise logs an error
// with t.Fatalf
func AssertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("expected no error but got:\n%s", err)
	}
}

// Equals asserts that the message has the expected subject and payload
func (m *Msg) Equals(t *testing.T, subject string, payload interface{}) *Msg {
	m.AssertSubject(t, subject)
	m.AssertPayload(t, payload)
	return m
}

// AssertSubject asserts that the message has the expected subject
func (m *Msg) AssertSubject(t *testing.T, subject string) *Msg {
	if m.Subject != subject {
		t.Fatalf("expected subject to be %#v, but got %#v", subject, m.Subject)
	}
	return m
}

// AssertPayload asserts that the message has the expected payload
func (m *Msg) AssertPayload(t *testing.T, payload interface{}) *Msg {
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

	if !reflect.DeepEqual(p, m.Payload) {
		t.Fatalf("expected message payload to be:\n%s\nbut got:\n%s", pj, m.RawPayload)
	}
	return m
}

// AssertResult asserts that the response has the expected result
func (m *Msg) AssertResult(t *testing.T, result interface{}) *Msg {
	m.AssertNoPath(t, "error")
	mr := m.PathPayload(t, "result")

	r, rj := jsonMap(t, result)

	if !reflect.DeepEqual(r, mr) {
		mrj, err := json.Marshal(mr)
		if err != nil {
			panic("test: error marshaling response result: " + err.Error())
		}
		t.Fatalf("expected response result to be:\n%s\nbut got:\n%s", rj, mrj)
	}
	return m
}

func jsonMap(t *testing.T, v interface{}) (interface{}, []byte) {
	var err error
	j, err := json.Marshal(v)
	if err != nil {
		panic("test: error marshaling value: " + err.Error())
	}

	var m interface{}
	err = json.Unmarshal(j, &m)
	if err != nil {
		panic("test: error unmarshaling value: " + err.Error())
	}

	return m, j
}

// AssertError asserts that the response has the expected error
func (m *Msg) AssertError(t *testing.T, rerr *res.Error) *Msg {
	// Assert it is an error
	m.AssertNoPath(t, "result")
	me := m.PathPayload(t, "error")

	e, ej := jsonMap(t, rerr)

	if !reflect.DeepEqual(e, me) {
		mej, err := json.Marshal(me)
		if err != nil {
			panic("test: error marshaling response error: " + err.Error())
		}
		t.Fatalf("expected response error to be:\n%s\nbut got:\n%s", ej, mej)
	}
	return m
}

// AssertErrorCode asserts that the response has the expected error code
func (m *Msg) AssertErrorCode(t *testing.T, code string) *Msg {
	// Assert it is not a successful result
	m.AssertNoPath(t, "result")
	c := m.PathPayload(t, "error.code")

	// Assert the code is a string
	s, ok := c.(string)
	if !ok {
		t.Fatalf("expected error code to be a string, but got type:\n%T", c)
	}

	if s != code {
		t.Fatalf("expected response error code to be:\n%#v\nbut got:\n%#v", code, c)
	}
	return m
}

// AssertPathPayload asserts that a the message payload at a given dot-separated
// path in a nested object has the expected payload.
func (m *Msg) AssertPathPayload(t *testing.T, path string, payload interface{}) *Msg {
	pp := m.PathPayload(t, path)

	var err error
	pj, err := json.Marshal(payload)
	if err != nil {
		panic("test: error marshaling assertion path payload: " + err.Error())
	}
	var p interface{}
	err = json.Unmarshal(pj, &p)
	if err != nil {
		panic("test: error unmarshaling assertion path payload: " + err.Error())
	}

	if !reflect.DeepEqual(p, pp) {
		ppj, err := json.Marshal(pp)
		if err != nil {
			panic("test: error marshaling message path payload: " + err.Error())
		}

		t.Fatalf("expected message payload of path %#v to be:\n%s\nbut got:\n%s", path, pj, ppj)
	}
	return m
}

// AssertPathType asserts that a the message payload at a given dot-separated
// path in a nested object has the same type as typ.
func (m *Msg) AssertPathType(t *testing.T, path string, typ interface{}) *Msg {
	pp := m.PathPayload(t, path)

	ppt := reflect.TypeOf(pp)
	pt := reflect.TypeOf(typ)

	if ppt != pt {
		t.Fatalf("expected message payload of path %#v to be of type \"%s\", but got \"%s\"", path, pt, ppt)
	}
	return m
}

// PathPayload returns the message payload at a given dot-separated path in a nested object.
// It gives a fatal error if the path doesn't exist.
func (m *Msg) PathPayload(t *testing.T, path string) interface{} {
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(m.Payload)
	for _, part := range parts {
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		typ := v.Type()
		if typ.Kind() != reflect.Map {
			t.Fatalf("expected to find path %#v, but part %#v is of type %s\n%#v", path, part, typ, v.Interface())
		}
		if typ.Key().Kind() != reflect.String {
			panic("test: key of part " + part + " of path " + path + " is not of type string")
		}
		v = v.MapIndex(reflect.ValueOf(part))
		if !v.IsValid() {
			t.Fatalf("expected to find path %#v, but missing map key %#v", path, part)
		}
	}

	return v.Interface()
}

// AssertNoPath asserts that a the message payload doesn't have a value at a given
// dot-separated path in a nested object.
func (m *Msg) AssertNoPath(t *testing.T, path string) *Msg {
	if m.Payload == nil {
		return m
	}
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(m.Payload)
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

	t.Fatalf("expected not to find path %#v, but found the value:\n%#v", path, v.Interface())
	return m
}

// GetParallelMsgs gets n number of published messages where the order is uncertain.
func (c *TestConn) GetParallelMsgs(t *testing.T, n int) ParallelMsgs {
	pm := make(ParallelMsgs, n)
	for i := 0; i < n; i++ {
		pm[i] = c.GetMsg(t)
	}
	return pm
}

// GetMsg returns a published message based on subject.
func (pm ParallelMsgs) GetMsg(t *testing.T, subject string) *Msg {
	for _, m := range pm {
		if m.Subject == subject {
			return m
		}
	}

	t.Fatalf("expected parallel messages to contain subject %#v, but found none", subject)
	return nil
}
