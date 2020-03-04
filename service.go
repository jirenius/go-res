package res

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jirenius/go-res/logger"
	"github.com/jirenius/timerqueue"
	nats "github.com/nats-io/nats.go"
)

// Supported RES protocol version.
const protocolVersion = "1.2.0"

// The size of the in channel receiving messages from NATS Server.
const inChannelSize = 256

// The number of default workers handling resource requests.
const workerCount = 32

const defaultQueryEventDuration = time.Second * 3

var (
	errNotStopped = errors.New("res: service is not stopped")
	errNotStarted = errors.New("res: service is not started")
)

// Option set one or more of the handler functions for a resource Handler.
type Option interface{ SetOption(*Handler) }

// The OptionFunc type is an adapter to allow the use of ordinary functions as
// options. If f is a function with the appropriate signature, OptionFunc(f) is
// an Option that calls f.
type OptionFunc func(*Handler)

// SetOption calls f(hs)
func (f OptionFunc) SetOption(hs *Handler) { f(hs) }

// AccessHandler is a function called on resource access requests
type AccessHandler func(AccessRequest)

// GetHandler is a function called on untyped get requests
type GetHandler func(GetRequest)

// ModelHandler is a function called on model get requests
type ModelHandler func(ModelRequest)

// CollectionHandler is a function called on collection get requests
type CollectionHandler func(CollectionRequest)

// CallHandler is a function called on resource call requests
type CallHandler func(CallRequest)

// NewHandler is a function called on new resource call requests
//
// Deprecated: Use CallHandler with Resource response instead; deprecated in RES
// protocol v1.2.0
type NewHandler func(NewRequest)

// AuthHandler is a function called on resource auth requests
type AuthHandler func(AuthRequest)

// ApplyChangeHandler is a function called to apply a model change event. Must
// return a map with the values to apply to revert the changes, or error.
type ApplyChangeHandler func(r Resource, changes map[string]interface{}) (map[string]interface{}, error)

// ApplyAddHandler is a function called to apply a collection add event. Must
// return an error if the add event couldn't be applied to the resource.
type ApplyAddHandler func(r Resource, value interface{}, idx int) error

// ApplyRemoveHandler is a function called to apply a collection remove event.
// Must return the value being removed, or error.
type ApplyRemoveHandler func(r Resource, idx int) (interface{}, error)

// ApplyCreateHandler is a function called to apply a resource create event.
// Must return an error if the resource couldn't be created.
type ApplyCreateHandler func(r Resource, data interface{}) error

// ApplyDeleteHandler is a function called to apply a resource delete event.
// Must return the resource data being removed, or error.
type ApplyDeleteHandler func(r Resource) (interface{}, error)

// Handler contains handler functions for a given resource pattern.
type Handler struct {
	// Resource type
	Type ResourceType

	// Access handler for access requests
	Access AccessHandler

	// Get handler for get requests.
	Get GetHandler

	// Call handlers for call requests
	Call map[string]CallHandler

	// New handler for new call requests
	//
	// Deprecated: Use Call with Resource response instead; deprecated in RES
	// protocol v1.2.0
	New NewHandler

	// Auth handler for auth requests
	Auth map[string]AuthHandler

	// ApplyChange handler for applying change event mutations
	ApplyChange ApplyChangeHandler

	// ApplyAdd handler for applying add event mutations
	ApplyAdd ApplyAddHandler

	// ApplyRemove handler for applying remove event mutations
	ApplyRemove ApplyRemoveHandler

	// ApplyCreate handler for applying create event
	ApplyCreate ApplyCreateHandler

	// ApplyDelete handler for applying delete event
	ApplyDelete ApplyDeleteHandler

	// Group is the identifier of the group the resource belongs to. All
	// resources of the same group will be handled on the same goroutine. The
	// group may contain tags, ${tagName}, where the tag name matches a
	// parameter placeholder name in the resource pattern. If empty, the
	// resource name will be used as identifier.
	Group string

	// OnRegister is callback that is to be call when the handler has been
	// registered to a service.
	//
	// The pattern is the full resource pattern for the resource, including any
	// service name or mount paths.
	//
	// The handler is the handler being registered, and should be considered
	// immutable.
	OnRegister func(service *Service, pattern Pattern, rh Handler)

	// Listeners is a map of event listeners, where the key is the resource
	// pattern being listened on, and the value being the callback called on
	// events.
	//
	// The callback will be called in the context of the resource emitting the
	// event.
	Listeners map[string]func(*Event)
}

