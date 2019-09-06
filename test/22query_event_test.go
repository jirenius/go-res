package test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
)

// Test QueryEvent sends a query event with a inbox subject
func TestQueryEvent(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {})
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		ev := s.GetMsg(t)
		_, ok := ev.PathPayload(t, "subject").(string)
		if !ok {
			t.Errorf("expected query event payload contain subject string, but got %#v", ev.Payload())
		}
		s.GetMsg(t).AssertSubject(t, inb)
	}, withGnatsd)
}

// Test QueryEvent expiration causes callback to be called with nil
func TestQueryEventExpiration(t *testing.T) {
	model := resource["test.model"]
	var done func()
	ch := make(chan struct{})

	runTestAsync(t, func(s *Session) {
		s.SetQueryEventDuration(time.Millisecond)
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {
					if r != nil {
						t.Errorf("expected query event callback to be called with nil")
					}
					close(ch)
				})
				r.OK(nil)
			}),
		)
	}, func(s *Session, d func()) {
		done = d
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t)
		s.GetMsg(t).AssertSubject(t, inb)
		<-ch
		done()
	}, withGnatsd)
}

// Test SetQueryEventDuration panics when called after starting service
func TestSetQueryEventDurationPanicsAfterStart(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *Session) {
		AssertPanic(t, func() {
			s.SetQueryEventDuration(time.Second * 5)
		})
	})
}

// Test QueryEvent callback called directly on failed query subscription
func TestQueryEventFailedSubscribe(t *testing.T) {
	model := resource["test.model"]
	var done func()

	runTestAsync(t, func(s *Session) {
		s.SetQueryEventDuration(time.Millisecond)
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
			res.Call("method", func(r res.CallRequest) {
				s.Close()
				callbackCalled := false
				r.QueryEvent(func(r res.QueryRequest) {
					if r != nil {
						t.Errorf("expected query event callback to be called with nil, but got: %s", r.Query())
					}
					callbackCalled = true
				})
				AssertEqual(t, "callbackCalled", callbackCalled, true)
				done()
			}),
		)
	}, func(s *Session, d func()) {
		done = d
		s.Request("call.test.model.method", nil)
	}, withGnatsd)
}

// Test QueryRequests being received on query event
func TestQueryRequest(t *testing.T) {
	model := resource["test.model"]
	query := "foo=bar"

	events := json.RawMessage(`{"events":[]}`)

	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {
					AssertEqual(t, "query", r.Query(), query)
				})
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		subj := s.GetMsg(t).PathPayload(t, "subject").(string)
		s.GetMsg(t).AssertSubject(t, inb)
		inb = s.Request(subj, json.RawMessage(`{"query":"`+query+`"}`))
		s.GetMsg(t).AssertSubject(t, inb).AssertResult(t, events)
	}, withGnatsd)
}

