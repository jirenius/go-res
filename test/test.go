package test

import (
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/resgate/logger"
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

func setup(t *testing.T, l logger.Logger, precb func(s *Session)) *Session {
	var s *Session
	c := NewTestConn()
	r := res.NewService("test")
	r.SetLogger(l)

	s = &Session{
		MockConn: c,
		Service:  r,
		cl:       make(chan struct{}),
	}

	if precb != nil {
		precb(s)
	}

	go func() {
		defer close(s.cl)
		if err := r.Serve(c); err != nil {
			panic("test: failed to start service: " + err.Error())
		}
	}()
	s.GetMsg(t).AssertSubject(t, "system.reset")

	return s
}

func runTest(t *testing.T, precb func(s *Session), cb func(s *Session)) {
	runTestWithLogger(t, newMemLogger(true, true), precb, cb)
}

func runTestWithLogger(t *testing.T, l logger.Logger, precb func(s *Session), cb func(s *Session)) {
	s := setup(t, l, precb)

	panicked := true
	defer func() {
		if panicked {
			l := s.Logger()
			if l != nil {
				t.Logf("Trace log:\n%s", l)
			}
		}
	}()

	if cb != nil {
		cb(s)
	}

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

	panicked = false
}

func runTestAsync(t *testing.T, precb func(s *Session), cb func(s *Session, done func())) {
	s := setup(t, newMemLogger(true, true), precb)

	panicked := true
	defer func() {
		if panicked {
			l := s.Logger()
			if l != nil {
				t.Logf("Trace log:\n%s", l)
			}
		}
	}()

	acl := make(chan struct{})
	if cb != nil {
		cb(s, func() {
			close(acl)
		})
	}

	select {
	case <-acl:
	case <-time.After(timeoutDuration):
		panic("test: async test failed by never calling done: timeout")
	}

	teardown(s)
	panicked = false
}
