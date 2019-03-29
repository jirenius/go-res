package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jirenius/go-res"
)

// Test auth response with result
func TestAuth(t *testing.T) {
	result := `{"foo":"bar","zoo":42}`

	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(json.RawMessage(result))
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":`+result+`}`))
	})
}

// Test AuthRequest getter methods
func TestAuthRequestGetters(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("foo", func(r res.AuthRequest) {
			AssertEqual(t, "Method", r.Method(), "foo")
			AssertEqual(t, "CID", r.CID(), defaultCID)
			AssertEqual(t, "Header", r.Header(), defaultHeader)
			AssertEqual(t, "Host", r.Host(), defaultHost)
			AssertEqual(t, "RemoteAddr", r.RemoteAddr(), defaultRemoteAddr)
			AssertEqual(t, "URI", r.URI(), defaultURI)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newAuthRequest()
		inb := s.Request("auth.test.model.foo", req)
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test auth response with nil result
func TestAuthWithNil(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.OK(nil)
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":null}`))
	})
}

// Test calling NotFound on a auth request results in system.notFound
func TestAuthNotFound(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", nil)
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
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test calling InvalidParams with no message on a auth request results in system.invalidParams
func TestAuthDefaultInvalidParams(t *testing.T) {
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
func TestAuthInvalidParams(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.InvalidParams("foo")
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, &res.Error{
				Code:    res.CodeInvalidParams,
				Message: "foo",
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
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrTimeout)
	})
}

// Test calling RawParams on a auth request with parameters
func TestAuthRawParams(t *testing.T) {
	params := json.RawMessage(`{"foo":"bar","baz":42}`)

	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			AssertEqual(t, "RawParams", r.RawParams(), params)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newAuthRequest()
		req.Params = params
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
		req := newAuthRequest()
		inb := s.Request("auth.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawToken on a auth request with token
func TestAuthRawToken(t *testing.T) {
	token := json.RawMessage(`{"user":"foo","id":42}`)

	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			AssertEqual(t, "RawToken", r.RawToken(), token)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newAuthRequest()
		req.Token = token
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
		req := newAuthRequest()
		inb := s.Request("auth.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseParams on a auth request with parameters
func TestAuthParseParams(t *testing.T) {
	params := json.RawMessage(`{"foo":"bar","baz":42}`)
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
		req := newAuthRequest()
		req.Params = params
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
		req := newAuthRequest()
		inb := s.Request("auth.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseToken on a auth request with token
func TestAuthParseToken(t *testing.T) {
	token := json.RawMessage(`{"user":"foo","id":42}`)
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
		req := newAuthRequest()
		req.Token = token
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
		req := newAuthRequest()
		inb := s.Request("auth.test.model.method", req)
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
	}, nil)
}

// Test that Timeout sends the pre-response with timeout
func TestAuthRequestTimeout(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", nil)
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
		inb := s.Request("auth.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, "system.internalError")
	})
}

// Test that TokenEvent sends a connection token event.
func TestAuthRequestTokenEvent(t *testing.T) {
	token := `{"id":42,"user":"foo","role":"admin"}`
	runTest(t, func(s *Session) {
		s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
			r.TokenEvent(json.RawMessage(token))
			r.OK(nil)
		}))
	}, func(s *Session) {
		inb := s.Request("auth.test.model.method", newAuthRequest())
		s.GetMsg(t).AssertSubject(t, "conn."+defaultCID+".token").AssertPayload(t, json.RawMessage(`{"token":`+token+`}`))
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
		inb := s.Request("auth.test.model.method", newAuthRequest())
		s.GetMsg(t).AssertSubject(t, "conn."+defaultCID+".token").AssertPayload(t, json.RawMessage(`{"token":null}`))
		s.GetMsg(t).AssertSubject(t, inb).AssertResult(t, nil)
	})
}
