package res

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/jirenius/resgate/logger"
	nats "github.com/nats-io/go-nats"
)

// The size of the in channel receiving messages from NATS Server.
const inChannelSize = 256

// The number of default workers handling resource requests.
const workerCount = 1

var (
	errNotStopped      = errors.New("res: service is not stopped")
	errNotStarted      = errors.New("res: service is not started")
	errHandlerNotFound = errors.New("res: no matching handlers found")
)

// Handler is a function for the handlers of a resource
type Handler func(*Handlers)

// AccessHandler is a function called on resource access requests
type AccessHandler func(AccessResponse, *Request)

// GetModelHandler is a function called on model get requests
type GetModelHandler func(GetModelResponse, *Request)

// GetCollectionHandler is a function called on collection get requests
type GetCollectionHandler func(GetCollectionResponse, *Request)

// CallHandler is a function called on resource call requests
type CallHandler func(CallResponse, *Request)

// NewHandler is a function called on new resource call requests
type NewHandler func(NewResponse, *Request)

// AuthHandler is a function called on resource auth requests
type AuthHandler func(AuthResponse, *Request)

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

const (
	stateStopped = iota
	stateStarting
	stateStarted
	stateStopping
)

// A Service handles incoming requests from NATS Server and calls the
// appropriate callback on the resource handlers.
type Service struct {
	// Name of the service.
	// Must be a non-empty alphanumeric string with no embedded whitespace.
	Name string

	state int32

	nc             *nats.Conn                    // NATS Server connection
	subs           map[string]*nats.Subscription // Request type nats subscriptions
	patterns       patterns                      // pattern store with all handlers
	inCh           chan *nats.Msg                // Channel for incoming nats messages
	rwork          map[string]*work              // map of resource work
	workCh         chan *work                    // Resource work channel, listened to by the workers
	wg             sync.WaitGroup                // WaitGroup for all workers
	mu             sync.Mutex                    // Mutex to protect rwork map
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
		logger:         logger.NewStdLogger(true, true),
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
	s.patterns.add(s.Name+"."+pattern, &hs)
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
// in the order they are received. For each request, it calls the appropriate
// handler, or replies with the appropriate error if no handler is available.
//
// In case of disconnect, it will try to reconnect until Close is called,
// or until successfully reconnecting, upon which Reset will be called.
//
// ListenAndServe returns an error if failes to connect or subscribe.
// Otherwise, nil is returned once the connection is closed using Close.
func (s *Service) ListenAndServe(url string) error {
	if !atomic.CompareAndSwapInt32(&s.state, stateStopped, stateStarting) {
		return errNotStopped
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

	return s.serve(nc)
}

// Serve subscribes to incoming requests on the *Conn nc, serving them on
// a single goroutine in the order they are received. For each request,
// it calls the appropriate handler, or replies with the appropriate
// error if no handler is available.
//
// Serve returns an error if failes to subscribe. Otherwise, nil is
// returned once the *Conn is closed.
func (s *Service) Serve(nc *nats.Conn) error {
	if !atomic.CompareAndSwapInt32(&s.state, stateStopped, stateStarting) {
		return errNotStopped
	}
	return s.serve(nc)
}

func (s *Service) serve(nc *nats.Conn) error {
	s.Logf("Starting service: %s", s.Name)

	// Initialize fields
	inCh := make(chan *nats.Msg, inChannelSize)
	workCh := make(chan *work, 1)
	s.nc = nc
	s.inCh = inCh
	s.workCh = workCh
	s.rwork = make(map[string]*work)

	// Start workers
	s.wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go s.startWorker(s.workCh)
	}

	atomic.StoreInt32(&s.state, stateStarted)

	err := s.subscribe()
	if err != nil {
		s.Logf("Failed to subscribe: %s", err)
		s.close()
	} else {
		// Always start with a reset
		s.Reset()

		s.Logf("Listening for requests")
		s.startListener(inCh)
	}

	// Stop all workers by closing worker channel
	close(workCh)

	// Wait for all workers to be done
	s.wg.Wait()
	return nil
}

// Shutdown closes any existing connection to NATS Server.
// Returns an error if service is not started.
func (s *Service) Shutdown() error {
	if !atomic.CompareAndSwapInt32(&s.state, stateStarted, stateStopping) {
		return errNotStarted
	}

	s.Logf("Stopping service...")
	s.close()

	// Wait for all workers to be done
	s.wg.Wait()

	s.inCh = nil
	s.nc = nil
	s.subs = nil
	s.workCh = nil

	atomic.StoreInt32(&s.state, stateStopped)

	s.Logf("Stopped")
	return nil
}

// close calls Close on the NATS connection, and closes the incoming channel
func (s *Service) close() {
	if !s.nc.IsClosed() {
		s.nc.Close()
	}
	close(s.inCh)
}

// Reset will send a system.reset to trigger any gateway to update their cache
func (s *Service) Reset() {
	if atomic.LoadInt32(&s.state) != stateStarted {
		s.Logf("failed to reset: service not started")
		return
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
func (s *Service) startListener(ch chan *nats.Msg) {
	for m := range ch {
		s.handleRequest(m)
	}
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

	hs, params := s.patterns.get(rname)
	s.RunWith(rname, func() {
		s.processRequest(m, rtype, rname, method, hs, params)
	})
}

// RunWith enqueues the callback, cb, to be called by the worker goroutine
// for the resource name.
func (s *Service) RunWith(rname string, cb func()) {
	if atomic.LoadInt32(&s.state) != stateStarted {
		return
	}

	s.mu.Lock()
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
		s.mu.Unlock()
		s.workCh <- w
	} else {
		// Append callback to existing work queue
		w.queue = append(w.queue, cb)
		s.mu.Unlock()
	}
}

// Get matches the resource ID, rid, with the registered Handlers
// before calling the callback, cb, on the worker goroutine for the
// resource name.
// Get will return an error and not call the callback if there are no
// no matching handlers found.
func (s *Service) Get(rid string, cb func(r *Resource)) error {
	rname, q := parseRID(rid)
	hs, params := s.patterns.get(rname)
	if hs == nil {
		return errHandlerNotFound
	}

	r := &Resource{
		ResourceName: rname,
		PathParams:   params,
		RawQuery:     q,
		s:            s,
		h:            hs,
	}

	s.RunWith(rname, func() {
		cb(r)
	})

	return nil
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
	s.Logf("Reconnected to NATS. Sending reset event.")
	s.Reset()
}

// handleDisconnect is called when nats is disconnected.
// It calls a system.reset to have the resgates update their caches.
func (s *Service) handleDisconnect(_ *nats.Conn) {
	s.Logf("Lost connection to NATS.")
}

func (s *Service) handleClosed(_ *nats.Conn) {
	s.Shutdown()
}

func assertNoGetHandler(hs *Handlers) {
	if hs.typ != rtypeUnset {
		panic("res: multiple get handlers")
	}
}

// subname returns the resource name without service name part
func subname(rname string) string {
	idx := strings.IndexByte(rname, '.')
	if idx < 0 {
		return ""
	}
	return rname[idx+1:]
}

// parseRID parses a resource ID, rid, and splits it into the resource name
// and query, if one is available.
// The question mark query separator is not included in the returned
// query string.
func parseRID(rid string) (rname string, q string) {
	i := strings.IndexByte(rid, '?')
	if i == -1 {
		return rid, ""
	}

	return rid[:i], rid[i+1:]
}
