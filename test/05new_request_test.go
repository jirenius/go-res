package test

import (
	"testing"
	"time"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test new response with result
func TestNew(t *testing.T) {
	rid := "model.12"

	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.New(res.Ref(rid))
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertResult(res.Ref(rid))
	})
}

// Test NewRequest getter methods
func TestNewRequestGetters(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			restest.AssertEqualJSON(t, "CID", r.CID(), mock.CID)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test new response with empty reference RID
func TestNewWithNil(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.New("")
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertErrorCode("system.internalError")
	})
}

// Test calling NotFound on a new request results in system.notFound
func TestNewNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling MethodNotFound on a new request results in system.newNotFound
func TestNewMethodNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.MethodNotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test calling InvalidParams with no message on a new request results in system.invalidParams
func TestNewDefaultInvalidParams(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.InvalidParams("")
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrInvalidParams)
	})
}

// Test calling InvalidParams on a new request results in system.invalidParams
func TestNewInvalidParams(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.InvalidParams("foo")
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidParams,
				Message: "foo",
			})
	})
}

// Test calling Error on a new request results in given error
func TestNewError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.Error(res.ErrTimeout)
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrTimeout)
	})
}

// Test calling InvalidQuery with no message on a new request results in system.invalidQuery
func TestNewInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("model", res.New(func(r res.NewRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "new", nil).
			Response().
			AssertError(res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on a new request results in system.invalidQuery
func TestNewInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("model", res.New(func(r res.NewRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "new", nil).
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test calling RawParams on a new request with parameters
func TestNewRawParams(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			restest.AssertEqualJSON(t, "RawParams", r.RawParams(), mock.Params)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		req.Params = mock.Params
		s.Call("test.collection", "new", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawParams on a new request with no parameters
func TestNewRawParamsWithNilParams(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			restest.AssertEqualJSON(t, "RawParams", r.RawParams(), nil)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawToken on a new request with token
func TestNewRawToken(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			restest.AssertEqualJSON(t, "RawToken", r.RawToken(), mock.Token)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		r := mock.DefaultRequest()
		r.Token = mock.Token
		s.Call("test.collection", "new", r).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling RawToken on a new request with no token
func TestNewRawTokenWithNoToken(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			restest.AssertEqualJSON(t, "RawToken", r.RawToken(), nil)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseParams on a new request with parameters
func TestNewParseParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.ParseParams(&p)
			restest.AssertEqualJSON(t, "p.Foo", p.Foo, "bar")
			restest.AssertEqualJSON(t, "p.Baz", p.Baz, 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		req.Params = mock.Params
		s.Call("test.collection", "new", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseParams on a new request with no parameters
func TestNewParseParamsWithNilParams(t *testing.T) {
	var p struct {
		Foo string `json:"foo"`
		Baz int    `json:"baz"`
	}

	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.ParseParams(&p)
			restest.AssertEqualJSON(t, "p.Foo", p.Foo, "")
			restest.AssertEqualJSON(t, "p.Baz", p.Baz, 0)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseToken on a new request with token
func TestNewParseToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.ParseToken(&o)
			restest.AssertEqualJSON(t, "o.User", o.User, "foo")
			restest.AssertEqualJSON(t, "o.ID", o.ID, 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := mock.DefaultRequest()
		req.Token = mock.Token
		s.Call("test.collection", "new", req).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test calling ParseToken on a new request with no token
func TestNewParseTokenWithNilToken(t *testing.T) {
	var o struct {
		User string `json:"user"`
		ID   int    `json:"id"`
	}

	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("collection", res.New(func(r res.NewRequest) {
			r.ParseToken(&o)
			restest.AssertEqualJSON(t, "o.User", o.User, "")
			restest.AssertEqualJSON(t, "o.ID", o.ID, 0)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Call("test.collection", "new", nil).
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test registering a new handler using the Call method does not cause panic
func TestRegisteringNewCall(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("new", func(r res.CallRequest) {
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "new", nil).
			Response().
			AssertResult(nil)
	})
}

// Test registered call new method is overridden by a new handler
func TestRegisteringNewCallOverriddenByNewHandler(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Call("new", func(r res.CallRequest) {
				r.OK(res.Ref("call.handler"))
			}),
			//lint:ignore SA1019 to allow test of deprecated feature
			res.New(func(r res.NewRequest) {
				r.New("new.handler")
			}),
		)
	}, func(s *restest.Session) {
		s.Call("test.model", "new", nil).
			Response().
			AssertResult(res.Ref("new.handler"))
	})
}

// Test registering multiple new handlers causes panic
func TestRegisteringMultipleNewHandlersPanics(t *testing.T) {
	runTest(t, func(s *res.Service) {
		restest.AssertPanic(t, func() {
			s.Handle("model",
				//lint:ignore SA1019 to allow test of deprecated feature
				res.New(func(r res.NewRequest) {
					r.NotFound()
				}),
				//lint:ignore SA1019 to allow test of deprecated feature
				res.New(func(r res.NewRequest) {
					r.NotFound()
				}),
			)
		})
	}, nil, restest.WithoutReset)
}

// Test that Timeout sends the pre-response with timeout
func TestNewRequestTimeout(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
		s.Handle("model", res.New(func(r res.NewRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := s.Call("test.model", "new", nil)
		req.Response().AssertRawPayload([]byte(`timeout:"42000"`))
		req.Response().AssertError(res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero
func TestNewRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *res.Service) {
		//lint:ignore SA1019 to allow test of deprecated feature
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
	}, func(s *restest.Session) {
		s.Call("test.model", "new", nil).
			Response().AssertErrorCode("system.internalError")
	})
}
