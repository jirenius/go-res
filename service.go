package res

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/jirenius/resgate/logger"
	nats "github.com/nats-io/go-nats"
)

// The size of the in channel receiving messages from NATS Server.
const inChannelSize = 256

// The number of default workers handling resource requests.
const workerCount = 1

var (
	errAlreadyServing = errors.New("res: service already serving")
)

// Handler is a function for the handlers of a resource
type Handler func(*Handlers)

// AccessHandler is a function called on resource access requests
type AccessHandler func(*Request, *AccessResponse)

// GetModelHandler is a function called on model get requests
type GetModelHandler func(*Request, *GetModelResponse)

// GetCollectionHandler is a function called on collection get requests
type GetCollectionHandler func(*Request, *GetCollectionResponse)

// CallHandler is a function called on resource call requests
type CallHandler func(*Request, *CallResponse)

// NewHandler is a function called on new resource call requests
type NewHandler func(*Request, *NewResponse)

// AuthHandler is a function called on resource auth requests
type AuthHandler func(*Request, *AuthResponse)

// Handlers contains handlers for a given resource pattern.
type Handlers struct {
	// Use middleware handlers for requests
	Use []func(Handler) Handler

	// Access handler for access requests
	Access AccessHandler

	// Get handler for models. If not nil, all other Get handlers must be nil.
	GetModel GetModelHandler

	// Get handler for collections. If not nil, all other Get handlers must be nil.
	GetCollection GetCollectionHandler

	// Call handler for call requests
	Call map[string]CallHandler

	// New handler for new call requests
	New NewHandler

	// Auth handler for auth requests
	Auth map[string]AuthHandler

	typ rtype
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
	inCh           chan *nats.Msg                // Channel for incoming nats messages
	rwork          map[string]*work              // map of resource work
	workCh         chan *work                    // Resource work channel, listened to by the workers
	wg             sync.WaitGroup                // WaitGroup for all workers
	mu             sync.Mutex                    // Mutex to protect rwork map
	stopped        chan struct{}                 // Channel that is closed by the listen worker, on stop.
	logger         logger.Logger                 // Logger
	withAccess     bool                          // Flag that is true if there are patterns with Access handlers
	resetResources []string                      // List of resource name patterns used on system.reset for resources. Defaults to serviceName+">"
	resetAccess    []string                      // List of resource name patterns used system.reset for access. Defaults to serviceName+">"
}

// NewService creates a new Service given a service name.
// The name must be a non-empty alphanumeric string with no embedded whitespace.
func NewService(name string) *Service {
	// [TODO] panic on invalid name
	return &Service{
		Name:           name,
		patterns:       patterns{root: &node{}},
		logger:         logger.NewStdLogger(false, false),
		resetResources: []string{name + ".>"},
		resetAccess:    []string{name + ".>"},
	}
}

// SetLogger sets the logger.
// Panics if service is already started.
func (s *Service) SetLogger(l logger.Logger) *Service {
	if s.nc != nil {
		panic("res: service already started")
	}

	s.logger = l
	return s
}

// Logf writes a formatted log message
func (s *Service) Logf(format string, v ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Logf("[Service] ", format, v...)
}

// Debugf writes a formatted debug message
func (s *Service) Debugf(format string, v ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Debugf("[Service] ", format, v...)
}

