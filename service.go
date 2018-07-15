package res

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/jirenius/resgate/resourceCache"
	nats "github.com/nats-io/go-nats"
)

// The size of the in channel receiving messages from NATS Server.
const inChannelSize = 256

// The number of default workers handling resource requests.
const workerCount = 10

// Debugging flag set by SetDebug
var debug = false

// Handler is a function for the handlers of a resource
type Handler func(*Handlers)

// AccessHandler is a function called on resource access requests
type AccessHandler func(*Request, *AccessResponse)

// GetHandler is a function called on resource get requests
type GetHandler func(*Request, *GetResponse)

// CallHandler is a function called on resource call requests
type CallHandler func(*Request, *CallResponse)

// AuthHandler is a function called on resource auth requests
type AuthHandler func(*Request, *AuthResponse)

// Handlers contains handlers for a given resource pattern.
type Handlers struct {
	Access AccessHandler
	Get    GetHandler
	Call   map[string]CallHandler
	Auth   map[string]AuthHandler
}

// A Service handles incoming requests from NATS Server and calls the
// appropriate callback on the resource handlers.
type Service struct {
	// Name of the service.
	// Must be a non-empty alphanumeric string with no embedded whitespace.
	Name string

	nc             *nats.Conn                    // NATS Server connection
	subs           map[string]*nats.Subscription // Request type nats subscriptions
	patterns       patterns                      // pattern store with all handlers
	rwork          map[string]*work              // map of resource work
	inCh           chan *nats.Msg                // Channel for incoming nats messages
	workCh         chan *work                    // Resource work channel, listened to by the workers
	stopped        chan struct{}                 // Channel that is closed by the listen worker, on stop.
	logger         *log.Logger                   // Logger
	mu             sync.Mutex                    // Mutex to protect rs map
	withAccess     bool                          // Flag that is true if there are patterns with Access handlers
	resetResources []string                      // List of resource name patterns used on system.reset for resources. Defaults to serviceName+">"
	resetAccess    []string                      // List of resource name patterns used system.reset for access. Defaults to serviceName+">"
}

// NewService creates a new Service given a service name.
// The name must be a non-empty alphanumeric string with no embedded whitespace.
func NewService(name string) *Service {
	logFlags := log.LstdFlags
	if debug {
		logFlags = log.Ltime
	}

	// [TODO] panic on invalid name
	return &Service{
		Name:           name,
		patterns:       patterns{root: &node{}},
		logger:         log.New(os.Stdout, "[Service] ", logFlags),
		resetResources: []string{name + ".>"},
		resetAccess:    []string{name + ".>"},
	}
}

// SetDebug enables debug logging
func SetDebug(enabled bool) {
	debug = enabled
	resourceCache.SetDebug(enabled)
}

// Log writes a log message
func (s *Service) Log(v ...interface{}) {
	s.logger.Print(v...)
}

// Logf writes a formatted log message
func (s *Service) Logf(format string, v ...interface{}) {
	s.logger.Printf(format, v...)
}

// Handle registers the handlers for the given resource pattern.
//
// A pattern may contain placeholders that acts as wildcards, and will be
// parsed and stored in the request.PathParams map.
// A placeholder is a resource name part starting with a dollar ($) character:
//  s.Handle("user.$id", handlers) // Will match "user.10", "user.foo", etc.
//
// If the pattern is already registered, or if there are conflicts among
// the handlers, Handle panics.
func (s *Service) Handle(pattern string, handlers ...Handler) {
	var hs Handlers
	for _, h := range handlers {
		h(&hs)
	}
	if hs.Access != nil {
		s.withAccess = true
	}
	s.patterns.add(pattern, &hs)
}

// Access is a handler for resource access requests
func Access(h AccessHandler) Handler {
	return func(hs *Handlers) {
		if hs.Access != nil {
			panic("res: multiple access handlers")
		}
		hs.Access = h
	}
}

// Get is a handler for resource get requests
func Get(h GetHandler) Handler {
	return func(hs *Handlers) {
		if hs.Get != nil {
			panic("res: multiple get handlers")
		}
		hs.Get = h
	}
}

// Call is a handler for resource call requests
func Call(method string, h CallHandler) Handler {
	return func(hs *Handlers) {
		if hs.Call == nil {
			hs.Call = make(map[string]CallHandler)
		}
		if _, ok := hs.Call[method]; ok {
			panic("res: multiple call handlers for method " + method)
		}
		hs.Call[method] = h
	}
}

// Auth is a handler for resource auth requests
func Auth(method string, h AuthHandler) Handler {
	return func(hs *Handlers) {
		if hs.Auth == nil {
			hs.Auth = make(map[string]AuthHandler)
		}
		if _, ok := hs.Auth[method]; ok {
			panic("res: multiple auth handlers for method " + method)
		}
		hs.Auth[method] = h
	}
}

// SetReset sets the patterns used for resources and access when a reset is made.¨
// For more details on system reset, see:
// https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#system-reset-event
func (s *Service) SetReset(resources, access []string) {
	s.resetResources = resources
	s.resetAccess = access
}

