package restest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

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

// GetParallelMsgs gets n number of published messages where the order is
// uncertain.
func (c *MockConn) GetParallelMsgs(n int) ParallelMsgs {
	msgs := make([]*Msg, n)
	for i := 0; i < n; i++ {
		msgs[i] = c.GetMsg()
	}
	return ParallelMsgs{c: c, msgs: msgs}
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