const (
	stateStopped = iota
	stateStarting
	stateStarted
	stateStopping
)

var (
	// Model sets handler type to model
	Model = OptionFunc(func(hs *Handler) {
		if hs.Type != TypeUnset {
			panic("res: resource type set multiple times")
		}
		hs.Type = TypeModel
	})

	// Collection sets handler type to collection
	Collection = OptionFunc(func(hs *Handler) {
		if hs.Type != TypeUnset {
			panic("res: resource type set multiple times")
		}
		hs.Type = TypeCollection
	})
)

// Option sets handler fields by passing one or more handler options.
func (h *Handler) Option(hf ...Option) {
	for _, f := range hf {
		f.SetOption(h)
	}
}

// A Service handles incoming requests from NATS Server and calls the
// appropriate callback on the resource handlers.
type Service struct {
	*Mux
	state          int32
	nc             Conn                   // NATS Server connection
	inCh           chan *nats.Msg         // Channel for incoming nats messages
	rwork          map[string]*work       // map of resource work
	workCh         chan *work             // Resource work channel, listened to by the workers
	wg             sync.WaitGroup         // WaitGroup for all workers
	mu             sync.Mutex             // Mutex to protect rwork map
	logger         logger.Logger          // Logger
	resetResources []string               // List of resource name patterns used on system.reset for resources. Defaults to serviceName+">"
	resetAccess    []string               // List of resource name patterns used system.reset for access. Defaults to serviceName+">"
	queryTQ        *timerqueue.Queue      // Timer queue for query events duration
	queryDuration  time.Duration          // Duration to listen for query requests on a query event
	onServe        func(*Service)         // Handler called after the starting to serve prior to calling system.reset
	onDisconnect   func(*Service)         // Handler called after the service has been disconnected from NATS server.
	onReconnect    func(*Service)         // Handler called after the service has reconnected to NATS server and sent a system reset event.
	onError        func(*Service, string) // Handler called on errors within the service, or incoming messages not complying with the RES protocol.
}

// NewService creates a new Service.
//
// The name is the service name which will be prefixed to all resources. It must
// be an alphanumeric string with no embedded whitespace, or empty. If name is
// an empty string, the Service will by default handle all resources for all
// namespaces. Use SetReset to limit the namespace scope.
func NewService(name string) *Service {
	s := &Service{
		Mux:           NewMux(name),
		logger:        logger.NewStdLogger(),
		queryDuration: defaultQueryEventDuration,
	}
	s.Mux.Register(s)
	return s
}

// SetLogger sets the logger. Panics if service is already started.
func (s *Service) SetLogger(l logger.Logger) *Service {
	if s.nc != nil {
		panic("res: service already started")
	}
	s.logger = l
	return s
}

// SetQueryEventDuration sets the duration for which the service will listen for
// query requests sent on a query event. Default is 3 seconds
func (s *Service) SetQueryEventDuration(d time.Duration) *Service {
	if s.nc != nil {
		panic("res: service already started")
	}
	s.queryDuration = d
	return s
}

// SetOnServe sets a function to call when the service has started after sending
// the initial system reset event.
func (s *Service) SetOnServe(f func(*Service)) {
	s.onServe = f
}

// SetOnDisconnect sets a function to call when the service has been
// disconnected from NATS server.
func (s *Service) SetOnDisconnect(f func(*Service)) {
	s.onDisconnect = f
}

