package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jirenius/go-res"
)

// Test call response with result
func TestCall(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.OK(mock.Result)
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).Equals(t, inb, mock.ResultResponse)
	})
}

// Test CallRequest getter methods
func TestCallRequestGetters(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("foo", func(r res.CallRequest) {
			AssertEqual(t, "Method", r.Method(), "foo")
			AssertEqual(t, "CID", r.CID(), mock.CID)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		inb := s.Request("call.test.model.foo", req)
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test call response with nil result
func TestCallWithNil(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.OK(nil)
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":null}`))
	})
}

// Test calling NotFound on a call request results in system.notFound
func TestCallNotFound(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling MethodNotFound on a call request results in system.methodNotFound
func TestCallMethodNotFound(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.MethodNotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test calling InvalidParams with no message on a call request results in system.invalidParams
func TestCallInvalidParams_EmptyMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidParams("")
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrInvalidParams)
	})
}

// Test calling InvalidParams on a call request results in system.invalidParams
func TestCallInvalidParams_CustomMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidParams(mock.ErrorMessage)
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, &res.Error{
				Code:    res.CodeInvalidParams,
				Message: mock.ErrorMessage,
			})
	})
}

// Test calling InvalidQuery with no message on a call request results in system.invalidQuery
func TestCallInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", mock.Request())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on a call request results in system.invalidQuery
func TestCallInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, &res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test calling Error on a call request results in given error
func TestCallError(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.Error(res.ErrTimeout)
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrTimeout)
	})
}

// Test calling RawParams on a call request with parameters
func TestCallRawParams(t *testing.T) {
	params := json.RawMessage(`{"foo":"bar","baz":42}`)

	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			AssertEqual(t, "RawParams", r.RawParams(), params)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		req.Params = params
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawParams on a call request with no parameters
func TestCallRawParamsWithNilParams(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			AssertEqual(t, "RawParams", r.RawParams(), nil)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawToken on a call request with token
func TestCallRawToken(t *testing.T) {
	token := json.RawMessage(`{"user":"foo","id":42}`)

	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			AssertEqual(t, "RawToken", r.RawToken(), token)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		req.Token = token
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawToken on a call request with no token
func TestCallRawTokenWithNoToken(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			AssertEqual(t, "RawToken", r.RawToken(), nil)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseParams on a call request with parameters
func TestCallParseParams(t *testing.T) {
	params := json.RawMessage(`{"foo":"bar","baz":42}`)
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ParseParams(&p)
			AssertEqual(t, "p.Foo", p.Foo, "bar")
			AssertEqual(t, "p.Baz", p.Baz, 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		req.Params = params
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseParams on a call request with no parameters
func TestCallParseParamsWithNilParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ParseParams(&p)
			AssertEqual(t, "p.Foo", p.Foo, "")
			AssertEqual(t, "p.Baz", p.Baz, 0)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseToken on a call request with token
func TestCallParseToken(t *testing.T) {
	token := json.RawMessage(`{"user":"foo","id":42}`)
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ParseToken(&o)
			AssertEqual(t, "o.User", o.User, "foo")
			AssertEqual(t, "o.ID", o.ID, 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		req.Token = token
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseToken on a call request with no token
func TestCallParseTokenWithNilToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ParseToken(&o)
			AssertEqual(t, "o.User", o.User, "")
			AssertEqual(t, "o.ID", o.ID, 0)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := mock.DefaultRequest()
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test set call response with result
func TestSetCall(t *testing.T) {
	result := `{"foo":"bar","zoo":42}`

	runTest(t, func(s *Session) {
		s.Handle("model", res.Set(func(r res.CallRequest) {
			r.OK(json.RawMessage(result))
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.set", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":`+result+`}`))
	})
}

// Test that registering call methods with duplicate names causes panic
func TestRegisteringDuplicateCallMethodPanics(t *testing.T) {
	runTest(t, func(s *Session) {
		AssertPanic(t, func() {
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
	}, nil, withoutReset)
}

// Test that Timeout sends the pre-response with timeout
func TestCallRequestTimeout(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertRawPayload(t, []byte(`timeout:"42000"`))
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero
func TestCallRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *Session) {
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
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, "system.internalError")
	})
}
