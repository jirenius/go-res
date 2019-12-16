package test

import (
	"log"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/logger"
)

const timeoutDuration = 100 * time.Second

// Session represents a test session with a res server
type Session struct {
	*MockConn
	*res.Service
	cl chan struct{}
}

func teardown(s *Session) {
	err := s.Shutdown()

	// Check error, as an error means that server hasn't had
	// time to start. We can then ignore waiting for the closing
	if err == nil {
		select {
		case <-s.cl:
		case <-time.After(timeoutDuration):
			panic("test: failed to shutdown service: timeout")
		}
	}
}

func setup(t *testing.T, cfg *runConfig) *Session {
	var s *Session
	c := NewTestConn(cfg.useGnatsd)
	r := res.NewService("test")
	r.SetLogger(cfg.logger)

	s = &Session{
		MockConn: c,
		Service:  r,
		cl:       make(chan struct{}),
	}

	if cfg.preCallback != nil {
		cfg.preCallback(s)
	}

	go func() {
		defer s.StopServer()
		defer close(s.cl)
		if err := r.Serve(c); err != nil {
			panic("test: failed to start service: " + err.Error())
		}
	}()

	if !cfg.noReset {
		ev := s.GetMsg(t).AssertSubject(t, "system.reset")
		if cfg.validateReset {
			m := make(map[string]interface{}, 2)
			if len(cfg.resetResources) > 0 {
				m["resources"] = cfg.resetResources
			}
			if len(cfg.resetAccess) > 0 {
				m["access"] = cfg.resetAccess
			}
			ev.AssertPayload(t, m)
		}
	}

	return s
}

func syncCallback(cb func(*Session)) func(s *Session, done func()) {
	return func(s *Session, done func()) {
		if cb != nil {
			cb(s)
		}
		done()
	}
}

type runConfig struct {
	name           string
	logger         logger.Logger
	preCallback    func(*Session)
	callback       func(*Session, func())
	useGnatsd      bool
	serveError     bool
	noReset        bool
	validateReset  bool
	resetResources []string
	resetAccess    []string
}

func callback(cb func(*Session)) func(*runConfig) {
	return func(cfg *runConfig) { cfg.callback = syncCallback(cb) }
}

func asyncCallback(cb func(s *Session, done func())) func(*runConfig) {
	return func(cfg *runConfig) { cfg.callback = cb }
}

func withName(name string) func(*runConfig) {
	return func(cfg *runConfig) { cfg.name = name }
}

func withLogger(l logger.Logger) func(*runConfig) {
	return func(cfg *runConfig) { cfg.logger = l }
}

func withGnatsd(cfg *runConfig) { cfg.useGnatsd = true }

func withError(cfg *runConfig) { cfg.serveError = true }

func withoutReset(cfg *runConfig) { cfg.noReset = true }

func withResources(resources []string) func(*runConfig) {
	return func(cfg *runConfig) {
		cfg.resetResources = resources
		cfg.validateReset = true
	}
}

func withAccess(access []string) func(*runConfig) {
	return func(cfg *runConfig) {
		cfg.resetAccess = access
		cfg.validateReset = true
	}
}

func runTest(t *testing.T, precb func(*Session), cb func(*Session), opts ...func(*runConfig)) {
	runTestAsync(t, precb, syncCallback(cb), opts...)
}

func newMemLogger() *logger.MemLogger {
	return logger.NewMemLogger().SetTrace(true).SetFlags(log.Ltime)
}

func runTestAsync(t *testing.T, precb func(*Session), cb func(*Session, func()), opts ...func(*runConfig)) {
	cfg := &runConfig{
		logger:         newMemLogger(),
		preCallback:    precb,
		callback:       cb,
		useGnatsd:      false,
		resetResources: nil,
		resetAccess:    nil,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	runTestInternal(t, cfg)
}

func runTestInternal(t *testing.T, cfg *runConfig) {
	s := setup(t, cfg)

	panicked := true
	defer func() {
		if panicked || t.Failed() {
			if cfg.name != "" {
				t.Logf("Failed test %s", cfg.name)
			}
			l := s.Logger()
			if l != nil {
				t.Logf("Trace log:\n%s", l)
			}
		}
	}()

	acl := make(chan struct{})
	if cfg.callback != nil {
		cfg.callback(s, func() {
			close(acl)
		})
	} else {
		close(acl)
	}

	select {
	case <-acl:
	case <-time.After(timeoutDuration):
		panic("test: async test failed by never calling done: timeout")
	}

	teardown(s)
	panicked = false
}
