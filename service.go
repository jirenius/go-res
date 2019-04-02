package res

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jirenius/resgate/logger"
	"github.com/jirenius/timerqueue"
	nats "github.com/nats-io/go-nats"
)

// The size of the in channel receiving messages from NATS Server.
const inChannelSize = 256

// The number of default workers handling resource requests.
const workerCount = 32

const defaultQueryEventDuration = time.Second * 3

var (
	errNotStopped      = errors.New("res: service is not stopped")
	errNotStarted      = errors.New("res: service is not started")
	errHandlerNotFound = errors.New("res: no matching handlers found")
)

// HandlerOption is a function that sets an option to a resource handler.
type HandlerOption func(*Handler)

// AccessHandler is a function called on resource access requests
type AccessHandler func(AccessRequest)

// ModelHandler is a function called on model get requests
type ModelHandler func(ModelRequest)

// CollectionHandler is a function called on collection get requests
type CollectionHandler func(CollectionRequest)

// CallHandler is a function called on resource call requests
type CallHandler func(CallRequest)

// NewHandler is a function called on new resource call requests
type NewHandler func(NewRequest)

// AuthHandler is a function called on resource auth requests
type AuthHandler func(AuthRequest)

// Handler contains handler functions for a given resource pattern.
type Handler struct {
	// Access handler for access requests
	Access AccessHandler

	// Get handler for models. If not nil, all other Get handlers must be nil.
	GetModel ModelHandler

	// Get handler for collections. If not nil, all other Get handlers must be nil.
	GetCollection CollectionHandler

	// Call handlers for call requests
	Call map[string]CallHandler

	// New handler for new call requests
	New NewHandler

	// Auth handler for auth requests
	Auth map[string]AuthHandler

	// Group is the identifier of the group the resource belongs to.
	// All resources of the same group will be handled on the same
	// goroutine.
	// If empty, the resource name will be used as identifier.
	Group string
}

type regHandler struct {
	Handler
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

	nc             Conn                          // NATS Server connection
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
	queryTQ        *timerqueue.Queue             // Timer queue for query events duration
	queryDuration  time.Duration                 // Duration to listen for query requests on a query event

}

