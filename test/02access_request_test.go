package test

import (
	"encoding/json"
	"errors"
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
	}, withAccess([]string{"test", "test.>"}))
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

// Test calling InvalidQuery with no message on an access request results in system.invalidQuery
func TestAccessInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", mock.QueryRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on an access request results in system.invalidQuery
func TestAccessInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", mock.QueryRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, &res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test that panicing in an access request results in system.internalError
func TestPanicOnAccess(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic("panic")
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

// Test that panicing with an Error in an access request results in the given error
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

// Test that panicing with generic value in an access request results in the given error
func TestPanicWithGenericValueOnAccess(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic(42)
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertErrorCode(t, "system.internalError")
	})
}

// Test that panicing with an error in an access request results in system.internalError
func TestPanicWithOsErrorOnAccess(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			panic(errors.New("panic"))
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertErrorCode(t, "system.internalError")
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
		AssertPanic(t, func() {
			s.Handle("model",
				res.Access(func(r res.AccessRequest) {
					r.NotFound()
				}),
				res.Access(func(r res.AccessRequest) {
					r.NotFound()
				}),
			)
		})
	}, nil, withoutReset)
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

// Test that an access request without any access handler gives no response
func TestAccess_WithoutAccessHandler_SendsNoResponse(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Model(mock.Model)
		}))
		s.Handle("collection", res.Access(func(r res.AccessRequest) {
			r.AccessGranted()
		}))
	}, func(s *Session) {
		s.Request("access.test.model", mock.DefaultRequest())
		inb := s.Request("access.test.collection", mock.DefaultRequest())
		// Validate that the response is for the collection access, and not model access
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test that multiple responses to access request causes panic
func TestAccess_WithMultipleResponses_CausesPanic(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(func(r res.AccessRequest) {
			r.AccessGranted()
			AssertPanic(t, func() {
				r.AccessDenied()
			})
		}))
	}, func(s *Session) {
		inb := s.Request("access.test.model", mock.Request())
		s.GetMsg(t).Equals(t, inb, mock.AccessGrantedResponse)
	})
}

func TestAccessRequest_InvalidJSON_RespondsWithInternalError(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model.foo",
			res.GetModel(func(r res.ModelRequest) { r.NotFound() }),
			res.Access(res.AccessGranted),
		)
	}, func(s *Session) {
		inb := s.RequestRaw("access.test.model.foo", mock.BrokenJSON)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertErrorCode(t, res.CodeInternalError)
	})
}