// SetOnReconnect sets a function to call when the service has reconnected to
// NATS server and sent a system reset event.
func (s *Service) SetOnReconnect(f func(*Service)) {
	s.onReconnect = f
}

// SetOnError sets a function to call on errors within the service, or incoming
// messages not complying with the RES protocol.
func (s *Service) SetOnError(f func(*Service, string)) {
	s.onError = f
}

// Logger returns the logger.
func (s *Service) Logger() logger.Logger {
	return s.logger
}

// ProtocolVersion returns the supported RES protocol version.
func (s *Service) ProtocolVersion() string {
	return protocolVersion
}

// infof logs a formatted info entry.
func (s *Service) infof(format string, v ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Infof(format, v...)
}

// errorf logs a formatted error entry.
func (s *Service) errorf(format string, v ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Errorf(format, v...)
	if s.onError != nil {
		s.onError(s, fmt.Sprintf(format, v...))
	}
}

// tracef logs a formatted trace entry.
func (s *Service) tracef(format string, v ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Tracef(format, v...)
}

// Access sets a handler for resource access requests.
func Access(h AccessHandler) Option {
	return OptionFunc(func(hs *Handler) {
		if hs.Access != nil {
			panic("res: multiple access handlers")
		}
		hs.Access = h
	})
}

// GetModel sets a handler for model get requests.
func GetModel(h ModelHandler) Option {
	return OptionFunc(func(hs *Handler) {
		Model(hs)
		validateGetHandler(*hs)
		hs.Get = func(r GetRequest) { h(ModelRequest(r)) }
	})
}

// GetCollection sets a handler for collection get requests.
func GetCollection(h CollectionHandler) Option {
	return OptionFunc(func(hs *Handler) {
		Collection(hs)
		validateGetHandler(*hs)
		hs.Get = func(r GetRequest) { h(CollectionRequest(r)) }
	})
}

// GetResource sets a handler for untyped resource get requests.
func GetResource(h GetHandler) Option {
	return OptionFunc(func(hs *Handler) {
		validateGetHandler(*hs)
		hs.Get = h
	})
}

// Call sets a handler for resource call requests.
//
// Panics if the method is the pre-defined call method set.
//
// For pre-defined set call methods, the handler Set should be used instead.
func Call(method string, h CallHandler) Option {
	if !isValidPart(method) {
		panic("res: invalid method name: " + method)
	}
	return OptionFunc(func(hs *Handler) {
		if hs.Call == nil {
			hs.Call = make(map[string]CallHandler)
		}
		if _, ok := hs.Call[method]; ok {
			panic("res: multiple call handlers for method " + method)
		}
		hs.Call[method] = h
	})
}

// Set sets a handler for set resource requests.
//
// Is a n alias for Call("set", h)
func Set(h CallHandler) Option {
	return Call("set", h)
}

// New sets a handler for new resource requests.
//
// Deprecated: Use Call with Resource response instead; deprecated in RES
// protocol v1.2.0
func New(h NewHandler) Option {
	return OptionFunc(func(hs *Handler) {
		if hs.New != nil {
			panic("res: multiple new handlers")
		}
		hs.New = h
	})
}

// Auth sets a handler for resource auth requests.
func Auth(method string, h AuthHandler) Option {
	if !isValidPart(method) {
		panic("res: invalid method name: " + method)
	}
	return OptionFunc(func(hs *Handler) {
		if hs.Auth == nil {
			hs.Auth = make(map[string]AuthHandler)
		}
		if _, ok := hs.Auth[method]; ok {
			panic("res: multiple auth handlers for method " + method)
		}
		hs.Auth[method] = h
	})
}

// ApplyChange sets a handler for applying change events.
func ApplyChange(h ApplyChangeHandler) Option {
	return OptionFunc(func(hs *Handler) {
		if hs.ApplyChange != nil {
			panic("res: multiple apply change handlers")
		}
		hs.ApplyChange = h
	})
}