// NewService creates a new Service given a service name.
// The name must be a non-empty alphanumeric string with no embedded whitespace.
func NewService(name string) *Service {
	// [TODO] panic on invalid name
	return &Service{
		Name:          name,
		patterns:      patterns{root: &node{}},
		logger:        logger.NewStdLogger(false, false),
		queryDuration: defaultQueryEventDuration,
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

// SetQueryEventDuration sets the duration for which the service
// will listen for query requests sent on a query event.
// Default is 3 seconds
func (s *Service) SetQueryEventDuration(d time.Duration) *Service {
	if s.nc != nil {
		panic("res: service already started")
	}
	s.queryDuration = d
	return s
}

// Logger returns the logger.
func (s *Service) Logger() logger.Logger {
	return s.logger
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

// Handle registers the handler functions for the given resource pattern.
//
// A pattern may contain placeholders that acts as wildcards, and will be
// parsed and stored in the request.PathParams map.
// A placeholder is a resource name part starting with a dollar ($) character:
//  s.Handle("user.$id", handlers) // Will match "user.10", "user.foo", etc.
//
// If the pattern is already registered, or if there are conflicts among
// the handlers, Handle panics.
func (s *Service) Handle(pattern string, hf ...HandlerOption) {
	var h Handler
	for _, f := range hf {
		f(&h)
	}
	s.AddHandler(pattern, h)
}

// AddHandler register a handler for the given resource pattern.
// The pattern used is the same as described for Handle.
func (s *Service) AddHandler(pattern string, hs Handler) {
	if hs.Access != nil {
		s.withAccess = true
	}
	h := regHandler{
		Handler: hs,
		typ:     validateGetHandlers(hs),
	}
	s.patterns.add(s.Name+"."+pattern, &h)
}

// Access sets a handler for resource access requests
func Access(h AccessHandler) HandlerOption {
	return func(hs *Handler) {
		if hs.Access != nil {
			panic("res: multiple access handlers")
		}
		hs.Access = h
	}
}

// GetModel sets a handler for model get requests
func GetModel(h ModelHandler) HandlerOption {
	return func(hs *Handler) {
		hs.GetModel = h
		validateGetHandlers(*hs)
	}
}

// GetCollection sets a handler for collection get requests
func GetCollection(h CollectionHandler) HandlerOption {
	return func(hs *Handler) {
		hs.GetCollection = h
		validateGetHandlers(*hs)
	}
}

// Call sets a handler for resource call requests.
// Panics if the method is one of the pre-defined call methods, set, or new.
// For pre-defined call methods, the matching handlers, Set, and New
// should be used instead.
func Call(method string, h CallHandler) HandlerOption {
	if method == "new" {
		panic("res: new handler should be registered using the New method")
	}
	return func(hs *Handler) {
		if hs.Call == nil {
			hs.Call = make(map[string]CallHandler)
		}
		if _, ok := hs.Call[method]; ok {
			panic("res: multiple call handlers for method " + method)
		}
		hs.Call[method] = h
	}
}

// Set sets a handler for set resource requests.
// Is a n alias for Call("set", h)
func Set(h CallHandler) HandlerOption {
	return Call("set", h)
}

// New sets a handler for new resource requests.
func New(h NewHandler) HandlerOption {
	return func(hs *Handler) {
		if hs.New != nil {
			panic("res: multiple new handlers")
		}
		hs.New = h
	}
}

// Auth sets a handler for resource auth requests
func Auth(method string, h AuthHandler) HandlerOption {
	return func(hs *Handler) {
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
func (s *Service) ListenAndServe(url string, options ...nats.Option) error {
	if !atomic.CompareAndSwapInt32(&s.state, stateStopped, stateStarting) {
		return errNotStopped
	}

	opts := []nats.Option{
		nats.Name(s.Name),
		nats.MaxReconnects(-1),
		nats.ReconnectHandler(s.handleReconnect),
		nats.DisconnectHandler(s.handleDisconnect),
		nats.ClosedHandler(s.handleClosed),
	}
	opts = append(opts, options...)

	s.Logf("Connecting to NATS server")
	nc, err := nats.Connect(url, opts...)
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
func (s *Service) Serve(nc Conn) error {
	if !atomic.CompareAndSwapInt32(&s.state, stateStopped, stateStarting) {
		return errNotStopped
	}
	return s.serve(nc)
}

func (s *Service) serve(nc Conn) error {
	s.Logf("Starting service: %s", s.Name)

	// Initialize fields
	inCh := make(chan *nats.Msg, inChannelSize)
	workCh := make(chan *work, 1)
	s.nc = nc
	s.inCh = inCh
	s.workCh = workCh
	s.rwork = make(map[string]*work)
	s.queryTQ = timerqueue.New(s.queryEventExpire, s.queryDuration)

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
		s.ResetAll()

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
	s.nc.Close()
	close(s.inCh)
}

// Reset sends a system reset for the provided resource patterns.
func (s *Service) Reset(resources []string, access []string) {
	if atomic.LoadInt32(&s.state) != stateStarted {
		s.Logf("failed to reset: service not started")
		return
	}

	lr := len(resources)
	la := len(access)

	// Quick escape
	if lr == 0 && la == 0 {
		return
	}

	if lr == 0 {
		resources = nil
	}

	if la == 0 {
		access = nil
	}

	s.event("system.reset", resetEvent{
		Resources: resources,
		Access:    access,
	})
}

// ResetAll will send a system.reset to trigger any gateway to update their cache
// for all resources owned by the service
func (s *Service) ResetAll() {
	var resources []string
	var access []string
	if s.resetResources == nil {
		resources = []string{s.Name + ".>"}
	} else {
		resources = s.resetResources
	}

	// Only reset access if there are access handlers
	if s.resetAccess == nil && s.withAccess {
		access = []string{s.Name + ".>"}
	} else {
		access = s.resetAccess
	}
	s.Reset(resources, access)
}

// TokenEvent sends a connection token event that sets the connection's access token,
// discarding any previously set token.
// A change of token will invalidate any previous access response received using the old token.
// A nil token clears any previously set token.
func (s *Service) TokenEvent(cid string, token interface{}) {
	if !isValidPart(cid) {
		panic(`res: invalid connection ID`)
	}
	s.event("conn."+cid+".token", tokenEvent{Token: token})
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
	subj := m.Subject
	s.Tracef("==> %s: %s", subj, m.Data)

	// Assert there is a reply subject
	if m.Reply == "" {
		s.Logf("missing reply subject on request: %s", subj)
		return
	}

	// Get request type
	idx := strings.IndexByte(subj, '.')
	if idx < 0 {
		// Shouldn't be possible unless NATS is really acting up
		s.Logf("invalid request subject: %s", subj)
		return
	}

	var method string
	rtype := subj[:idx]
	rname := subj[idx+1:]

	if rtype == "call" || rtype == "auth" {
		idx = strings.LastIndexByte(rname, '.')
		if idx < 0 {
			// No method? Resgate must be acting up
			s.Logf("invalid request subject: %s", subj)
			return
		}
		method = rname[idx+1:]
		rname = rname[:idx]
	}

	hs, params := s.patterns.get(rname)

	s.runWith(hs, rname, func() {
		s.processRequest(m, rtype, rname, method, hs, params)
	})
}

// runWith enqueues the callback, cb, to be called by the worker goroutine.
// The worker ID of the worker is the hs.wid value, if one is set.
// Otherwise the worker ID will fall back to rname.
func (s *Service) runWith(hs *regHandler, rname string, cb func()) {
	if atomic.LoadInt32(&s.state) != stateStarted {
		return
	}

	wid := rname
	if hs != nil && hs.Group != "" {
		wid = hs.Group
	}

	s.mu.Lock()
	// Get current work queue for the resource
	w, ok := s.rwork[wid]
	if !ok {
		// Create a new work queue and pass it to a worker
		w = &work{
			s:     s,
			wid:   wid,
			queue: []func(){cb},
		}
		s.rwork[wid] = w
		s.mu.Unlock()
		s.workCh <- w
	} else {
		// Append callback to existing work queue
		w.queue = append(w.queue, cb)
		s.mu.Unlock()
	}
}

// With matches the resource ID, rid, with the registered Handlers
// before calling the callback, cb, on the worker goroutine for the
// resource name.
// With will return an error and not call the callback if there are no
// no matching handlers found.
func (s *Service) With(rid string, cb func(r Resource)) error {
	rname, q := parseRID(rid)
	hs, params := s.patterns.get(rname)
	if hs == nil {
		return errHandlerNotFound
	}

	r := &resource{
		rname:      rname,
		pathParams: params,
		query:      q,
		s:          s,
		hs:         hs,
	}

	s.runWith(hs, rname, func() {
		cb(r)
	})

	return nil
}

// event marshals the data and publishes it on a subject,
// and logs it as an outgoing event.
func (s *Service) event(subj string, data interface{}) {
	if data == nil {
		s.rawEvent(subj, nil)
		return
	}

	payload, err := json.Marshal(data)
	if err == nil {
		s.Tracef("<-- %s: %s", subj, payload)
		err = s.nc.Publish(subj, payload)
	}
	if err != nil {
		s.Logf("error sending event %s: %s", subj, err)
	}
}

// rawEvent publishes the payload on a subject,
// and logs it as an outgoing event.
func (s *Service) rawEvent(subj string, payload []byte) {
	s.Tracef("<-- %s: %s", subj, payload)
	err := s.nc.Publish(subj, payload)
	if err != nil {
		s.Logf("error sending event %s: %s", subj, err)
	}
}

// handleReconnect is called when nats has reconnected.
// It calls a system.reset to have the resgates update their caches.
func (s *Service) handleReconnect(_ *nats.Conn) {
	s.Logf("Reconnected to NATS. Sending reset event.")
	s.ResetAll()
}

// handleDisconnect is called when nats is disconnected.
// It calls a system.reset to have the resgates update their caches.
func (s *Service) handleDisconnect(_ *nats.Conn) {
	s.Logf("Lost connection to NATS.")
}

func (s *Service) handleClosed(_ *nats.Conn) {
	s.Shutdown()
}

func validateGetHandlers(h Handler) rtype {
	c := 0
	rtype := rtypeUnset
	if h.GetModel != nil {
		c++
		rtype = rtypeModel
	}
	if h.GetCollection != nil {
		c++
		rtype = rtypeCollection
	}
	if c > 1 {
		panic("res: multiple get handlers")
	}
	return rtype
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

// processRequest is executed by the worker to process an incoming request.
func (s *Service) processRequest(m *nats.Msg, rtype, rname, method string, hs *regHandler, pathParams map[string]string) {
	r := Request{
		resource: resource{
			rname:      rname,
			pathParams: pathParams,
			s:          s,
			hs:         hs,
		},
		rtype:  rtype,
		method: method,
		msg:    m,
	}

	if hs == nil {
		r.reply(responseNotFound)
		return
	}

	var rc resRequest
	err := json.Unmarshal(m.Data, &rc)
	if err != nil {
		s.Logf("error unmarshaling incoming request: %s", err)
		r.error(ToError(err))
		return
	}

	r.cid = rc.CID
	r.params = rc.Params
	r.token = rc.Token
	r.header = rc.Header
	r.host = rc.Host
	r.remoteAddr = rc.RemoteAddr
	r.uri = rc.URI
	r.query = rc.Query

	r.executeHandler()
}

func (s *Service) queryEventExpire(v interface{}) {
	qe := v.(*queryEvent)
	qe.sub.Drain()
	s.runWith(qe.r.hs, qe.r.rname, func() {
		qe.cb(nil)
	})
}
