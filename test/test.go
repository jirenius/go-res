package test

import (
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/resgate/logger"
)

// Session represents a test session with a res server
type Session struct {
	*TestConn
	*res.Service
	cl chan struct{}
}

func teardown(s *Session) {
	err := s.Shutdown()
	// Check error, as it means that server hasn't had
	// time to start. We can then ignore waiting for the closing
	if err != nil {
		return
	}
	<-s.cl
}

func runTest(t *testing.T, precb func(s *Session), cb func(s *Session)) {
	var s *Session
	l := logger.NewMemLogger(true, true)
	c := NewTestConn()
	r := res.NewService("test")
	r.SetLogger(l)

	s = &Session{
		TestConn: c,
		Service:  r,
		cl:       make(chan struct{}),
	}

	if precb != nil {
		precb(s)
	}

	panicked := true
	defer func() {
		if panicked {
			t.Logf("Trace log:\n%s", l)
		}
	}()

	go func() {
		defer close(s.cl)
		if err := r.Serve(c); err != nil {
			panic("test: failed to start service: " + err.Error())
		}
	}()
	s.GetMsg(t).AssertSubject(t, "system.reset")

	if cb != nil {
		cb(s)
	}

	err := s.Shutdown()

	// Check error, as an error means that server hasn't had
	// time to start. We can then ignore waiting for the closing
	if err == nil {
		select {
		case <-s.cl:
		case <-time.After(3 * time.Second):
			panic("test: failed to shutdown service: timeout")
		}
	}

	panicked = false
}
