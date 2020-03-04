package test

import (
	"encoding/json"
	"fmt"
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

var listenerChangeEventTestData = []struct {
	Changed map[string]interface{}
}{
	{map[string]interface{}{"foo": 42}},
	{map[string]interface{}{"foo": "bar"}},
	{map[string]interface{}{"foo": nil}},
	{map[string]interface{}{"foo": 12, "bar": true}},
	{map[string]interface{}{"foo": res.DeleteAction}},
	{map[string]interface{}{"foo": res.Ref("test.model.bar")}},
}

func TestListenerChangeEvent_WithApplyChange_CallsListener(t *testing.T) {
	for i, l := range listenerChangeEventTestData {
		runTest(t, func(s *res.Service) {
			ctx := fmt.Sprintf("test %d", i)
			called := 0
			rev := map[string]interface{}{"foo": "baz"}
			s.Handle("model",
				res.Call("method", func(r res.CallRequest) {
					restest.AssertEqualJSON(t, "called", called, 0, ctx)
					r.ChangeEvent(l.Changed)
					restest.AssertEqualJSON(t, "called", called, 1, ctx)
					r.OK(nil)
				}),
				res.ApplyChange(func(re res.Resource, changed map[string]interface{}) (map[string]interface{}, error) {
					return rev, nil
				}),
			)
			s.AddListener("model", func(ev *res.Event) {
				called++
				restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.model", ctx)
				restest.AssertEqualJSON(t, "ev.Name", ev.Name, "change", ctx)
				restest.AssertEqualJSON(t, "ev.NewValues", ev.NewValues, l.Changed, ctx)
				restest.AssertEqualJSON(t, "ev.OldValues", ev.OldValues, rev, ctx)
			})
		}, func(s *restest.Session) {
			req := s.Call("test.model", "method", nil)
			s.GetMsg().
				AssertChangeEvent("test.model", l.Changed)
			req.Response().
				AssertResult(nil)
		})
	}
}

func TestListenerChangeEvent_WithoutApplyChange_CallsListener(t *testing.T) {
	for i, l := range listenerChangeEventTestData {
		runTest(t, func(s *res.Service) {
			ctx := fmt.Sprintf("test %d", i)
			called := 0
			s.Handle("model",
				res.Call("method", func(r res.CallRequest) {
					restest.AssertEqualJSON(t, "called", called, 0, ctx)
					r.ChangeEvent(l.Changed)
					restest.AssertEqualJSON(t, "called", called, 1, ctx)
					r.OK(nil)
				}),
			)
			s.AddListener("model", func(ev *res.Event) {
				called++
				restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.model", ctx)
				restest.AssertEqualJSON(t, "ev.Name", ev.Name, "change", ctx)
				restest.AssertEqualJSON(t, "ev.NewValues", ev.NewValues, l.Changed, ctx)
				restest.AssertEqualJSON(t, "ev.OldValues", ev.OldValues, nil, ctx)
			})
		}, func(s *restest.Session) {
			req := s.Call("test.model", "method", nil)
			s.GetMsg().
				AssertChangeEvent("test.model", l.Changed)
			req.Response().
				AssertResult(nil)
		})
	}
}

func TestListenerChangeEvent_EmptyRevertMap_NoCallToListener(t *testing.T) {
	for i, l := range listenerChangeEventTestData {
		runTest(t, func(s *res.Service) {
			ctx := fmt.Sprintf("test %d", i)
			called := 0
			rev := map[string]interface{}{}
			s.Handle("model",
				res.Call("method", func(r res.CallRequest) {
					r.ChangeEvent(l.Changed)
					restest.AssertEqualJSON(t, "called", called, 0, ctx)
					r.OK(nil)
				}),
				res.ApplyChange(func(re res.Resource, changed map[string]interface{}) (map[string]interface{}, error) {
					return rev, nil
				}),
			)
			s.AddListener("model", func(ev *res.Event) { called++ })
		}, func(s *restest.Session) {
			s.Call("test.model", "method", nil).
				Response().
				AssertResult(nil)
		})
	}
}

func TestListenerChangeEvent_NilRevertMap_CallToListener(t *testing.T) {
	for i, l := range listenerChangeEventTestData {
		runTest(t, func(s *res.Service) {
			ctx := fmt.Sprintf("test %d", i)
			called := 0
			s.Handle("model",
				res.Call("method", func(r res.CallRequest) {
					r.ChangeEvent(l.Changed)
					restest.AssertEqualJSON(t, "called", called, 1, ctx)
					r.OK(nil)
				}),
				res.ApplyChange(func(re res.Resource, changed map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				}),
			)
			s.AddListener("model", func(ev *res.Event) {
				called++
				restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.model", ctx)
				restest.AssertEqualJSON(t, "ev.Name", ev.Name, "change", ctx)
				restest.AssertEqualJSON(t, "ev.NewValues", ev.NewValues, l.Changed, ctx)
				restest.AssertEqualJSON(t, "ev.OldValues", ev.OldValues, nil, ctx)
			})
		}, func(s *restest.Session) {
			req := s.Call("test.model", "method", nil)
			s.GetMsg().
				AssertChangeEvent("test.model", l.Changed)
			req.Response().
				AssertResult(nil)
		})
	}
}

var listenerAddEventTestData = []struct {
	Value    interface{}
	Idx      int
	Expected json.RawMessage
}{
	{42, 0, json.RawMessage(`{"value":42,"idx":0}`)},
	{"bar", 1, json.RawMessage(`{"value":"bar","idx":1}`)},
	{nil, 2, json.RawMessage(`{"value":null,"idx":2}`)},
	{true, 3, json.RawMessage(`{"value":true,"idx":3}`)},
	{res.Ref(`test.model.bar`), 4, json.RawMessage(`{"value":{"rid":"test.model.bar"},"idx":4}`)},
}

func TestListenerAddEvent_WithApplyAdd_CallsListener(t *testing.T) {
	for i, l := range listenerAddEventTestData {
		runTest(t, func(s *res.Service) {
			ctx := fmt.Sprintf("test %d", i)
			called := 0
			s.Handle("collection",
				res.Call("method", func(r res.CallRequest) {
					restest.AssertEqualJSON(t, "called", called, 0, ctx)
					r.AddEvent(l.Value, l.Idx)
					restest.AssertEqualJSON(t, "called", called, 1, ctx)
					r.OK(nil)
				}),
				res.ApplyAdd(func(re res.Resource, value interface{}, idx int) error {
					return nil
				}),
			)
			s.AddListener("collection", func(ev *res.Event) {
				called++
				restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.collection", ctx)
				restest.AssertEqualJSON(t, "ev.Name", ev.Name, "add", ctx)
				restest.AssertEqualJSON(t, "ev.Value", ev.Value, l.Value, ctx)
				restest.AssertEqualJSON(t, "ev.Idx", ev.Idx, l.Idx, ctx)
			})
		}, func(s *restest.Session) {
			req := s.Call("test.collection", "method", nil)
			s.GetMsg().
				AssertEventName("test.collection", "add").
				AssertPayload(l.Expected)
			req.Response().
				AssertResult(nil)
		})
	}
}

func TestListenerAddEvent_WithoutApplyAdd_CallsListener(t *testing.T) {
	for i, l := range listenerAddEventTestData {
		runTest(t, func(s *res.Service) {
			ctx := fmt.Sprintf("test %d", i)
			called := 0
			s.Handle("collection",
				res.Call("method", func(r res.CallRequest) {
					restest.AssertEqualJSON(t, "called", called, 0, ctx)
					r.AddEvent(l.Value, l.Idx)
					restest.AssertEqualJSON(t, "called", called, 1, ctx)
					r.OK(nil)
				}),
			)
			s.AddListener("collection", func(ev *res.Event) {
				called++
				restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.collection", ctx)
				restest.AssertEqualJSON(t, "ev.Name", ev.Name, "add", ctx)
				restest.AssertEqualJSON(t, "ev.Value", ev.Value, l.Value, ctx)
				restest.AssertEqualJSON(t, "ev.Idx", ev.Idx, l.Idx, ctx)
			})
		}, func(s *restest.Session) {
			req := s.Call("test.collection", "method", nil)
			s.GetMsg().
				AssertEventName("test.collection", "add").
				AssertPayload(l.Expected)
			req.Response().
				AssertResult(nil)
		})
	}
}

var listenerRemoveEventTestData = []struct {
	Idx      int
	Expected json.RawMessage
}{
	{0, json.RawMessage(`{"idx":0}`)},
	{1, json.RawMessage(`{"idx":1}`)},
	{2, json.RawMessage(`{"idx":2}`)},
}

func TestListenerRemoveEvent_WithApplyRemove_CallsListener(t *testing.T) {
	for i, l := range listenerRemoveEventTestData {
		runTest(t, func(s *res.Service) {
			ctx := fmt.Sprintf("test %d", i)
			called := 0
			s.Handle("collection",
				res.Call("method", func(r res.CallRequest) {
					restest.AssertEqualJSON(t, "called", called, 0, ctx)
					r.RemoveEvent(l.Idx)
					restest.AssertEqualJSON(t, "called", called, 1, ctx)
					r.OK(nil)
				}),
				res.ApplyRemove(func(re res.Resource, idx int) (interface{}, error) {
					return mock.IntValue, nil
				}),
			)
			s.AddListener("collection", func(ev *res.Event) {
				called++
				restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.collection", ctx)
				restest.AssertEqualJSON(t, "ev.Name", ev.Name, "remove", ctx)
				restest.AssertEqualJSON(t, "ev.Value", ev.Value, mock.IntValue, ctx)
				restest.AssertEqualJSON(t, "ev.Idx", ev.Idx, l.Idx, ctx)
			})
		}, func(s *restest.Session) {
			req := s.Call("test.collection", "method", nil)
			s.GetMsg().
				AssertEventName("test.collection", "remove").
				AssertPayload(l.Expected)
			req.Response().
				AssertResult(nil)
		})
	}
}

func TestListenerRemoveEvent_WithoutApplyRemove_CallsListener(t *testing.T) {
	for i, l := range listenerRemoveEventTestData {
		runTest(t, func(s *res.Service) {
			ctx := fmt.Sprintf("test %d", i)
			called := 0
			s.Handle("collection",
				res.Call("method", func(r res.CallRequest) {
					restest.AssertEqualJSON(t, "called", called, 0, ctx)
					r.RemoveEvent(l.Idx)
					restest.AssertEqualJSON(t, "called", called, 1, ctx)
					r.OK(nil)
				}),
			)
			s.AddListener("collection", func(ev *res.Event) {
				called++
				restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.collection", ctx)
				restest.AssertEqualJSON(t, "ev.Name", ev.Name, "remove", ctx)
				restest.AssertEqualJSON(t, "ev.Value", ev.Value, nil, ctx)
				restest.AssertEqualJSON(t, "ev.Idx", ev.Idx, l.Idx, ctx)
			})
		}, func(s *restest.Session) {
			req := s.Call("test.collection", "method", nil)
			s.GetMsg().
				AssertEventName("test.collection", "remove").
				AssertPayload(l.Expected)
			req.Response().
				AssertResult(nil)
		})
	}
}

func TestListenerCreateEvent_WithApplyCreate_CallsListener(t *testing.T) {
	runTest(t, func(s *res.Service) {
		called := 0
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				restest.AssertEqualJSON(t, "called", called, 0)
				r.CreateEvent(mock.Model)
				restest.AssertEqualJSON(t, "called", called, 1)
				r.OK(nil)
			}),
			res.ApplyCreate(func(re res.Resource, data interface{}) error {
				return nil
			}),
		)
		s.AddListener("model", func(ev *res.Event) {
			called++
			restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.model")
			restest.AssertEqualJSON(t, "ev.Name", ev.Name, "create")
			restest.AssertEqualJSON(t, "ev.Data", ev.Data, mock.Model)
		})
	}, func(s *restest.Session) {
		req := s.Call("test.model", "method", nil)
		s.GetMsg().
			AssertEventName("test.model", "create")
		req.Response().
			AssertResult(nil)
	})
}