// Start connects to the NATS Server and subscribes to all handled resources.
func (s *Service) Start(natsURL string) error {
	if s.nc != nil {
		return errors.New("res: service already started")
	}

	s.Logf("Starting service: %s", s.Name)
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		s.Logf("Failed to connect to NATS server: %s", err)
		return err
	}
	nc.SetReconnectHandler(s.handleReconnect)

	stopped := make(chan struct{})
	s.nc = nc
	s.inCh = make(chan *nats.Msg, inChannelSize)
	s.rwork = make(map[string]*work)
	s.workCh = make(chan *work)
	s.stopped = stopped
	go s.startListener(s.inCh, s.stopped)
	for i := 0; i < workerCount; i++ {
		go s.startWorker(s.workCh)
	}

	err = s.subscribe()
	if err != nil {
		s.Stop()
		return err
	}

	s.Logf("Listening for requests")

	// Always start with a reset
	s.Reset()

	<-stopped
	return nil
}

// subscribe makes a nats subscription for each required request type.
func (s *Service) subscribe() error {
	s.subs = make(map[string]*nats.Subscription, 4)
	for _, t := range []string{RequestTypeAccess, RequestTypeGet, RequestTypeCall, RequestTypeAuth} {
		if t == RequestTypeAccess && !s.withAccess {
			continue
		}
		sub, err := s.nc.ChanSubscribe(t+"."+s.Name+".>", s.inCh)
		if err != nil {
			return err
		}
		s.subs[t] = sub
	}
	return nil
}

// Stop closes any existing connection to NATS Server.
func (s *Service) Stop() {
	if s.nc == nil {
		return
	}

	s.Log("Stopping service...")

	if !s.nc.IsClosed() {
		s.nc.Close()
	}
	close(s.inCh)
	stopped := s.stopped
	s.stopped = nil
	s.inCh = nil
	s.nc = nil
	s.subs = nil

	<-stopped

	s.Log("Stopped")
}

// Reset will send a system.reset to trigger any gateway to update their cache
func (s *Service) Reset() {
	if s.nc == nil {
		s.Logf("failed to reset: no connection")
	}

	type resetEvent struct {
		Resources []string `json:"resources,omitempty"`
		Access    []string `json:"access,omitempty"`
	}
	ev := resetEvent{Resources: s.resetResources}
	// Only reset access if there are access handlers
	if s.withAccess {
		ev.Access = s.resetAccess
	}
	s.send("system.reset", ev)
}

// startListener listens for nats messages and passes them on to a worker.
func (s *Service) startListener(ch chan *nats.Msg, stopped chan struct{}) {
	for m := range ch {
		s.handleRequest(m)
	}

	close(stopped)
}

// handleRequest is called by the nats listener on incoming messages.
func (s *Service) handleRequest(m *nats.Msg) {
	// Debug logging
	if debug {
		s.Logf("==> %s: %s", m.Subject, m.Data)
	}

	// Assert there is a reply subject
	if m.Reply == "" {
		s.Logf("missing reply subject on request: %s", m.Subject)
		return
	}

	// Get request type
	idx := strings.IndexByte(m.Subject, '.')
	if idx < 0 {
		// Shouldn't be possible unless NATS is really acting up
		s.Logf("invalid request subject: %s", m.Subject)
		return
	}

	var method string
	rtype := m.Subject[:idx]
	rname := m.Subject[idx+1:]

	if rtype == "call" || rtype == "auth" {
		idx = strings.LastIndexByte(rname, '.')
		if idx < 0 {
			// No method? Resgate must be acting up
			s.Logf("invalid request subject: %s", m.Subject)
			return
		}
		method = rname[idx+1:]
		rname = rname[:idx]
	}

	// Get resource name without service name part
	rsub := rname
	idx = strings.IndexByte(rsub, '.')
	if idx < 0 {
		rsub = ""
	} else {
		rsub = rsub[idx+1:]
	}

	hs, params := s.patterns.get(rsub)

	cb := func() {
		s.processRequest(m, rtype, rname, method, hs, params)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Get current work queue for the resource
	w, ok := s.rwork[rname]
	if !ok {
		// Create a new work queue and pass it to a worker
		w = &work{
			s:     s,
			rname: rname,
			queue: []func(){cb},
		}
		s.rwork[rname] = w
		s.workCh <- w
	} else {
		// Append callback to existing work queue
		w.queue = append(w.queue, cb)
	}
}

// send marshals the data and sends a message to the NATS server.
func (s *Service) send(subj string, data interface{}) {
	payload, err := json.Marshal(data)
	if err == nil {
		if debug {
			s.Logf("<-- %s: %s", subj, payload)
		}
		err = s.nc.Publish(subj, payload)
	}
	if err != nil {
		s.Logf("error sending request %s: %s", subj, err)
	}
}

// handleReconnect is called when nats has reconnected.
// It calls a system.reset to have the resgates update their caches.
func (s *Service) handleReconnect(_ *nats.Conn) {
	s.Reset()
}
