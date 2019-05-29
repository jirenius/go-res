package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jirenius/go-res"
)

// Test new response with result
func TestNew(t *testing.T) {
	rid := "model.12"

	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.New(res.Ref(rid))
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.collection.new", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"rid":"`+rid+`"}}`))
	})
}

// Test NewRequest getter methods
func TestNewRequestGetters(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			AssertEqual(t, "CID", r.CID(), defaultCID)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test new response with empty reference RID
func TestNewWithNil(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.New("")
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.collection.new", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertErrorCode(t, "system.internalError")
	})
}

// Test calling NotFound on a new request results in system.notFound
func TestNewNotFound(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.collection.new", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling MethodNotFound on a new request results in system.newNotFound
func TestNewMethodNotFound(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.MethodNotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.collection.new", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test calling InvalidParams with no message on a new request results in system.invalidParams
func TestNewDefaultInvalidParams(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.InvalidParams("")
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.collection.new", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrInvalidParams)
	})
}

// Test calling InvalidParams on a new request results in system.invalidParams
func TestNewInvalidParams(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.InvalidParams("foo")
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.collection.new", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, &res.Error{
				Code:    res.CodeInvalidParams,
				Message: "foo",
			})
	})
}

// Test calling Error on a new request results in given error
func TestNewError(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.Error(res.ErrTimeout)
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.collection.new", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrTimeout)
	})
}

// Test calling RawParams on a new request with parameters
func TestNewRawParams(t *testing.T) {
	params := json.RawMessage(`{"foo":"bar","baz":42}`)

	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			AssertEqual(t, "RawParams", r.RawParams(), params)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		req.Params = params
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawParams on a new request with no parameters
func TestNewRawParamsWithNilParams(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			AssertEqual(t, "RawParams", r.RawParams(), nil)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawToken on a new request with token
func TestNewRawToken(t *testing.T) {
	token := json.RawMessage(`{"user":"foo","id":42}`)

	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			AssertEqual(t, "RawToken", r.RawToken(), token)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		req.Token = token
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling RawToken on a new request with no token
func TestNewRawTokenWithNoToken(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			AssertEqual(t, "RawToken", r.RawToken(), nil)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseParams on a new request with parameters
func TestNewParseParams(t *testing.T) {
	params := json.RawMessage(`{"foo":"bar","baz":42}`)
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.ParseParams(&p)
			AssertEqual(t, "p.Foo", p.Foo, "bar")
			AssertEqual(t, "p.Baz", p.Baz, 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		req.Params = params
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseParams on a new request with no parameters
func TestNewParseParamsWithNilParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.ParseParams(&p)
			AssertEqual(t, "p.Foo", p.Foo, "")
			AssertEqual(t, "p.Baz", p.Baz, 0)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseToken on a new request with token
func TestNewParseToken(t *testing.T) {
	token := json.RawMessage(`{"user":"foo","id":42}`)
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.ParseToken(&o)
			AssertEqual(t, "o.User", o.User, "foo")
			AssertEqual(t, "o.ID", o.ID, 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		req.Token = token
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test calling ParseToken on a new request with no token
func TestNewParseTokenWithNilToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *Session) {
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.ParseToken(&o)
			AssertEqual(t, "o.User", o.User, "")
			AssertEqual(t, "o.ID", o.ID, 0)
			r.NotFound()
		}))
	}, func(s *Session) {
		req := newDefaultRequest()
		inb := s.Request("call.test.collection.new", req)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test registering a new handler using the Call method causes panic
func TestRegisteringNewCallPanics(t *testing.T) {
	runTest(t, func(s *Session) {
		AssertPanic(t, func() {
			s.Handle("model", res.Call("new", func(r res.CallRequest) {
				r.OK(nil)
			}))
		})
	}, nil, withoutReset)
}

// Test registering multiple new handlers causes panic
func TestRegisteringMultipleNewHandlersPanics(t *testing.T) {
	runTest(t, func(s *Session) {
		AssertPanic(t, func() {
			s.Handle("model",
				res.New(func(r res.NewRequest) {
					r.NotFound()
				}),
				res.New(func(r res.NewRequest) {
					r.NotFound()
				}),
			)
		})
	}, nil, withoutReset)
}

// Test that Timeout sends the pre-response with timeout
func TestNewRequestTimeout(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.New(func(r res.NewRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.new", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertRawPayload(t, []byte(`timeout:"42000"`))
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero
func TestNewRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.New(func(r res.NewRequest) {
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
		inb := s.Request("call.test.model.new", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, "system.internalError")
	})
}
