package test

import (
	"testing"
	"time"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test auth OK response with result
func TestAuthOK(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(mock.Result)
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertResult(mock.Result)
	})
}

// Test AuthRequest getter methods
func TestAuthRequestGetters(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("foo", func(r res.AuthRequest) {
			restest.AssertEqualJSON(t, "Method", r.Method(), "foo")
			restest.AssertEqualJSON(t, "CID", r.CID(), mock.CID)
			restest.AssertEqualJSON(t, "Header", r.Header(), mock.Header)
			restest.AssertEqualJSON(t, "Host", r.Host(), mock.Host)
			restest.AssertEqualJSON(t, "RemoteAddr", r.RemoteAddr(), mock.RemoteAddr)
			restest.AssertEqualJSON(t, "URI", r.URI(), mock.URI)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "foo", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test auth OK response with nil result
func TestAuthWithNil(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertResult(nil)
	})
}

// Test auth Resource response with valid resource ID
func TestAuthResource_WithValidRID_SendsResourceResponse(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.Resource("test.foo")
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertResource("test.foo")
	})
}

// Test auth Resource response with invalid resource ID causes panic
func TestAuthResource_WithInvalidRID_CausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			restest.AssertPanicNoRecover(t, func() {
				r.Resource("test..foo")
			})
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertErrorCode(res.CodeInternalError)
	})
}

// Test calling NotFound on a auth request results in system.notFound
func TestAuthNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling MethodNotFound on a auth request results in system.methodNotFound
func TestAuthMethodNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.MethodNotFound()
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test calling InvalidParams with no message on a auth request results in system.invalidParams
func TestAuthInvalidParams_NoMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidParams("")
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrInvalidParams)
	})
}

// Test calling InvalidParams on a auth request results in system.invalidParams
func TestAuthInvalidParams_CustomMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidParams("foo")
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidParams,
				Message: "foo",
			})
	})
}

// Test calling InvalidQuery with no message on a auth request results in system.invalidQuery
func TestAuthInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on a auth request results in system.invalidQuery
func TestAuthInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test calling Error on a auth request results in given error
func TestAuthError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.Error(res.ErrTimeout)
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrTimeout)
	})
}

// Test calling RawParams on a auth request with parameters
func TestAuthRawParams(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			restest.AssertEqualJSON(t, "RawParams", r.RawParams(), mock.Params)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.AuthRequest()
		req.Params = mock.Params
		s.Auth("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawParams on a auth request with no parameters
func TestAuthRawParamsWithNilParams(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			restest.AssertEqualJSON(t, "RawParams", r.RawParams(), nil)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawToken on a auth request with token
func TestAuthRawToken(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			restest.AssertEqualJSON(t, "RawToken", r.RawToken(), mock.Token)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.AuthRequest()
		req.Token = mock.Token
		s.Auth("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawToken on a auth request with no token
func TestAuthRawTokenWithNoToken(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			restest.AssertEqualJSON(t, "RawToken", r.RawToken(), nil)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseParams on a auth request with parameters
func TestAuthParseParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.ParseParams(&p)
			restest.AssertEqualJSON(t, "p.Foo", p.Foo, "bar")
			restest.AssertEqualJSON(t, "p.Baz", p.Baz, 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.AuthRequest()
		req.Params = mock.Params
		s.Auth("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseParams on a auth request with no parameters
func TestAuthParseParamsWithNilParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.ParseParams(&p)
			restest.AssertEqualJSON(t, "p.Foo", p.Foo, "")
			restest.AssertEqualJSON(t, "p.Baz", p.Baz, 0)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseToken on a auth request with token
func TestAuthParseToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.ParseToken(&o)
			restest.AssertEqualJSON(t, "o.User", o.User, "foo")
			restest.AssertEqualJSON(t, "o.ID", o.ID, 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.AuthRequest()
		req.Token = mock.Token
		s.Auth("test.model", "method", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseToken on a auth request with no token
func TestAuthParseTokenWithNilToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.ParseToken(&o)
			restest.AssertEqualJSON(t, "o.User", o.User, "")
			restest.AssertEqualJSON(t, "o.ID", o.ID, 0)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test that registering auth methods with duplicate names causes panic
func TestRegisteringDuplicateAuthMethodPanics(t *testing.T) {
	runTest(t, func(s *res.Service) {
		restest.AssertPanic(t, func() {
			s.Handle("model",
				res.Auth("foo", func(r res.AuthRequest) {
					r.OK(nil)
				}),
				res.Auth("bar", func(r res.AuthRequest) {
					r.OK(nil)
				}),
				res.Auth("foo", func(r res.AuthRequest) {
					r.OK(nil)
				}),
			)
		})
	}, nil, restest.WithoutReset)
}

// Test that Timeout sends the pre-response with timeout
func TestAuthRequestTimeout(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := s.Auth("test.model", "method", nil)
		req.Response().AssertRawPayload([]byte(`timeout:"42000"`))
		req.Response().AssertError(res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero
func TestAuthRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
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
		s.Auth("test.model", "method", nil).
			Response().
			AssertErrorCode("system.internalError")
	})
}

// Test that TokenEvent sends a connection token event.
func TestAuthRequestTokenEvent(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.TokenEvent(mock.Token)
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		req := s.Auth("test.model", "method", nil)
		s.GetMsg().
			AssertTokenEvent(mock.CID, mock.Token)
		req.Response().
			AssertResult(nil)
	})
}

// Test that TokenEvent with nil sends a connection token event with a nil token.
func TestAuthRequestNilTokenEvent(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.TokenEvent(nil)
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		req := s.Auth("test.model", "method", nil)
		s.GetMsg().
			AssertTokenEvent(mock.CID, nil)
		req.Response().
			AssertResult(nil)
	})
}

// Test auth request with an unset method returns error system.methodNotFound
func TestAuthRequest_UnknownMethod_ErrorMethodNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "unset", nil).
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test that multiple responses to auth request causes panic
func TestAuth_WithMultipleResponses_CausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(nil)
			restest.AssertPanic(t, func() {
				r.MethodNotFound()
			})
		}))
	}, func(s *restest.Session) {
		s.Auth("test.model", "method", mock.Request()).
			Response().
			AssertResult(nil)
	})
}

func TestAuthRequest_InvalidJSON_RespondsWithInternalError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo",
			res.Auth("method", func(r res.AuthRequest) { r.OK(nil) }),
		)
	}, func(s *restest.Session) {
		inb := s.RequestRaw("auth.test.model.foo.method", mock.BrokenJSON)
		s.GetMsg().
			AssertSubject(inb).
			AssertErrorCode(res.CodeInternalError)
	})
}
