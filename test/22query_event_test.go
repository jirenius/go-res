package test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test QueryEvent sends a query event with a inbox subject
func TestQueryEvent(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(mock.Model)
			}),
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {})
				r.OK(nil)
			}),
		)
	}, func(s *restest.Session) {
		req := s.Call("test.model", "method", nil)
		s.GetMsg().
			AssertQueryEvent("test.model", nil)
		req.Response()
	}, restest.WithGnatsd)
}

// Test QueryEvent expiration causes callback to be called with nil
func TestQueryEventExpiration(t *testing.T) {
	var done func()
	ch := make(chan struct{})

	runTestAsync(t, func(s *res.Service) {
		s.SetQueryEventDuration(time.Millisecond)
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(mock.Model)
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
	}, func(s *restest.Session, d func()) {
		done = d
		req := s.Call("test.model", "method", nil)
		s.GetMsg().AssertEventName("test.model", "query")
		req.Response()
		<-ch
		done()
	}, restest.WithGnatsd)
}

// Test SetQueryEventDuration panics when called after starting service
func TestSetQueryEventDurationPanicsAfterStart(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *restest.Session) {
		restest.AssertPanic(t, func() {
			s.Service().SetQueryEventDuration(time.Second * 5)
		})
	})
}

// Test QueryEvent callback called directly on failed query subscription
func TestQueryEventFailedSubscribe(t *testing.T) {
	var session *restest.Session

	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(mock.Model)
			}),
			res.Call("method", func(r res.CallRequest) {
				session.FailNextSubscription()
				callbackCalled := false
				r.QueryEvent(func(r res.QueryRequest) {
					if r != nil {
						t.Errorf("expected query event callback to be called with nil, but got: %s", r.Query())
					}
					callbackCalled = true
				})
				restest.AssertEqualJSON(t, "callbackCalled", callbackCalled, true)
			}),
		)
	}, func(s *restest.Session) {
		session = s
		s.Request("call.test.model.method", nil).Response()
	}, restest.WithGnatsd)
}

// Test QueryRequests being received on query event
func TestQueryRequest(t *testing.T) {
	events := json.RawMessage(`{"events":[]}`)

	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(mock.Model)
			}),
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {
					if r != nil {
						restest.AssertEqualJSON(t, "query", r.Query(), mock.Query)
					}
				})
				r.OK(nil)
			}),
		)
	}, func(s *restest.Session) {
		var subj string
		req := s.Call("test.model", "method", nil)
		s.GetMsg().AssertQueryEvent("test.model", &subj)
		req.Response()
		s.QueryRequest(subj, mock.Query).
			Response().
			AssertResult(events)
	}, restest.WithGnatsd)
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
				restest.AssertEqualJSON(t, "query", r.Query(), "foo=none")
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
			"foo=invalidQuery",
			func(r res.QueryRequest) {
				if r != nil {
					r.InvalidQuery("")
				}
			},
			res.ErrInvalidQuery,
		},
		{
			"foo=invalidQuery_with_message",
			func(r res.QueryRequest) {
				if r != nil {
					r.InvalidQuery(mock.ErrorMessage)
				}
			},
			&res.Error{Code: res.CodeInvalidQuery, Message: mock.ErrorMessage},
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
		{
			"foo=model_response",
			func(r res.QueryRequest) {
				if r != nil {
					r.Model(mock.Model)
				}
			},
			json.RawMessage(`{"model":{"id":42,"foo":"bar"}}`),
		},
		{
			"foo=event_with_model_response",
			func(r res.QueryRequest) {
				if r != nil {
					r.ChangeEvent(map[string]interface{}{"foo": "bar"})
					r.Model(mock.Model)
				}
			},
			json.RawMessage(`{"model":{"id":42,"foo":"bar"}}`),
		},
		{
			"foo=collection_response",
			func(r res.QueryRequest) {
				if r != nil {
					r.Collection(mock.Collection)
				}
			},
			json.RawMessage(`{"collection":[42,"foo",null]}`),
		},
		{
			"foo=event_with_collection_response",
			func(r res.QueryRequest) {
				if r != nil {
					r.ChangeEvent(map[string]interface{}{"foo": "bar"})
					r.Collection(mock.Collection)
				}
			},
			json.RawMessage(`{"collection":[42,"foo",null]}`),
		},
	}
	lookup := make(map[string]func(r res.QueryRequest), len(tbl))

	runTest(t, func(s *res.Service) {
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
	}, func(s *restest.Session) {
		// Run tests within a single session
		// because starting gnatsd takes a couple of seconds
		for _, l := range tbl {
			var subj string
			lookup[l.Query] = l.Callback
			r := mock.DefaultRequest()
			r.Params = json.RawMessage(`"` + l.Query + `"`)
			req := s.Call("test.model", "method", r)
			s.GetMsg().AssertQueryEvent("test.model", &subj)

			req.Response()
			resp := s.QueryRequest(subj, l.Query).Response()
			switch v := l.Expected.(type) {
			case *res.Error:
				resp.AssertError(v)
			case string:
				resp.AssertErrorCode(v)
			default:
				resp.AssertResult(v)
			}
		}
	}, restest.WithGnatsd)
}