// Test QueryRequest responses.
func TestQueryRequestResponse(t *testing.T) {
	tbl := []struct {
		Query    string
		Callback func(r res.QueryRequest)
		Expected interface{}
	}{
		{
			"foo=none",
			func(r res.QueryRequest) {
				AssertEqual(t, "query", r.Query(), "foo=none")
			},
			json.RawMessage(`{"events":[]}`),
		},
		{
			"foo=change",
			func(r res.QueryRequest) {
				if r != nil {
					r.ChangeEvent(map[string]interface{}{"foo": "bar"})
				}
			},
			json.RawMessage(`{"events":[{"event":"change","data":{"values":{"foo":"bar"}}}]}`),
		},
		{
			"foo=change_with_empty",
			func(r res.QueryRequest) {
				if r != nil {
					r.ChangeEvent(nil)
				}
			},
			json.RawMessage(`{"events":[]}`),
		},
		{
			"foo=add",
			func(r res.QueryRequest) {
				if r != nil {
					r.AddEvent("bar", 2)
				}
			},
			json.RawMessage(`{"events":[{"event":"add","data":{"value":"bar","idx":2}}]}`),
		},
		{
			"foo=invalid_add",
			func(r res.QueryRequest) {
				if r != nil {
					r.AddEvent("bar", -1)
				}
			},
			res.CodeInternalError,
		},
		{
			"foo=remove",
			func(r res.QueryRequest) {
				if r != nil {
					r.RemoveEvent(3)
				}
			},
			json.RawMessage(`{"events":[{"event":"remove","data":{"idx":3}}]}`),
		},
		{
			"foo=invalid_remove",
			func(r res.QueryRequest) {
				if r != nil {
					r.RemoveEvent(-1)
				}
			},
			res.CodeInternalError,
		},
		{
			"foo=removeadd",
			func(r res.QueryRequest) {
				if r != nil {
					r.RemoveEvent(3)
					r.AddEvent("bar", 2)
				}
			},
			json.RawMessage(`{"events":[{"event":"remove","data":{"idx":3}},{"event":"add","data":{"value":"bar","idx":2}}]}`),
		},
		{
			"foo=addremove",
			func(r res.QueryRequest) {
				if r != nil {
					r.AddEvent("bar", 2)
					r.RemoveEvent(3)
				}
			},
			json.RawMessage(`{"events":[{"event":"add","data":{"value":"bar","idx":2}},{"event":"remove","data":{"idx":3}}]}`),
		},
		{
			"foo=notFound",
			func(r res.QueryRequest) {
				if r != nil {
					r.NotFound()
				}
			},
			res.ErrNotFound,
		},
		{
			"foo=error",
			func(r res.QueryRequest) {
				if r != nil {
					r.Error(res.ErrTimeout)
				}
			},
			res.ErrTimeout,
		},
		{
			"foo=panic_res.Error",
			func(r res.QueryRequest) {
				if r != nil {
					panic(res.ErrNotFound)
				}
			},
			res.ErrNotFound,
		},
		{
			"foo=panic_error",
			func(r res.QueryRequest) {
				if r != nil {
					panic(errors.New("panic"))
				}
			},
			res.CodeInternalError,
		},
		{
			"foo=panic_string",
			func(r res.QueryRequest) {
				if r != nil {
					panic("panic")
				}
			},
			res.CodeInternalError,
		},
		{
			"foo=panic_other",
			func(r res.QueryRequest) {
				if r != nil {
					panic(42)
				}
			},
			res.CodeInternalError,
		},
		{
			"foo=error_after_event",
			func(r res.QueryRequest) {
				if r != nil {
					r.ChangeEvent(map[string]interface{}{"foo": "bar"})
					r.Error(res.ErrNotFound)
				}
			},
			res.ErrNotFound,
		},
		{
			"foo=panic_after_event",
			func(r res.QueryRequest) {
				if r != nil {
					r.ChangeEvent(map[string]interface{}{"foo": "bar"})
					panic(res.ErrNotFound)
				}
			},
			res.ErrNotFound,
		},
		{
			"foo=multiple_error",
			func(r res.QueryRequest) {
				if r != nil {
					r.NotFound()
					r.Error(res.ErrInternalError)
				}
			},
			res.ErrNotFound,
		},
		{
			"foo=error_and_panic",
			func(r res.QueryRequest) {
				if r != nil {
					r.NotFound()
					panic(res.ErrInternalError)
				}
			},
			res.ErrNotFound,
		},
	}
	lookup := make(map[string]func(r res.QueryRequest), len(tbl))

	runTest(t, func(s *Session) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				var q string
				r.ParseParams(&q)
				l, ok := lookup[q]
				if !ok {
					t.Errorf("expected to find callback for test query %#v, but found none", q)
				} else {
					r.QueryEvent(l)
				}
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		// Run tests within a single session
		// because starting gnatsd takes a couple of seconds
		for _, l := range tbl {
			lookup[l.Query] = l.Callback
			req := newDefaultRequest()
			req.Params = json.RawMessage(`"` + l.Query + `"`)
			inb := s.Request("call.test.model.method", req)
			subj := s.GetMsg(t).PathPayload(t, "subject").(string)

			s.GetMsg(t).AssertSubject(t, inb)
			inb = s.Request(subj, json.RawMessage(`{"query":"`+l.Query+`"}`))
			resp := s.GetMsg(t).AssertSubject(t, inb)
			switch v := l.Expected.(type) {
			case *res.Error:
				resp.AssertError(t, v)
			case string:
				resp.AssertErrorCode(t, v)
			default:
				resp.AssertResult(t, v)
			}
		}
	}, withGnatsd)
}

// Test QueryRequest' Timeout sends a correct pre-response
func TestQueryRequestTimeout(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {
					r.Timeout(time.Second * 42)
				})
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		subj := s.GetMsg(t).PathPayload(t, "subject").(string)
		s.GetMsg(t).AssertSubject(t, inb)
		inb = s.Request(subj, json.RawMessage(`{"query":"foo=bar"}`))
		s.GetMsg(t).AssertSubject(t, inb).AssertRawPayload(t, []byte(`timeout:"42000"`))
		s.GetMsg(t).AssertSubject(t, inb).AssertResult(t, json.RawMessage(`{"events":[]}`))
	}, withGnatsd)
}

// Test QueryRequest' with invalid Timeout duration returns error
func TestQueryRequestInvalidTimeout(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {
					r.Timeout(time.Second * -42)
				})
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		subj := s.GetMsg(t).PathPayload(t, "subject").(string)
		s.GetMsg(t).AssertSubject(t, inb)
		inb = s.Request(subj, json.RawMessage(`{"query":"foo=bar"}`))
		s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, res.CodeInternalError)
	}, withGnatsd)
}

// Test Invalid QueryRequests gets an internal error response
func TestInvalidQueryRequest(t *testing.T) {
	tbl := []struct {
		Payload json.RawMessage
	}{
		{json.RawMessage(`{}`)},
		{json.RawMessage(`{"query":""}`)},
		{json.RawMessage(`]`)},
	}

	runTest(t, func(s *Session) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {})
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		for i, l := range tbl {
			inb := s.Request("call.test.model.method", newDefaultRequest())
			subj := s.GetMsg(t).PathPayload(t, "subject").(string)

			s.GetMsg(t).AssertSubject(t, inb)
			inb = s.RequestRaw(subj, l.Payload)
			s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, res.CodeInternalError)

			if t.Failed() {
				t.Logf("failed on test idx %d", i)
				break
			}
		}
	}, withGnatsd)
}
