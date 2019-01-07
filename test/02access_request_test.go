package test

import (
	"encoding/json"
	"testing"

	"github.com/jirenius/go-res"
)

// Test that access response is sent on access request
func TestAccess(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.Access(true, "bar")
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"get":true,"call":"bar"}}`))
	})
}

// Test that access granted response is sent when calling AccessGranted
func TestAccessGranted(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.AccessGranted()
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"get":true,"call":"*"}}`))
	})
}

// Test that system.accessDenied response is sent when calling AccessDenied
func TestAccessDenied(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.AccessDenied()
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrAccessDenied)
	})
}

// Test that calling Error on an access request results in given error
func TestAccessError(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.Error(res.ErrMethodNotFound)
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test that panicing in an access request results in system.internalError
func TestPanicOnAccess(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic("Panic!")
		}))
	}, func(s *Session) {
		for i := 0; i < 10; i++ {
			inb := s.Request("access.test.model", nil)
			s.GetMsg(t).
				AssertSubject(t, inb).
				AssertErrorCode(t, "system.internalError")
		}
	})
}

// Test that panicing with an Error in am access request results in the given error
func TestPanicWithErrorOnAccess(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic(res.ErrMethodNotFound)
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test sending multiple access requests for the same resource
// and assert they are handled in order
func TestMultipleAccess(t *testing.T) {
	const requestCount = 100

	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.AccessGranted()
		}))
	}, func(s *Session) {
		inbs := make([]string, requestCount)

		// Test getting the model
		for i := 0; i < requestCount; i++ {
			inbs[i] = s.Request("access.test.model", nil)
		}

		for _, inb := range inbs {
			s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"get":true,"call":"*"}}`))
		}
	})
}

// Test registering multiple access handlers causes panic.
func TestRegisteringMultipleAccessHandlersPanics(t *testing.T) {
	runTest(t, func(s *Session) {
		defer func() {
			v := recover()
			if v == nil {
				t.Errorf(`expected test to panic, but nothing happened`)
			}
		}()
		s.Handle("model",
			res.Access(func(r res.AccessRequest) {
				r.NotFound()
			}),
			res.Access(func(r res.AccessRequest) {
				r.NotFound()
			}),
		)
	}, nil)
}

// Test that access granted response is sent when using AccessGranted handler.
func TestAccessGrantedHandler(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"get":true,"call":"*"}}`))
	})
}

// Test that system.accessDenied response is sent when using AccessDenied handler.
func TestAccessDeniedHandler(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(res.AccessDenied))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrAccessDenied)
	})
}