// ApplyAdd sets a handler for applying add events.
func ApplyAdd(h ApplyAddHandler) Option {
	return OptionFunc(func(hs *Handler) {
		if hs.ApplyAdd != nil {
			panic("res: multiple apply add handlers")
		}
		hs.ApplyAdd = h
	})
}

// ApplyRemove sets a handler for applying remove events.
func ApplyRemove(h ApplyRemoveHandler) Option {
	return OptionFunc(func(hs *Handler) {
		if hs.ApplyRemove != nil {
			panic("res: multiple apply remove handlers")
		}
		hs.ApplyRemove = h
	})
}

// ApplyCreate sets a handler for applying create events.
func ApplyCreate(h ApplyCreateHandler) Option {
	return OptionFunc(func(hs *Handler) {
		if hs.ApplyCreate != nil {
			panic("res: multiple apply create handlers")
		}
		hs.ApplyCreate = h
	})
}

// ApplyDelete sets a handler for applying delete events.
func ApplyDelete(h ApplyDeleteHandler) Option {
	return OptionFunc(func(hs *Handler) {
		if hs.ApplyDelete != nil {
			panic("res: multiple apply delete handlers")
		}
		hs.ApplyDelete = h
	})
}

// Group sets a group ID. All resources of the same group will be handled on the
// same goroutine.
//
// The group may contain tags, ${tagName}, where the tag name matches a
// parameter placeholder name in the resource pattern.
func Group(group string) Option {
	return OptionFunc(func(hs *Handler) {
		hs.Group = group
	})
}

// OnRegister sets a callback to be called when the handler is registered to a
// service.
//
// If a callback is already registered, the new callback will be called after
// the previous one.
func OnRegister(callback func(service *Service, pattern Pattern, rh Handler)) Option {
	return OptionFunc(func(hs *Handler) {
		if hs.OnRegister != nil {
			prevcb := hs.OnRegister
			hs.OnRegister = func(service *Service, pattern Pattern, rh Handler) {
				prevcb(service, pattern, rh)
				callback(service, pattern, rh)
			}
		} else {
			hs.OnRegister = callback
		}
	})
}

// SetReset is an alias for SetOwnedResources.
//
// Deprecated: Renamed to SetOwnedResources to match API of similar libraries.
func (s *Service) SetReset(resources, access []string) *Service {
	return s.SetOwnedResources(resources, access)
}

// SetOwnedResources sets the patterns which the service will handle requests
// for. The resources slice patterns ill be listened to for get, call, and auth
// requests. The access slice patterns will be listened to for access requests.
// These patterns will be used when a ResetAll is made.
//
//  // Handle all requests for resources prefixed "library."
//  service.SetOwnedResources([]string{"library.>"}, []string{"library.>"})
//  // Handle access requests for any resource
//  service.SetOwnedResources([]string{}, []string{">"})
//  // Handle non-access requests for a subset of resources
//  service.SetOwnedResources([]string{"library.book", "library.books.*"}, []string{})
//
// If set to nil (default), the service will default to set ownership of all
// resources prefixed with its own path if one was provided when creating the
// service (eg. "serviceName.>"), or to all resources if no name was provided.
// It will take resource ownership if it has at least one registered handler has
// a Get, Call, or Auth handler method not being nil. It will take access
// ownership if it has at least one registered handler with the Access method
// not being nil.
//
// For more details on system reset, see:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#system-reset-event
func (s *Service) SetOwnedResources(resources, access []string) *Service {
	s.resetResources = resources
	s.resetAccess = access
	return s
}

