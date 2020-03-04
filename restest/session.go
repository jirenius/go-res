package restest

import (
	"log"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/logger"
)

// DefaultTimeoutDuration is the duration the session awaits any message before
// timing out.
const DefaultTimeoutDuration = 1 * time.Second

// Session represents a test session with a res server
type Session struct {
	*MockConn
	s          *res.Service
	cfg        *SessionConfig
	cl         chan struct{}
	logPrinted bool
}

// SessionConfig represents the configuration for a session.
type SessionConfig struct {
	TestName         string
	KeepLogger       bool
	NoReset          bool
	ValidateReset    bool
	ResetResources   []string
	ResetAccess      []string
	FailSubscription bool
	MockConnConfig
}

// NewSession creates a new Session and connects the service to a mock NATS
// connection.
//
// A service logger will by default be set to a new MemLogger. To set any other
// logger, add the option:
//  WithLogger(logger)
//
// If the tests sends any query event, a real NATS instance is required, which
// is slower than using the default mock connection. To use a real NATS
// instance, add the option:
//  WithGnatsd
func NewSession(t *testing.T, service *res.Service, opts ...func(*SessionConfig)) *Session {
	cfg := &SessionConfig{
		MockConnConfig: MockConnConfig{TimeoutDuration: DefaultTimeoutDuration},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	c := NewMockConn(t, &cfg.MockConnConfig)
	s := &Session{
		MockConn: c,
		s:        service,
		cl:       make(chan struct{}),
		cfg:      cfg,
	}

	if cfg.FailSubscription {
		c.FailNextSubscription()
	}

	if !cfg.KeepLogger {
		service.SetLogger(logger.NewMemLogger().SetTrace(true).SetFlags(log.Ltime))
	}

	go func() {
		defer s.StopServer()
		defer close(s.cl)
		if err := s.s.Serve(c); err != nil {
			panic("test: failed to start service: " + err.Error())
		}
	}()

	if !s.cfg.NoReset {
		msg := s.GetMsg()
		if s.cfg.ValidateReset {
			msg.AssertSystemReset(cfg.ResetResources, cfg.ResetAccess)
		} else {
			msg.AssertSubject("system.reset")
		}
	}

	return s
}

// Service returns the associated res.Service.
func (s *Session) Service() *res.Service {
	return s.s
}

// Close closes the session.
func (s *Session) Close() error {
	// Check for panics
	e := recover()
	defer func() {
		// Re-panic
		if e != nil {
			panic(e)
		}
	}()
	// Output memlog if test failed or we are panicking
	if e != nil || s.t.Failed() {
		s.printLog()
	}

	// Try to shutdown the service
	ch := make(chan error)
	go func() {
		ch <- s.s.Shutdown()
	}()

	// Await the closing
	var err error
	select {
	case err = <-ch:
	case <-time.After(s.cfg.TimeoutDuration):
		s.t.Fatalf("failed to shutdown service: timeout")
	}
	return err
}

// WithKeepLogger sets the KeepLogger option, to prevent Session to override the
// service logger with its own MemLogger.
func WithKeepLogger(cfg *SessionConfig) { cfg.KeepLogger = true }

// WithGnatsd sets the UseGnatsd option to use a real NATS instance.
//
// This option should be set if the test involves query events.
func WithGnatsd(cfg *SessionConfig) { cfg.UseGnatsd = true }

// WithTest sets the TestName option.
//
// The test name will be outputted when logging test errors.
func WithTest(name string) func(*SessionConfig) {
	return func(cfg *SessionConfig) { cfg.TestName = name }
}

// WithoutReset sets the NoReset option to not expect an initial system.reset
// event on server start.
func WithoutReset(cfg *SessionConfig) { cfg.NoReset = true }

// WithFailSubscription sets FailSubscription to make first subscription to fail.
func WithFailSubscription(cfg *SessionConfig) { cfg.FailSubscription = true }

// WithReset sets the ValidateReset option to validate that the system.reset
// includes the specific access and resources strings.
func WithReset(resources []string, access []string) func(*SessionConfig) {
	return func(cfg *SessionConfig) {
		cfg.ResetResources = resources
		cfg.ResetAccess = access
		cfg.ValidateReset = true
	}
}

func (s *Session) printLog() {
	if s.logPrinted {
		return
	}
	s.logPrinted = true
	if s.cfg.TestName != "" {
		s.t.Logf("Failed test %s", s.cfg.TestName)
	}
	// Print log if we have a MemLogger
	if l, ok := s.s.Logger().(*logger.MemLogger); ok {
		s.t.Logf("Trace log:\n%s", l)
	}
}