// Test QueryRequest' Timeout sends a correct pre-response
func TestQueryRequestTimeout(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {
					r.Timeout(time.Second * 42)
				})
				r.OK(nil)
			}),
		)
	}, func(s *restest.Session) {
		var subj string
		req := s.Call("test.model", "method", nil)
		s.GetMsg().AssertQueryEvent("test.model", &subj)
		req.Response()
		qreq := s.QueryRequest(subj, mock.Query)
		qreq.Response().AssertRawPayload([]byte(`timeout:"42000"`))
		qreq.Response().AssertResult(json.RawMessage(`{"events":[]}`))
	}, restest.WithGnatsd)
}

// Test QueryRequest' with invalid Timeout duration returns error
func TestQueryRequestInvalidTimeout(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {
					r.Timeout(time.Second * -42)
				})
				r.OK(nil)
			}),
		)
	}, func(s *restest.Session) {
		var subj string
		req := s.Call("test.model", "method", nil)
		s.GetMsg().AssertQueryEvent("test.model", &subj)
		req.Response()
		s.QueryRequest(subj, mock.Query).
			Response().
			AssertErrorCode(res.CodeInternalError)
	}, restest.WithGnatsd)
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

	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.QueryEvent(func(r res.QueryRequest) {})
				r.OK(nil)
			}),
		)
	}, func(s *restest.Session) {
		for i, l := range tbl {
			var subj string
			req := s.Call("test.model", "method", nil)
			s.GetMsg().AssertQueryEvent("test.model", &subj)
			req.Response()
			inb := s.RequestRaw(subj, l.Payload)
			s.GetMsg().
				AssertSubject(inb).
				AssertErrorCode(res.CodeInternalError)

			if t.Failed() {
				t.Logf("failed on test idx %d", i)
				break
			}
		}
	}, restest.WithGnatsd)
}

// Test Invalid QueryRequests gets an internal error response
func TestInvalidQueryResponse(t *testing.T) {
	tbl := []struct {
		RID                 string
		QueryRequestHandler func(t *testing.T, r res.QueryRequest)
	}{
		// Collection response on query model
		{"test.model", func(t *testing.T, r res.QueryRequest) {
			restest.AssertPanicNoRecover(t, func() {
				r.Collection(mock.Collection)
			})
		}},
		// Model response on query collection
		{"test.collection", func(t *testing.T, r res.QueryRequest) {
			restest.AssertPanicNoRecover(t, func() {
				r.Model(mock.Model)
			})
		}},
		// Change event on query collection
		{"test.collection", func(t *testing.T, r res.QueryRequest) {
			restest.AssertPanicNoRecover(t, func() {
				r.ChangeEvent(map[string]interface{}{"foo": "bar"})
			})
		}},
		// Add event on query model
		{"test.model", func(t *testing.T, r res.QueryRequest) {
			restest.AssertPanicNoRecover(t, func() {
				r.AddEvent("foo", 0)
			})
		}},
		// Remove event on query model
		{"test.model", func(t *testing.T, r res.QueryRequest) {
			restest.AssertPanicNoRecover(t, func() {
				r.RemoveEvent(0)
			})
		}},
		// Unserializable change value
		{"test.model", func(t *testing.T, r res.QueryRequest) {
			r.ChangeEvent(map[string]interface{}{"foo": mock.UnserializableValue})
		}},
		// Unserializable add value
		{"test.collection", func(t *testing.T, r res.QueryRequest) {
			r.AddEvent(mock.UnserializableValue, 0)
		}},
		// Unserializable model
		{"test.model", func(t *testing.T, r res.QueryRequest) {
			r.Model(mock.UnserializableValue)
		}},
		// Unserializable collection
		{"test.collection", func(t *testing.T, r res.QueryRequest) {
			r.Collection(mock.UnserializableValue)
		}},
		// Unserializable error
		{"test.model", func(t *testing.T, r res.QueryRequest) {
			r.Error(mock.UnserializableError)
		}},
	}

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.QueryModel(mock.Model, mock.NormalizedQuery)
		}))
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.QueryCollection(mock.Collection, mock.NormalizedQuery)
		}))
	}, func(s *restest.Session) {
		ch := make(chan bool, 1)
		for i, l := range tbl {
			func() {
				defer func() { ch <- true }()

				s.Service().With(l.RID, func(r res.Resource) {
					r.QueryEvent(func(r res.QueryRequest) {
						if r != nil {
							l.QueryRequestHandler(t, r)
						}
					})
				})

				// Assert a query event for the resource
				var subj string
				s.GetMsg().AssertQueryEvent(l.RID, &subj)

				// Send query request to the subject.
				// Make sure we have an internal error
				s.QueryRequest(subj, mock.Query).
					Response().
					AssertErrorCode(res.CodeInternalError)
			}()
			select {
			case <-ch:
			case <-time.After(timeoutDuration):
				if t == nil {
					t.Fatal("expected query request to get a query response, but it timed out")
				}
			}
			if t.Failed() {
				t.Logf("failed on test idx %d", i)
				break
			}
		}
		close(ch)
	}, restest.WithGnatsd)
}