// ListenAndServe connects to the NATS server at the url. Once connected, it
// subscribes to incoming requests and serves them on a single goroutine in the
// order they are received. For each request, it calls the appropriate handler,
// or replies with the appropriate error if no handler is available.
//
// In case of disconnect, it will try to reconnect until Close is called, or
// until successfully reconnecting, upon which Reset will be called.
//
// ListenAndServe returns an error if failes to connect or subscribe. Otherwise,
// nil is returned once the connection is closed using Close.
func (s *Service) ListenAndServe(url string, options ...nats.Option) error {
	if !atomic.CompareAndSwapInt32(&s.state, stateStopped, stateStarting) {
		return errNotStopped
	}

	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectHandler(s.handleReconnect),
		nats.DisconnectHandler(s.handleDisconnect),
		nats.ClosedHandler(s.handleClosed),
	}
	if s.Mux.path != "" {
		opts = append(opts, nats.Name(s.Mux.path))
	}
	opts = append(opts, options...)

	s.infof("Connecting to NATS server")
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		s.errorf("Failed to connect to NATS server: %s", err)
		return err
	}

	nc.SetReconnectHandler(s.handleReconnect)
	nc.SetDisconnectHandler(s.handleDisconnect)
	nc.SetClosedHandler(s.handleClosed)

	return s.serve(nc)
}

// Serve subscribes to incoming requests on the *Conn nc, serving them on a
// single goroutine in the order they are received. For each request, it calls
// the appropriate handler, or replies with the appropriate error if no handler
// is available.
//
// Serve returns an error if failes to subscribe. Otherwise, nil is returned
// once the *Conn is closed.
func (s *Service) Serve(nc Conn) error {
	if !atomic.CompareAndSwapInt32(&s.state, stateStopped, stateStarting) {
		return errNotStopped
	}
	return s.serve(nc)
}

func (s *Service) serve(nc Conn) error {
	s.infof("Starting service")

	// Validate that there are resources registered
	// for all the event listeners.
	err := s.ValidateListeners()
	if err != nil {
		return err
	}

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

	err = s.subscribe()
	if err != nil {
		s.errorf("Failed to subscribe: %s", err)
		go s.Shutdown()
	} else {
		// Send a system.reset
		s.ResetAll()
		// Call onServe callback
		if s.onServe != nil {
			s.onServe(s)
		}

		s.infof("Listening for requests")
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

	s.infof("Stopping service...")
	s.close()

	// Wait for all workers to be done
	s.wg.Wait()

	s.inCh = nil
	s.nc = nil
	s.workCh = nil

	atomic.StoreInt32(&s.state, stateStopped)

	s.infof("Stopped")
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
		s.errorf("Failed to reset: service not started")
		return
	}

	s.reset(resources, access)
}

