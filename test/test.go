package test

import (
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

const timeoutDuration = 1 * time.Second

func syncCallback(cb func(*restest.Session)) func(s *restest.Session, done func()) {
	return func(s *restest.Session, done func()) {
		if cb != nil {
			cb(s)
		}
		done()
	}
}

func runTest(t *testing.T, precb func(*res.Service), cb func(*restest.Session), opts ...func(*restest.SessionConfig)) {
	runTestAsync(t, precb, syncCallback(cb), opts...)
}

func runTestAsync(t *testing.T, precb func(*res.Service), cb func(*restest.Session, func()), opts ...func(*restest.SessionConfig)) {
	rs := res.NewService("test")

	if precb != nil {
		precb(rs)
	}

	s := restest.NewSession(t, rs, opts...)
	defer s.Close()

	acl := make(chan struct{})
	if cb != nil {
		cb(s, func() {
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
}
