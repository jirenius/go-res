package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jirenius/go-res"
)

// Test auth OK response with result
func TestAuthOK(t *testing.T) {
	result := `{"foo":"bar","zoo":42}`

	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(json.RawMessage(result))
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":`+result+`}`))
	})
}

// Test AuthRequest getter methods
func TestAuthRequestGetters(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("foo", func(r res.AuthRequest) {
			AssertEqual(t, "Method", r.Method(), "foo")
			AssertEqual(t, "CID", r.CID(), mock.CID)
			AssertEqual(t, "Header", r.Header(), mock.Header)
			AssertEqual(t, "Host", r.Host(), mock.Host)
			AssertEqual(t, "RemoteAddr", r.RemoteAddr(), mock.RemoteAddr)
			AssertEqual(t, "URI", r.URI(), mock.URI)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.foo", mock.AuthRequest())
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test auth OK response with nil result
func TestAuthWithNil(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(nil)
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":null}`))
	})
}

// Test auth Resource response with valid resource ID
func TestAuthResource_WithValidRID_SendsResourceResponse(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.Resource("test.foo")
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"resource":{"rid":"test.foo"}}`))
	})
}

// Test auth Resource response with invalid resource ID causes panic
func TestAuthResource_WithInvalidRID_CausesPanic(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			AssertPanicNoRecover(t, func() {
				r.Resource("test..foo")
			})
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, res.CodeInternalError)
	})
}

// Test calling NotFound on a auth request results in system.notFound
func TestAuthNotFound(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling MethodNotFound on a auth request results in system.methodNotFound
func TestAuthMethodNotFound(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.MethodNotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test calling InvalidParams with no message on a auth request results in system.invalidParams
func TestAuthInvalidParams_NoMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidParams("")
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrInvalidParams)
	})
}

// Test calling InvalidParams on a auth request results in system.invalidParams
func TestAuthInvalidParams_CustomMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidParams("foo")
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, &res.Error{
				Code:    res.CodeInvalidParams,
				Message: "foo",
			})
	})
}

// Test calling InvalidQuery with no message on a auth request results in system.invalidQuery
func TestAuthInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on a auth request results in system.invalidQuery
func TestAuthInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, &res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test calling Error on a auth request results in given error
func TestAuthError(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.Error(res.ErrTimeout)
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrTimeout)
	})
}

// Test calling RawParams on a auth request with parameters
func TestAuthRawParams(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			AssertEqual(t, "RawParams", r.RawParams(), mock.Params)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.AuthRequest()
		req.Params = mock.Params
		inb := s.Request("auth.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawParams on a auth request with no parameters
func TestAuthRawParamsWithNilParams(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			AssertEqual(t, "RawParams", r.RawParams(), nil)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawToken on a auth request with token
func TestAuthRawToken(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			AssertEqual(t, "RawToken", r.RawToken(), mock.Token)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.AuthRequest()
		req.Token = mock.Token
		inb := s.Request("auth.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawToken on a auth request with no token
func TestAuthRawTokenWithNoToken(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			AssertEqual(t, "RawToken", r.RawToken(), nil)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseParams on a auth request with parameters
func TestAuthParseParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.ParseParams(&p)
			AssertEqual(t, "p.Foo", p.Foo, "bar")
			AssertEqual(t, "p.Baz", p.Baz, 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.AuthRequest()
		req.Params = mock.Params
		inb := s.Request("auth.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseParams on a auth request with no parameters
func TestAuthParseParamsWithNilParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.ParseParams(&p)
			AssertEqual(t, "p.Foo", p.Foo, "")
			AssertEqual(t, "p.Baz", p.Baz, 0)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseToken on a auth request with token
func TestAuthParseToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.ParseToken(&o)
			AssertEqual(t, "o.User", o.User, "foo")
			AssertEqual(t, "o.ID", o.ID, 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.AuthRequest()
		req.Token = mock.Token
		inb := s.Request("auth.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseToken on a auth request with no token
func TestAuthParseTokenWithNilToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.ParseToken(&o)
			AssertEqual(t, "o.User", o.User, "")
			AssertEqual(t, "o.ID", o.ID, 0)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test that registering auth methods with duplicate names causes panic
func TestRegisteringDuplicateAuthMethodPanics(t *testing.T) {
	runTest(t, func(s *Session) {
		AssertPanic(t, func() {
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
	}, nil, withoutReset)
}

// Test that Timeout sends the pre-response with timeout
func TestAuthRequestTimeout(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).AssertSubject(t, inb).AssertRawPayload(t, []byte(`timeout:"42000"`))
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero
func TestAuthRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *Session) {
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
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, "system.internalError")
	})
}

// Test that TokenEvent sends a connection token event.
func TestAuthRequestTokenEvent(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.TokenEvent(mock.Token)
			r.OK(nil)
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).
			AssertSubject(t, "conn."+mock.CID+".token").
			AssertPayload(t, json.RawMessage(`{"token":{"user":"foo","id":42}}`))
		s.GetMsg(t).AssertSubject(t, inb).AssertResult(t, nil)
	})
}

// Test that TokenEvent with nil sends a connection token event with a nil token.
func TestAuthRequestNilTokenEvent(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.TokenEvent(nil)
			r.OK(nil)
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.AuthRequest())
		s.GetMsg(t).AssertSubject(t, "conn."+mock.CID+".token").AssertPayload(t, json.RawMessage(`{"token":null}`))
		s.GetMsg(t).AssertSubject(t, inb).AssertResult(t, nil)
	})
}

// Test auth request with an unset method returns error system.methodNotFound
func TestAuthRequest_UnknownMethod_ErrorMethodNotFound(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(nil)
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.unset", mock.AuthRequest())
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrMethodNotFound)
	})
}

// Test that multiple responses to auth request causes panic
func TestAuth_WithMultipleResponses_CausesPanic(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(nil)
			AssertPanic(t, func() {
				r.MethodNotFound()
			})
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", mock.Request())
		s.GetMsg(t).AssertSubject(t, inb).AssertResult(t, nil)
	})
}
