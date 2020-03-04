package test

import (
	"encoding/json"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/logger"
	"github.com/jirenius/go-res/restest"
)

// TestService is the doc.go usage example
func TestService(t *testing.T) {
	// Create service to test
	s := res.NewService("foo")
	s.Handle("bar.$id",
		res.Access(res.AccessGranted),
		res.GetModel(func(r res.ModelRequest) {
			r.Model(struct {
				Message string `json:"msg"`
			}{r.PathParam("id")})
		}),
	)

	// Create test session
	c := restest.NewSession(t, s)
	defer c.Close()

	// Test sending get request and validate response
	c.Get("foo.bar.42").
		Response().
		AssertModel(map[string]string{"msg": "42"})
}

// Test that the service returns the correct protocol version
func TestServiceProtocolVersion(t *testing.T) {
	s := res.NewService("test")
	restest.AssertEqualJSON(t, "ProtocolVersion()", s.ProtocolVersion(), "1.2.0")
}

// Test that the service can be served without error
func TestServiceStart(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, nil)
}

// Test that service can be served without logger
func TestServiceWithoutLogger(t *testing.T) {
	s := res.NewService("test")
	s.SetLogger(nil)
	s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	session := restest.NewSession(t, s, restest.WithKeepLogger)
	defer session.Close()
}

// Test that Logger returns the logger set with SetLogger
func TestServiceSetLogger(t *testing.T) {
	s := res.NewService("test")
	l := logger.NewMemLogger()
	s.SetLogger(l)
	if s.Logger() != l {
		t.Errorf("expected Logger to return the logger passed to SetLogger, but it didn't")
	}
	s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))

	session := restest.NewSession(t, s, restest.WithKeepLogger)
	defer session.Close()
}

// Test that With returns an error if there is no registered pattern matching the resource
func TestServiceWith_WithoutMatchingPattern(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		err := s.Service().With("test.model", func(r res.Resource) {})
		if err == nil {
			t.Errorf("expected With to return an error, but it didn't")
		}
	})
}

// Test that SetOwnedResources sets which resources are reset when calling Reset.
func TestServiceSetOwnedResources(t *testing.T) {
	resources := []string{"test.foo.>", "test.bar.>"}
	access := []string{"test.zoo.>", "test.baz.>"}

	runTest(t, func(s *res.Service) {
		s.SetOwnedResources(resources, access)
	}, nil, restest.WithReset(resources, access))
}

// Test that TokenEvent sends a connection token event.
func TestServiceTokenEvent_WithObjectToken_SendsToken(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		s.Service().TokenEvent(mock.CID, mock.Token)
		s.GetMsg().AssertTokenEvent(mock.CID, mock.Token)
	})
}

// Test that TokenEvent with nil sends a connection token event with a nil token.
func TestServiceTokenEvent_WithNilToken_SendsNilToken(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		s.Service().TokenEvent(mock.CID, nil)
		s.GetMsg().AssertTokenEvent(mock.CID, nil)
	})
}

// Test that TokenEvent with an invalid cid causes panic.
func TestServiceTokenEvent_WithInvalidCID_CausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		restest.AssertPanic(t, func() {
			s.Service().TokenEvent("invalid.*.cid", nil)
		})
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
		runTest(t, func(s *res.Service) {
			s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		}, func(s *restest.Session) {
			s.Service().Reset(l.Resources, l.Access)
			// Send token event to flush any system.reset event
			s.Service().TokenEvent(mock.CID, nil)

			if l.Expected != nil {
				s.GetMsg().
					AssertSubject("system.reset").
					AssertPayload(l.Expected)
			}

			s.GetMsg().AssertTokenEvent(mock.CID, nil)
		})
	}
}

func TestServiceSetOnServe_ValidCallback_IsCalledOnServe(t *testing.T) {
	ch := make(chan bool)
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		s.SetOnServe(func(s *res.Service) {
			close(ch)
		})
	}, func(s *restest.Session) {
		select {
		case <-ch:
		case <-time.After(timeoutDuration):
			if t == nil {
				t.Fatal("expected OnServe callback to be called, but it wasn't")
			}
		}
	})
}

func TestServiceSetOnError_ValidCallback_IsCalledOnError(t *testing.T) {
	var done func()
	runTestAsync(t, func(s *res.Service) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		s.SetOnError(func(s *res.Service, msg string) {
			done()
		})
	}, func(s *restest.Session, d func()) {
		done = d
	}, restest.WithFailSubscription, restest.WithoutReset)
}

func TestServiceResource_WithMatchingResource_ReturnsResource(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		resource, err := s.Service().Resource("test.model.foo")
		restest.AssertNoError(t, err)
		restest.AssertNotNil(t, resource)
		restest.AssertEqualJSON(t, "ResourceName", resource.ResourceName(), "test.model.foo")
		restest.AssertEqualJSON(t, "PathParams", resource.PathParams(), map[string]string{"id": "foo"})
	})
}

func TestServiceResource_WithNonMatchingResource_ReturnsError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		resource, err := s.Service().Resource("test.model")
		restest.AssertError(t, err)
		restest.AssertNil(t, resource)
	})
}

func TestServiceWithResource_WithMatchingResource_CallsCallback(t *testing.T) {
	ch := make(chan bool)
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Group("foo"), res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		resource, err := s.Service().Resource("test.model")
		restest.AssertNoError(t, err)
		s.Service().WithResource(resource, func() {
			close(ch)
		})
		select {
		case <-ch:
		case <-time.After(timeoutDuration):
			if t == nil {
				t.Fatal("expected WithResource callback to be called, but it wasn't")
			}
		}
	})
}

func TestServiceWithGroup_WithMatchingResource_CallsCallback(t *testing.T) {
	ch := make(chan bool)
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Group("foo"), res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		s.Service().WithGroup("foo", func(serv *res.Service) {
			restest.AssertTrue(t, "param to be service instance", serv == s.Service())
			close(ch)
		})
		select {
		case <-ch:
		case <-time.After(timeoutDuration):
			if t == nil {
				t.Fatal("expected WithGroup callback to be called, but it wasn't")
			}
		}
	})
}
