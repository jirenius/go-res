package restest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/nats-io/nats-server/v2/server"
	ntest "github.com/nats-io/nats-server/v2/test"
	nats "github.com/nats-io/nats.go"
)

// MockConn mocks a client connection to a NATS server.
type MockConn struct {
	t          *testing.T
	cfg        MockConnConfig
	mu         sync.Mutex
	subs       map[*nats.Subscription]*mockSubscription
	subStrings map[string]struct{}
	rch        chan *nats.Msg

	// Mock server fields
	closed               bool
	failNextSubscription bool

	// Real server fields
	gnatsd *server.Server
	nc     *nats.Conn // nats connection for service
	rc     *nats.Conn // nats connection for Resgate
}

// MockConnConfig holds MockConn configuration.
type MockConnConfig struct {
	UseGnatsd       bool
	TimeoutDuration time.Duration
}

// mockSubscription mocks a subscription made to NATS server.
type mockSubscription struct {
	c       *MockConn
	subject string
	parts   []string
	isFWC   bool
	ch      chan *nats.Msg
}

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

var gnatsdMutex sync.Mutex
var testOptions = server.Options{
	Host:   "localhost",
	Port:   4300,
	NoLog:  true,
	NoSigs: true,
}

// NewMockConn creates a new MockConn.
func NewMockConn(t *testing.T, cfg *MockConnConfig) *MockConn {
	if cfg != nil {
		cfg = &MockConnConfig{TimeoutDuration: DefaultTimeoutDuration}
	}
	if !cfg.UseGnatsd {
		// Use a fake server for speeds sake
		return &MockConn{
			t:          t,
			cfg:        *cfg,
			subs:       make(map[*nats.Subscription]*mockSubscription),
			subStrings: make(map[string]struct{}),
			rch:        make(chan *nats.Msg, 256),
		}
	}

	gnatsdMutex.Lock()
	defer gnatsdMutex.Unlock()
	opts := testOptions
	testOptions.Port = testOptions.Port%100 + 4301

	// Set up a real gnatsd server
	gnatsd := ntest.RunServer(&opts)
	if gnatsd == nil {
		panic("Could not start GNATS queue server")
	}

	nc, err := nats.Connect(fmt.Sprintf("nats://%s:%d", opts.Host, opts.Port))
	if err != nil {
		panic(err)
	}
	rc, err := nats.Connect(fmt.Sprintf("nats://%s:%d", opts.Host, opts.Port), nats.NoEcho())
	if err != nil {
		panic(err)
	}
	rch := make(chan *nats.Msg, 256)
	_, err = rc.ChanSubscribe(">", rch)
	if err != nil {
		panic(err)
	}
	err = rc.Flush()
	if err != nil {
		panic(err)
	}
	return &MockConn{
		t:          t,
		cfg:        *cfg,
		gnatsd:     gnatsd,
		nc:         nc,
		rc:         rc,
		rch:        rch,
		subStrings: make(map[string]struct{}),
	}
}

// StopServer stops the gnatsd server.
func (c *MockConn) StopServer() {
	if c.cfg.UseGnatsd {
		c.gnatsd.Shutdown()
		c.gnatsd = nil
	}
}

