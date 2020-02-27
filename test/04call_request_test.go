package test

import (
	"testing"
	"time"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test call OK response with result
func TestCallOK(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.OK(mock.Result)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertPayload(mock.ResultResponse)
	})
}

// Test CallRequest getter methods
func TestCallRequestGetters(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("foo", func(r res.CallRequest) {
			restest.AssertEqualJSON(t, "Method", r.Method(), "foo")
			restest.AssertEqualJSON(t, "CID", r.CID(), mock.CID)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "foo", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test call OK response with nil result
func TestCallOKWithNil(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertResult(nil)
	})
}

// Test call Resource response with valid resource ID
func TestCallResource_WithValidRID_SendsResourceResponse(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.Resource("test.foo")
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertResource("test.foo")
	})
}

// Test call Resource response with invalid resource ID causes panic
func TestCallResource_WithInvalidRID_CausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			restest.AssertPanicNoRecover(t, func() {
				r.Resource("test..foo")
			})
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertErrorCode(res.CodeInternalError)
	})
}

// Test calling NotFound on a call request results in system.notFound
func TestCallNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling MethodNotFound on a call request results in system.methodNotFound
func TestCallMethodNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.MethodNotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test calling InvalidParams with no message on a call request results in system.invalidParams
func TestCallInvalidParams_EmptyMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidParams("")
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertError(res.ErrInvalidParams)
	})
}

// Test calling InvalidParams on a call request results in system.invalidParams
func TestCallInvalidParams_CustomMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidParams(mock.ErrorMessage)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidParams,
				Message: mock.ErrorMessage,
			})
	})
}

// Test calling InvalidQuery with no message on a call request results in system.invalidQuery
func TestCallInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", mock.Request()).
			Response().
			AssertError(res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on a call request results in system.invalidQuery
func TestCallInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test calling Error on a call request results in given error
func TestCallError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.Error(res.ErrTimeout)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertError(res.ErrTimeout)
	})
}

// Test calling RawParams on a call request with parameters
func TestCallRawParams(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			restest.AssertEqualJSON(t, "RawParams", r.RawParams(), mock.Params)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		req.Params = mock.Params
		s.Call("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawParams on a call request with no parameters
func TestCallRawParamsWithNilParams(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			restest.AssertEqualJSON(t, "RawParams", r.RawParams(), nil)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		s.Call("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawToken on a call request with token
func TestCallRawToken(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			restest.AssertEqualJSON(t, "RawToken", r.RawToken(), mock.Token)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		req.Token = mock.Token
		s.Call("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawToken on a call request with no token
func TestCallRawTokenWithNoToken(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			restest.AssertEqualJSON(t, "RawToken", r.RawToken(), nil)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		s.Call("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseParams on a call request with parameters
func TestCallParseParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ParseParams(&p)
			restest.AssertEqualJSON(t, "p.Foo", p.Foo, "bar")
			restest.AssertEqualJSON(t, "p.Baz", p.Baz, 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		req.Params = mock.Params
		s.Call("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseParams on a call request with no parameters
func TestCallParseParamsWithNilParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ParseParams(&p)
			restest.AssertEqualJSON(t, "p.Foo", p.Foo, "")
			restest.AssertEqualJSON(t, "p.Baz", p.Baz, 0)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		s.Call("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseToken on a call request with token
func TestCallParseToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ParseToken(&o)
			restest.AssertEqualJSON(t, "o.User", o.User, "foo")
			restest.AssertEqualJSON(t, "o.ID", o.ID, 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		req.Token = mock.Token
		s.Call("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseToken on a call request with no token
func TestCallParseTokenWithNilToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ParseToken(&o)
			restest.AssertEqualJSON(t, "o.User", o.User, "")
			restest.AssertEqualJSON(t, "o.ID", o.ID, 0)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		s.Call("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test set call response with result
func TestSetCall(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Set(func(r res.CallRequest) {
			r.OK(mock.Result)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "set", nil).
			Response().
			AssertResult(mock.Result)
	})
}

// Test that registering call methods with duplicate names causes panic
func TestRegisteringDuplicateCallMethodPanics(t *testing.T) {
	runTest(t, func(s *res.Service) {
		restest.AssertPanic(t, func() {
			s.Handle("model",
				res.Call("foo", func(r res.CallRequest) {
					r.OK(nil)
				}),
				res.Call("bar", func(r res.CallRequest) {
					r.OK(nil)
				}),
				res.Call("foo", func(r res.CallRequest) {
					r.OK(nil)
				}),
			)
		})
	}, nil, restest.WithoutReset)
}

// Test that Timeout sends the pre-response with timeout
func TestCallRequestTimeout(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := s.Call("test.model", "method", nil)
		req.Response().AssertRawPayload([]byte(`timeout:"42000"`))
		req.Response().AssertError(res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero
func TestCallRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			panicked := true
			defer func() {
				if !panicked {
					t.Errorf("expected Timeout to panic, but nothing happened")
				}
			}()
			r.Timeout(-time.Millisecond * 10)
			r.NotFound()
			panicked = false
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertErrorCode("system.internalError")
	})
}

// Test call request with an unset method returns error system.methodNotFound
func TestCallRequest_UnknownMethod_ErrorMethodNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "unset", nil).
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test that multiple responses to call request causes panic
func TestCall_WithMultipleResponses_CausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.OK(nil)
			restest.AssertPanic(t, func() {
				r.MethodNotFound()
			})
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", mock.Request()).
			Response().
			AssertResult(nil)
	})
}

func TestCallRequest_InvalidJSON_RespondsWithInternalError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo",
			res.Call("method", func(r res.CallRequest) { r.OK(nil) }),
		)
	}, func(s *restest.Session) {
		inb := s.RequestRaw("call.test.model.foo.method", mock.BrokenJSON)
		s.GetMsg().
			AssertSubject(inb).
			AssertErrorCode(res.CodeInternalError)
	})
}