func TestListenerCreateEvent_WithoutApplyCreate_CallsListener(t *testing.T) {
	runTest(t, func(s *res.Service) {
		called := 0
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				restest.AssertEqualJSON(t, "called", called, 0)
				r.CreateEvent(mock.Model)
				restest.AssertEqualJSON(t, "called", called, 1)
				r.OK(nil)
			}),
		)
		s.AddListener("model", func(ev *res.Event) {
			called++
			restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.model")
			restest.AssertEqualJSON(t, "ev.Name", ev.Name, "create")
			restest.AssertEqualJSON(t, "ev.Data", ev.Data, mock.Model)
		})
	}, func(s *restest.Session) {
		req := s.Call("test.model", "method", nil)
		s.GetMsg().
			AssertEventName("test.model", "create")
		req.Response().
			AssertResult(nil)
	})
}

func TestListenerDeleteEvent_WithApplyDelete_CallsListener(t *testing.T) {
	runTest(t, func(s *res.Service) {
		called := 0
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				restest.AssertEqualJSON(t, "called", called, 0)
				r.DeleteEvent()
				restest.AssertEqualJSON(t, "called", called, 1)
				r.OK(nil)
			}),
			res.ApplyDelete(func(re res.Resource) (interface{}, error) {
				return mock.Model, nil
			}),
		)
		s.AddListener("model", func(ev *res.Event) {
			called++
			restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.model")
			restest.AssertEqualJSON(t, "ev.Name", ev.Name, "delete")
			restest.AssertEqualJSON(t, "ev.Data", ev.Data, mock.Model)
		})
	}, func(s *restest.Session) {
		req := s.Call("test.model", "method", nil)
		s.GetMsg().
			AssertEventName("test.model", "delete")
		req.Response().
			AssertResult(nil)
	})
}

func TestListenerDeleteEvent_WithoutApplyDelete_CallsListener(t *testing.T) {
	runTest(t, func(s *res.Service) {
		called := 0
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				restest.AssertEqualJSON(t, "called", called, 0)
				r.DeleteEvent()
				restest.AssertEqualJSON(t, "called", called, 1)
				r.OK(nil)
			}),
		)
		s.AddListener("model", func(ev *res.Event) {
			called++
			restest.AssertEqualJSON(t, "re.ResourceName", ev.Resource.ResourceName(), "test.model")
			restest.AssertEqualJSON(t, "ev.Name", ev.Name, "delete")
			restest.AssertEqualJSON(t, "ev.Data", ev.Data, nil)
		})
	}, func(s *restest.Session) {
		req := s.Call("test.model", "method", nil)
		s.GetMsg().
			AssertEventName("test.model", "delete")
		req.Response().
			AssertResult(nil)
	})
}