// Publish publishes the data argument to the given subject.
func (c *MockConn) Publish(subj string, payload []byte) error {
	if c.cfg.UseGnatsd {
		return c.nc.Publish(subj, payload)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	msg := &nats.Msg{
		Subject: subj,
		Data:    payload,
	}

	c.rch <- msg

	return nil
}

// ChanSubscribe subscribes to messages matching the subject pattern.
func (c *MockConn) ChanSubscribe(subj string, ch chan *nats.Msg) (*nats.Subscription, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failNextSubscription {
		c.failNextSubscription = false
		return nil, errors.New("test: failing subscription as requested")
	}

	if _, ok := c.subStrings[subj]; ok {
		panic("test: subscription for " + subj + " already exists")
	}

	c.subStrings[subj] = struct{}{}

	if c.cfg.UseGnatsd {
		return c.nc.ChanSubscribe(subj, ch)
	}

	sub := &nats.Subscription{}
	msub := newMockSubscription(c, subj, ch)
	c.subs[sub] = msub

	return sub, nil
}

// Close will close the connection to the server.
func (c *MockConn) Close() {
	if c.cfg.UseGnatsd {
		c.nc.Close()
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	close(c.rch)
	c.rch = nil
	c.closed = true
}

// Request mocks a request from NATS and returns a NATSRequest.
func (c *MockConn) Request(subj string, payload interface{}) *NATSRequest {
	data, err := json.Marshal(payload)
	if err != nil {
		panic("test: error marshaling request: " + err.Error())
	}
	return &NATSRequest{
		c:   c,
		inb: c.RequestRaw(subj, data),
	}
}

// RequestRaw mocks a raw byte request from NATS and returns the reply inbox
// used.
func (c *MockConn) RequestRaw(subj string, data []byte) string {
	if c.cfg.UseGnatsd {
		inbox := c.rc.NewRespInbox()
		err := c.rc.PublishRequest(subj, inbox, data)
		if err != nil {
			panic("test: error sending request: " + err.Error())
		}
		return inbox
	}

	inbox := nats.NewInbox()
	msg := nats.Msg{
		Subject: subj,
		Reply:   inbox,
		Data:    data,
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, msub := range c.subs {
		if msub.matches(subj) {
			msub.ch <- &msg
		}
	}
	return inbox
}

// QueryRequest mocks a query request from NATS and returns a NATSRequest.
func (c *MockConn) QueryRequest(querySubj string, query string) *NATSRequest {
	return c.Request(querySubj, struct {
		Query string `json:"query"`
	}{query})
}

// IsClosed tests if the client connection has been closed.
func (c *MockConn) IsClosed() bool {
	if c.cfg.UseGnatsd {
		return c.nc.IsClosed()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// AssertSubscription asserts that the given subjects is subscribed to with the
// channel.
func (c *MockConn) AssertSubscription(subj string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.subStrings[subj]
	if !ok {
		c.t.Fatalf("expected subscription for %#v, but found none", subj)
	}
}

// AssertNoSubscription asserts that there is no subscription for the given
// subject.
func (c *MockConn) AssertNoSubscription(subj string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.subStrings[subj]
	if ok {
		c.t.Fatalf("expected no subscription for %#v, but found one", subj)
	}
}

// GetMsg gets a pending message that is published to NATS.
//
// If no message is received within a set amount of time, it will log it as a
// fatal error.
func (c *MockConn) GetMsg() *Msg {
	select {
	case r := <-c.rch:
		return &Msg{
			Msg: r,
			c:   c,
		}
	case <-time.After(c.cfg.TimeoutDuration):
		if c.t == nil {
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			panic("expected a message but found none")
		} else {
			c.t.Fatal("expected a message but found none")
		}
	}
	return nil
}

// FailNextSubscription flags that the next subscription attempt should fail.
func (c *MockConn) FailNextSubscription() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failNextSubscription = true
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

// AssertEvent asserts that the message is an event for the given resource ID.
func (m *Msg) AssertEvent(rid string, event string) *Msg {
	m.AssertSubject("event." + rid + "." + event)
	return m
}

// AssertQueryEvent asserts that the message is a query event for the given
// resource ID, setting the event subject.
//
// The subject may be nil, and will then not be set.
func (m *Msg) AssertQueryEvent(rid string, subject *string) *Msg {
	m.AssertEvent(rid, "query")
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
	m.AssertEvent(rid, "change")
	m.AssertPayload(struct {
		Values interface{} `json:"values"`
	}{values})
	return m
}

// AssertAddEvent asserts that the message is an add event for the given
// resource ID, matching value and idx.
func (m *Msg) AssertAddEvent(rid string, value interface{}, idx int) *Msg {
	m.AssertEvent(rid, "add")
	m.AssertPayload(struct {
		Value interface{} `json:"value"`
		Idx   int         `json:"idx"`
	}{value, idx})
	return m
}

// AssertRemoveEvent asserts that the message is a remove event for the given
// resource ID, matching idx.
func (m *Msg) AssertRemoveEvent(rid string, idx int) *Msg {
	m.AssertEvent(rid, "remove")
	m.AssertPayload(struct {
		Idx int `json:"idx"`
	}{idx})
	return m
}

// AssertReaccessEvent asserts that the message is a reaccess event for the given
// resource ID, with no payload.
func (m *Msg) AssertReaccessEvent(rid string) *Msg {
	m.AssertEvent(rid, "reaccess")
	m.AssertPayload(nil)
	return m
}

// AssertCreateEvent asserts that the message is a create event for the given
// resource ID, with no payload.
func (m *Msg) AssertCreateEvent(rid string) *Msg {
	m.AssertEvent(rid, "create")
	m.AssertPayload(nil)
	return m
}

// AssertDeleteEvent asserts that the message is a delete event for the given
// resource ID, with no payload.
func (m *Msg) AssertDeleteEvent(rid string) *Msg {
	m.AssertEvent(rid, "delete")
	m.AssertPayload(nil)
	return m
}

// AssertCustomEvent asserts that the message is a custom event for the given
// resource ID, with matching payload.
func (m *Msg) AssertCustomEvent(rid string, event string, payload interface{}) *Msg {
	m.AssertEvent(rid, event)
	m.AssertPayload(payload)
	return m
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

// GetParallelMsgs gets n number of published messages where the order is
// uncertain.
func (c *MockConn) GetParallelMsgs(n int) ParallelMsgs {
	msgs := make([]*Msg, n)
	for i := 0; i < n; i++ {
		msgs[i] = c.GetMsg()
	}
	return ParallelMsgs{c: c, msgs: msgs}
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

func newMockSubscription(c *MockConn, subject string, ch chan *nats.Msg) *mockSubscription {
	s := &mockSubscription{c: c, subject: subject, ch: ch}

	s.isFWC = subject == ">" || strings.HasSuffix(subject, ".>")
	s.parts = strings.Split(subject, ".")
	if s.isFWC {
		s.parts = s.parts[:len(s.parts)-1]
	}
	return s
}

func (s *mockSubscription) matches(subj string) bool {
	mparts := strings.Split(subj, ".")
	if len(mparts) < len(s.parts) || (!s.isFWC && len(mparts) != len(s.parts)) {
		return false
	}
	for i := 0; i < len(s.parts); i++ {
		if s.parts[i] != mparts[i] && s.parts[i] != "*" {
			return false
		}
	}
	return true
}
