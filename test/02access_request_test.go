package test

import (
	"errors"
	"testing"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test that access response is sent on access request
func TestAccess(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.Access(true, "bar")
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertAccess(true, "bar")
	}, restest.WithReset(nil, []string{"test", "test.>"}))
}

// Test that access granted response is sent when calling AccessGranted
func TestAccessGranted(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.AccessGranted()
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertAccess(true, "*")
	})
}

// Test that system.accessDenied response is sent when calling AccessDenied
func TestAccessDenied(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.AccessDenied()
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertError(res.ErrAccessDenied)
	})
}

// Test that calling Error on an access request results in given error
func TestAccessError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.Error(res.ErrMethodNotFound)
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test calling InvalidQuery with no message on an access request results in system.invalidQuery
func TestAccessInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", mock.QueryRequest()).
			Response().
			AssertError(res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on an access request results in system.invalidQuery
func TestAccessInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", mock.QueryRequest()).
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test that panicking in an access request results in system.internalError
func TestPanicOnAccess(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic("panic")
		}))
	}, func(s *restest.Session) {
		for i := 0; i < 10; i++ {
			s.Access("test.model", nil).
				Response().
				AssertErrorCode("system.internalError")
		}
	})
}

// Test that panicking with an Error in an access request results in the given error
func TestPanicWithErrorOnAccess(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic(res.ErrMethodNotFound)
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test that panicing with generic value in an access request results in the given error
func TestPanicWithGenericValueOnAccess(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic(42)
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertErrorCode("system.internalError")
	})
}

// Test that panicing with an error in an access request results in system.internalError
func TestPanicWithOsErrorOnAccess(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic(errors.New("panic"))
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertErrorCode("system.internalError")
	})
}

// Test sending multiple access requests for the same resource
// and assert they are handled in order
func TestMultipleAccess(t *testing.T) {
	const requestCount = 100

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.AccessGranted()
		}))
	}, func(s *restest.Session) {
		reqs := make([]*restest.NATSRequest, requestCount)

		// Test sending access requests
		for i := 0; i < requestCount; i++ {
			reqs[i] = s.Access("test.model", nil)
		}

		for _, req := range reqs {
			req.Response().AssertAccess(true, "*")
		}
	})
}

// Test registering multiple access handlers causes panic.
func TestRegisteringMultipleAccessHandlersPanics(t *testing.T) {
	runTest(t, func(s *res.Service) {
		restest.AssertPanic(t, func() {
			s.Handle("model",
				res.Access(func(r res.AccessRequest) {
					r.NotFound()
				}),
				res.Access(func(r res.AccessRequest) {
					r.NotFound()
				}),
			)
		})
	}, nil, restest.WithoutReset)
}

// Test that access granted response is sent when using AccessGranted handler.
func TestAccessGrantedHandler(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertAccess(true, "*")
	})
}

// Test that system.accessDenied response is sent when using AccessDenied handler.
func TestAccessDeniedHandler(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(res.AccessDenied))
	}, func(s *restest.Session) {
		s.Access("test.model", nil).
			Response().
			AssertError(res.ErrAccessDenied)
	})
}

// Test that an access request without any access handler gives no response
func TestAccess_WithoutAccessHandler_SendsNoResponse(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Model(mock.Model)
		}))
		s.Handle("collection", res.Access(func(r res.AccessRequest) {
			r.AccessGranted()
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", nil)
		// Validate that the response is for the collection access, and not model access
		s.Access("test.collection", nil).Response()
	})
}

// Test that multiple responses to access request causes panic
func TestAccess_WithMultipleResponses_CausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.AccessGranted()
			restest.AssertPanic(t, func() {
				r.AccessDenied()
			})
		}))
	}, func(s *restest.Session) {
		s.Access("test.model", mock.Request()).
			Response().
			AssertAccess(true, "*")
	})
}

func TestAccessRequest_InvalidJSON_RespondsWithInternalError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo",
			res.GetModel(func(r res.ModelRequest) { r.NotFound() }),
			res.Access(res.AccessGranted),
		)
	}, func(s *restest.Session) {
		inb := s.RequestRaw("access.test.model.foo", mock.BrokenJSON)
		s.GetMsg().
			AssertSubject(inb).
			AssertErrorCode(res.CodeInternalError)
	})
}
