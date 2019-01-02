package test

import (
	"encoding/json"
	"testing"

	"github.com/jirenius/go-res"
)

// Test call response with result
func TestCall(t *testing.T) {
	result := `{"foo":"bar","zoo":42}`

	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.OK(json.RawMessage(result))
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":`+result+`}`))
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
func TestCallDefaultInvalidParams(t *testing.T) {
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
func TestCallInvalidParams(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.InvalidParams("foo")
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, &res.Error{
				Code:    res.CodeInvalidParams,
				Message: "foo",
			})
	})
}

// Test calling Error on a call request results in given error
func TestCallError(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.Error(res.ErrDisposing)
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrDisposing)
	})
}

// Test calling RawParams on a call request with parameters
func TestCallRawParams(t *testing.T) {
	params := json.RawMessage(`{"foo":"bar","baz":42}`)

	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			AssertEqual(t, r.RawParams(), params)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
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
			AssertEqual(t, r.RawParams(), nil)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
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
			AssertEqual(t, r.RawToken(), token)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
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
			AssertEqual(t, r.RawToken(), nil)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
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
			AssertEqual(t, p.Foo, "bar")
			AssertEqual(t, p.Baz, 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
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
			AssertEqual(t, p.Foo, "")
			AssertEqual(t, p.Baz, 0)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
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
			AssertEqual(t, o.User, "foo")
			AssertEqual(t, o.ID, 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
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
			AssertEqual(t, o.User, "")
			AssertEqual(t, o.ID, 0)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		inb := s.Request("call.test.model.method", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}