func (s *Service) reset(resources []string, access []string) {
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

// ResetAll will send a system.reset to trigger any gateway to update their
// cache for all resources handled by the service.
//
// The method is automatically called on server start and reconnects.
func (s *Service) ResetAll() {
	if atomic.LoadInt32(&s.state) != stateStarted {
		s.errorf("Failed to reset: service not started")
		return
	}

	s.setDefaultOwnership()

	s.reset(s.resetResources, s.resetAccess)
}

// TokenEvent sends a connection token event that sets the connection's access
// token, discarding any previously set token.
//
// A change of token will invalidate any previous access response received using
// the old token.
//
// A nil token clears any previously set token.
func (s *Service) TokenEvent(cid string, token interface{}) {
	if atomic.LoadInt32(&s.state) != stateStarted {
		s.errorf("Failed to send token event: service not started")
		return
	}

	if !isValidPart(cid) {
		panic(`res: invalid connection ID`)
	}
	s.event("conn."+cid+".token", tokenEvent{Token: token})
}

func (s *Service) setDefaultOwnership() {
	if s.resetResources == nil {
		if s.Contains(func(h Handler) bool {
			return h.Get != nil || len(h.Call) > 0 || len(h.Auth) > 0 || h.New != nil
		}) {
			s.resetResources = []string{s.Mux.path, mergePattern(s.Mux.path, ">")}
		} else {
			s.resetResources = []string{}
		}
	}

	if s.resetAccess == nil {
		if s.Contains(func(h Handler) bool {
			return h.Access != nil
		}) {
			s.resetAccess = []string{s.Mux.path, mergePattern(s.Mux.path, ">")}
		} else {
			s.resetAccess = []string{}
		}
	}
}

// subscribe makes a nats subscription for each required request type, based on
// the patterns used for ResetAll.
func (s *Service) subscribe() error {
	s.setDefaultOwnership()
	if len(s.resetResources) == 0 && len(s.resetAccess) == 0 {
		return errors.New("res: no resources to serve")
	}
	var patterns []string
	for _, t := range []string{RequestTypeGet, RequestTypeCall, RequestTypeAuth} {
		for _, p := range s.resetResources {
			pattern := t + "." + p
			if pattern[len(pattern)-1] != '>' && t != RequestTypeGet {
				pattern += ".*"
			}
			patterns = append(patterns, pattern)

		}
	}
	for _, p := range s.resetAccess {
		pattern := "access." + p
		s.tracef("sub %s", pattern)
		_, err := s.nc.ChanSubscribe(pattern, s.inCh)
		if err != nil {
			return err
		}
	}

next:
	for i, pattern := range patterns {
		// Skip patterns that overlap one another
		for j, mpattern := range patterns {
			if i != j && Pattern(mpattern).Matches(pattern) {
				continue next
			}
		}
		s.tracef("sub %s", pattern)
		_, err := s.nc.ChanSubscribe(pattern, s.inCh)
		if err != nil {
			return err
		}
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
	s.tracef("==> %s: %s", subj, m.Data)

	// Assert there is a reply subject
	if m.Reply == "" {
		s.errorf("Missing reply subject on request: %s", subj)
		return
	}

	// Get request type
	idx := strings.IndexByte(subj, '.')
	if idx < 0 {
		// Shouldn't be possible unless NATS is really acting up
		s.errorf("Invalid request subject: %s", subj)
		return
	}

	var method string
	rtype := subj[:idx]
	rname := subj[idx+1:]

	if rtype == "call" || rtype == "auth" {
		idx = strings.LastIndexByte(rname, '.')
		if idx < 0 {
			// No method? Resgate must be acting up
			s.errorf("Invalid request subject: %s", subj)
			return
		}
		method = rname[idx+1:]
		rname = rname[:idx]
	}

	group := rname
	mh := s.GetHandler(rname)
	if mh != nil {
		group = mh.Group
	}

	s.runWith(group, func() {
		s.processRequest(m, rtype, rname, method, mh)
	})
}

// runWith enqueues the callback, cb, to be called by the worker goroutine
// defined by the worker ID (wid).
func (s *Service) runWith(wid string, cb func()) {
	if atomic.LoadInt32(&s.state) != stateStarted {
		return
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

// With matches the resource ID, rid, with the registered Handlers before
// calling the callback, cb, on the worker goroutine for the resource name or
// group.
//
// With will return an error and not call the callback if there is no matching
// handler found.
func (s *Service) With(rid string, cb func(r Resource)) error {
	r, err := s.Resource(rid)
	if err != nil {
		return err
	}

	s.runWith(r.Group(), func() {
		cb(r)
	})

	return nil
}

// WithResource enqueues the callback, cb, to be called by the resource's worker
// goroutine. If the resource belongs to a group, it will be called on the
// group's worker goroutine.
func (s *Service) WithResource(r Resource, cb func()) {
	s.runWith(r.Group(), cb)
}

// WithGroup calls the callback, cb, on the group's worker goroutine.
func (s *Service) WithGroup(group string, cb func(s *Service)) {
	s.runWith(group, func() { cb(s) })
}

// Resource matches the resource ID, rid, with the registered Handlers and
// returns the resource, or an error if there is no matching handler found.
//
// Should only be called from within the resource's group goroutine. Using the
// returned value from another goroutine may cause race conditions.
func (s *Service) Resource(rid string) (Resource, error) {
	rname, q := parseRID(rid)
	mh := s.GetHandler(rname)
	if mh == nil {
		return nil, fmt.Errorf("res: no matching handlers found for %#v", rid)
	}

	return &resource{
		rname:      rname,
		pathParams: mh.Params,
		query:      q,
		group:      mh.Group,
		s:          s,
		h:          mh.Handler,
		listeners:  mh.Listeners,
	}, nil
}

// event marshals the data and publishes it on a subject, and logs it as an
// outgoing event.
func (s *Service) event(subj string, data interface{}) {
	if data == nil {
		s.rawEvent(subj, nil)
		return
	}

	payload, err := json.Marshal(data)
	if err == nil {
		s.tracef("<-- %s: %s", subj, payload)
		err = s.nc.Publish(subj, payload)
	}
	if err != nil {
		s.errorf("Error sending event %s: %s", subj, err)
	}
}

// rawEvent publishes the payload on a subject, and logs it as an outgoing
// event.
func (s *Service) rawEvent(subj string, payload []byte) {
	s.tracef("<-- %s: %s", subj, payload)
	err := s.nc.Publish(subj, payload)
	if err != nil {
		s.errorf("Error sending event %s: %s", subj, err)
	}
}

// handleReconnect is called when nats has reconnected.
//
// It calls a system.reset to have the resgates update their caches.
func (s *Service) handleReconnect(_ *nats.Conn) {
	s.infof("Reconnected to NATS. Sending reset event.")
	s.ResetAll()
	if s.onReconnect != nil {
		s.onReconnect(s)
	}
}

// handleDisconnect is called when nats is disconnected.
//
// It calls a system.reset to have the resgates update their caches.
func (s *Service) handleDisconnect(_ *nats.Conn) {
	s.infof("Disconnected from NATS.")
	if s.onDisconnect != nil {
		s.onDisconnect(s)
	}
}

func (s *Service) handleClosed(_ *nats.Conn) {
	s.Shutdown()
}

func validateGetHandler(h Handler) {
	if h.Get != nil {
		panic("res: multiple get handlers")
	}
}

// parseRID parses a resource ID, rid, and splits it into the resource name and
// query, if one is available.
//
// The question mark query separator is not included in the returned query
// string.
func parseRID(rid string) (rname string, q string) {
	i := strings.IndexByte(rid, '?')
	if i == -1 {
		return rid, ""
	}

	return rid[:i], rid[i+1:]
}

// processRequest is executed by the worker to process an incoming request.
func (s *Service) processRequest(m *nats.Msg, rtype, rname, method string, mh *Match) {
	var r *Request
	if mh == nil {
		r = &Request{resource: resource{s: s}, msg: m}
		r.reply(responseNotFound)
		return
	}

	var rc resRequest
	if len(m.Data) > 0 {
		err := json.Unmarshal(m.Data, &rc)
		if err != nil {
			r = &Request{resource: resource{s: s}, msg: m}
			s.errorf("Error unmarshaling incoming request: %s", err)
			r.error(ToError(err))
			return
		}
	}

	r = &Request{
		resource: resource{
			rname:      rname,
			pathParams: mh.Params,
			group:      mh.Group,
			s:          s,
			h:          mh.Handler,
			listeners:  mh.Listeners,
			query:      rc.Query,
		},
		rtype:      rtype,
		method:     method,
		msg:        m,
		cid:        rc.CID,
		params:     rc.Params,
		token:      rc.Token,
		header:     rc.Header,
		host:       rc.Host,
		remoteAddr: rc.RemoteAddr,
		uri:        rc.URI,
	}

	r.executeHandler()
}

func (s *Service) queryEventExpire(v interface{}) {
	qe := v.(*queryEvent)
	qe.sub.Drain()
	s.runWith(qe.r.Group(), func() {
		qe.cb(nil)
	})
}