// Tracef writes a formatted trace message
func (s *Service) Tracef(format string, v ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Tracef("[Service] ", format, v...)
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

// GetModel is a handler for model get requests
func GetModel(h GetModelHandler) Handler {
	return func(hs *Handlers) {
		assertNoGetHandler(hs)
		hs.GetModel = h
		hs.typ = rtypeModel
	}
}

// GetCollection is a handler for collection get requests
func GetCollection(h GetCollectionHandler) Handler {
	return func(hs *Handlers) {
		assertNoGetHandler(hs)
		hs.GetCollection = h
		hs.typ = rtypeCollection
	}
}

// Call is a handler for resource call requests.
// Panics if the method is one of the pre-defined call methods, set, or new.
// For pre-defined call methods, the matching handlers, Set, and New
// should be used instead.
func Call(method string, h CallHandler) Handler {
	if method == "new" {
		panic("res: use New to handle new calls")
	}
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

// Set is a handler for set resource requests.
// Is a n alias for Call("set", h)
func Set(h CallHandler) Handler {
	return Call("set", h)
}

// New is a handler for new resource requests.
func New(h NewHandler) Handler {
	return func(hs *Handlers) {
		if hs.New != nil {
			panic("res: multiple new handlers")
		}
		hs.New = h
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

// ListenAndServe connects to the NATS server at the url. Once connected,
// it subscribes to incoming requests and serves them on a single goroutine
// in the order they are recieved. For each request, it calls the appropriate
// handler, or replies with the appropriate error if no handler is available.
//
// In case of disconnect, it will try to reconnect until Close is called,
// or until successfully reconnecting, upon which Reset will be called.
//
// ListenAndServe returns an error if failes to connect or subscribe.
// Otherwise, nil is returned once the connection is closed using Close.
func (s *Service) ListenAndServe(url string) error {
	if s.nc != nil {
		return errAlreadyServing
	}

	opts := nats.Options{
		Url:            url,
		Name:           s.Name,
		AllowReconnect: true,
		MaxReconnect:   -1,
	}

	s.Logf("Connecting to NATS server")
	nc, err := opts.Connect()
	if err != nil {
		s.Logf("Failed to connect to NATS server: %s", err)
		return err
	}

	nc.SetReconnectHandler(s.handleReconnect)
	nc.SetDisconnectHandler(s.handleDisconnect)
	nc.SetClosedHandler(s.handleClosed)

	return s.Serve(nc)
}

// Serve subscribes to incoming requests on the *Conn nc, serving them on
// a single goroutine in the order they are recieved. For each request,
// it calls the appropriate handler, or replies with the appropriate
// error if no handler is available.
//
// Serve returns an error if failes to subscribe. Otherwise, nil is
// returned once the *Conn is closed.
func (s *Service) Serve(nc *nats.Conn) error {
	if s.nc != nil {
		return errAlreadyServing
	}

	s.Logf("Starting service: %s", s.Name)

	s.nc = nc
	s.inCh = make(chan *nats.Msg, inChannelSize)
	s.stopped = make(chan struct{})

	s.rwork = make(map[string]*work)
	s.workCh = make(chan *work)
	s.wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go s.startWorker(s.workCh)
	}

	err := s.subscribe()
	if err != nil {
		s.Stop()
		return err
	}

	// Always start with a reset
	s.Reset()

	s.Logf("Listening for requests")
	s.startListener(s.inCh, s.stopped)

	return nil
}

// Stop closes any existing connection to NATS Server.
func (s *Service) Stop() {
	nc := s.nc
	if nc == nil {
		return
	}

	s.Logf("Stopping service...")
	if !nc.IsClosed() {
		nc.Close()
	}

	// Stop listener by closing incoming channel
	close(s.inCh)
	stopped := s.stopped

	// Wait until listener has stopped
	<-stopped

	// Stop all workers by closing worker channel
	close(s.workCh)
	s.wg.Wait()

	s.stopped = nil
	s.inCh = nil
	s.nc = nil
	s.subs = nil
	s.workCh = nil

	s.Logf("Stopped")
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

// startListener listens for nats messages and passes them on to a worker.
func (s *Service) startListener(ch chan *nats.Msg, stopped chan struct{}) {
	for m := range ch {
		s.handleRequest(m)
	}
	close(stopped)
}

// handleRequest is called by the nats listener on incoming messages.
func (s *Service) handleRequest(m *nats.Msg) {
	s.Tracef("==> %s: %s", m.Subject, m.Data)

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
	s.Enqueue(rname, func() {
		s.processRequest(m, rtype, rname, method, hs, params)
	})
}

func (s *Service) Enqueue(rname string, cb func()) {
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
		s.Tracef("<-- %s: %s", subj, payload)
		err = s.nc.Publish(subj, payload)
	}
	if err != nil {
		s.Logf("error sending request %s: %s", subj, err)
	}
}

// handleReconnect is called when nats has reconnected.
// It calls a system.reset to have the resgates update their caches.
func (s *Service) handleReconnect(_ *nats.Conn) {
	s.Logf("Reconnected to NATS. Sending system.reset eve.")
	s.Reset()
}

// handleDisconnect is called when nats is disconnected.
// It calls a system.reset to have the resgates update their caches.
func (s *Service) handleDisconnect(_ *nats.Conn) {
	s.Logf("Connection to NATS is lost.")
}

func (s *Service) handleClosed(_ *nats.Conn) {
	s.Stop()
}

func assertNoGetHandler(hs *Handlers) {
	if hs.typ != rtypeUnset {
		panic("res: multiple get handlers")
	}
}
