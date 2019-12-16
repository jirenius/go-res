package test

import (
	"encoding/json"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
)

// Test that the service can be served without error
func TestStart(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, nil)
}

// Test that service can be served without logger
func TestWithoutLogger(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, nil, withLogger(nil))
}

// Test that Logger returns the logger set with SetLogger
func TestServiceLogger(t *testing.T) {
	l := newMemLogger()
	runTest(t, func(s *Session) {
		if s.Logger() != l {
			t.Errorf("expected Logger to return the logger passed to SetLogger, but it didn't")
		}
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, nil, withLogger(l))
}

// Test that With returns an error if there is no registered pattern matching the resource
func TestServiceWithWithoutMatchingPattern(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *Session) {
		err := s.With("test.model", func(r res.Resource) {})
		if err == nil {
			t.Errorf("expected With to return an error, but it didn't")
		}
	})
}

// Test that SetReset sets which resources are reset when calling Reset.
func TestServiceSetReset(t *testing.T) {
	resources := []string{"test.foo.>", "test.bar.>"}
	access := []string{"test.zoo.>", "test.baz.>"}

	s := setup(t, &runConfig{
		logger: newMemLogger(),
		preCallback: func(s *Session) {
			s.SetReset(resources, access)
		},
		resetResources: resources,
		resetAccess:    access,
	})

	teardown(s)
}

// Test that TokenEvent sends a connection token event.
func TestServiceTokenEvent(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *Session) {
		s.TokenEvent(mock.CID, mock.Token)
		s.GetMsg(t).
			AssertSubject(t, "conn."+mock.CID+".token").
			AssertPayload(t, json.RawMessage(`{"token":{"user":"foo","id":42}}`))
	})
}

// Test that TokenEvent with nil sends a connection token event with a nil token.
func TestServiceNilTokenEvent(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *Session) {
		s.TokenEvent(mock.CID, nil)
		s.GetMsg(t).AssertSubject(t, "conn."+mock.CID+".token").AssertPayload(t, json.RawMessage(`{"token":null}`))
	})
}

// Test that TokenEvent with an invalid cid causes panic.
func TestServiceTokenEventWithInvalidCID(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *Session) {
		defer func() {
			v := recover()
			if v == nil {
				t.Fatalf("expected a panic, but nothing happened")
			}
		}()
		s.TokenEvent("invalid.*.cid", nil)
	})
}

// Test that Reset sends a system.reset event.
func TestServiceReset(t *testing.T) {
	tbl := []struct {
		Resources []string
		Access    []string
		Expected  interface{}
	}{
		{nil, nil, nil},
		{[]string{}, nil, nil},
		{nil, []string{}, nil},
		{[]string{}, []string{}, nil},

		{[]string{"test.foo.>"}, nil, json.RawMessage(`{"resources":["test.foo.>"]}`)},
		{nil, []string{"test.foo.>"}, json.RawMessage(`{"access":["test.foo.>"]}`)},
		{[]string{"test.foo.>"}, []string{"test.bar.>"}, json.RawMessage(`{"resources":["test.foo.>"],"access":["test.bar.>"]}`)},

		{[]string{"test.foo.>"}, []string{}, json.RawMessage(`{"resources":["test.foo.>"]}`)},
		{[]string{}, []string{"test.foo.>"}, json.RawMessage(`{"access":["test.foo.>"]}`)},
	}

	for _, l := range tbl {
		runTest(t, func(s *Session) {
			s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		}, func(s *Session) {
			s.Reset(l.Resources, l.Access)
			// Send token event to flush any system.reset event
			s.TokenEvent(mock.CID, nil)

			if l.Expected != nil {
				s.GetMsg(t).
					AssertSubject(t, "system.reset").
					AssertPayload(t, l.Expected)
			}

			s.GetMsg(t).AssertSubject(t, "conn."+mock.CID+".token")
		})
	}
}

// Test OnServe is called on serve
func TestOnServeIsCalledOnServe(t *testing.T) {
	ch := make(chan bool)
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		s.SetOnServe(func(s *res.Service) {
			close(ch)
		})
	}, func(s *Session) {
		select {
		case <-ch:
		case <-time.After(timeoutDuration):
			if t == nil {
				t.Fatal("expected OnServe callback to be called, but it wasn't")
			}
		}
	})
}

// Test OnServe
func TestOnErrorIsCalledOnError(t *testing.T) {
	ch := make(chan bool)
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		s.SetOnError(func(s *res.Service, msg string) {
			close(ch)
		})
		s.FailNextSubscription()
	}, func(s *Session) {
		select {
		case <-ch:
		case <-time.After(timeoutDuration):
			if t == nil {
				t.Fatal("expected OnError callback to be called, but it wasn't")
			}
		}
	}, withoutReset)
}
