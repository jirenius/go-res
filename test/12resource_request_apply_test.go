package test

import (
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test ApplyChange is called on change event.
func TestApplyChangeEvent(t *testing.T) {
	for _, l := range changeEventTestTbl {
		called := false

		runTest(t, func(s *res.Service) {
			s.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.ChangeEvent(l.Payload)
					restest.AssertEqualJSON(t, "called", called, true)
					r.OK(nil)
				}),
				res.ApplyChange(func(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
					called = true
					restest.AssertEqualJSON(t, "changes", changes, l.Payload)
					restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.model")
					return map[string]interface{}{"foo": 12}, nil
				}),
			)
		}, func(s *restest.Session) {
			req := s.Call("test.model", "method", nil)
			s.GetMsg().AssertEventName("test.model", "change")
			req.Response()
		})
	}
}

// Test ApplyChange is not called when no properties has been changed.
func TestApplyEmptyChangeEvent(t *testing.T) {
	called := false

	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.NotFound()
			}),
			res.Call("method", func(r res.CallRequest) {
				r.ChangeEvent(nil)
				restest.AssertEqualJSON(t, "called", called, false)
				r.OK(nil)
			}),
			res.ApplyChange(func(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
				called = true
				return nil, nil
			}),
		)
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).Response()

	})
}

// Test ApplyChange is called on change event using With
func TestApplyChangeEventUsingWith(t *testing.T) {
	for _, l := range changeEventTestTbl {
		called := false

		runTest(t, func(s *res.Service) {
			s.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.NotFound()
				}),
				res.ApplyChange(func(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
					called = true
					restest.AssertEqualJSON(t, "changes", changes, l.Payload)
					restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.model")
					return map[string]interface{}{"foo": 12}, nil
				}),
			)
		}, func(s *restest.Session) {
			restest.AssertNoError(t, s.Service().With("test.model", func(r res.Resource) {
				r.ChangeEvent(l.Payload)
				restest.AssertEqualJSON(t, "called", called, true)
			}))
			s.GetMsg().AssertEventName("test.model", "change")
		})
	}
}

// Test ApplyChange error causes panic
func TestApplyChangeErrorCausesPanic(t *testing.T) {
	for _, l := range changeEventTestTbl {
		runTest(t, func(s *res.Service) {
			s.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					restest.AssertPanicNoRecover(t, func() {
						r.ChangeEvent(l.Payload)
					})
				}),
				res.ApplyChange(func(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
					return nil, res.ErrTimeout
				}),
			)
		}, func(s *restest.Session) {
			s.Call("test.model", "method", nil).
				Response().
				AssertError(res.ErrTimeout)
		})
	}
}

// Test ApplyAdd is called on add event.
func TestApplyAddEvent(t *testing.T) {
	for _, l := range addEventTestTbl {
		called := false
		runTest(t, func(s *res.Service) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.AddEvent(l.Value, l.Idx)
					restest.AssertEqualJSON(t, "called", called, true)
					r.OK(nil)
				}),
				res.ApplyAdd(func(r res.Resource, value interface{}, idx int) error {
					called = true
					restest.AssertEqualJSON(t, "value", value, l.Value)
					restest.AssertEqualJSON(t, "idx", idx, l.Idx)
					restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.collection")
					return nil
				}),
			)
		}, func(s *restest.Session) {
			req := s.Call("test.collection", "method", nil)
			s.GetMsg().AssertEventName("test.collection", "add")
			req.Response()
		})
	}
}

// Test ApplyAddEvent is called on add event, using With
func TestApplyAddEventUsingWith(t *testing.T) {
	for _, l := range addEventTestTbl {
		called := false
		runTest(t, func(s *res.Service) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.ApplyAdd(func(r res.Resource, value interface{}, idx int) error {
					called = true
					restest.AssertEqualJSON(t, "value", value, l.Value)
					restest.AssertEqualJSON(t, "idx", idx, l.Idx)
					restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.collection")
					return nil
				}),
			)
		}, func(s *restest.Session) {
			restest.AssertNoError(t, s.Service().With("test.collection", func(r res.Resource) {
				r.AddEvent(l.Value, l.Idx)
				restest.AssertEqualJSON(t, "called", called, true)
			}))
			s.GetMsg().AssertEventName("test.collection", "add")
		})
	}
}

// Test ApplyAdd error causes panic
func TestApplyAddErrorCausesPanic(t *testing.T) {
	for _, l := range addEventTestTbl {
		runTest(t, func(s *res.Service) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					restest.AssertPanicNoRecover(t, func() {
						r.AddEvent(l.Value, l.Idx)
					})
				}),
				res.ApplyAdd(func(r res.Resource, value interface{}, idx int) error {
					return res.ErrTimeout
				}),
			)
		}, func(s *restest.Session) {
			s.Call("test.collection", "method", nil).
				Response().
				AssertError(res.ErrTimeout)
		})
	}
}

// Test ApplyRemove is called on remove event.
func TestApplyRemoveEvent(t *testing.T) {
	for _, l := range removeEventTestTbl {
		called := false
		runTest(t, func(s *res.Service) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.RemoveEvent(l.Idx)
					restest.AssertEqualJSON(t, "called", called, true)
					r.OK(nil)
				}),
				res.ApplyRemove(func(r res.Resource, idx int) (interface{}, error) {
					called = true
					restest.AssertEqualJSON(t, "idx", idx, l.Idx)
					restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.collection")
					return 42, nil
				}),
			)
		}, func(s *restest.Session) {
			req := s.Call("test.collection", "method", nil)
			s.GetMsg().AssertEventName("test.collection", "remove")
			req.Response()
		})
	}
}

// Test ApplyRemoveEvent sends an remove event with idx, using With
func TestApplyRemoveEventUsingWith(t *testing.T) {
	for _, l := range removeEventTestTbl {
		called := false
		runTest(t, func(s *res.Service) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.ApplyRemove(func(r res.Resource, idx int) (interface{}, error) {
					called = true
					restest.AssertEqualJSON(t, "idx", idx, l.Idx)
					restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.collection")
					return 42, nil
				}),
			)
		}, func(s *restest.Session) {
			restest.AssertNoError(t, s.Service().With("test.collection", func(r res.Resource) {
				r.RemoveEvent(l.Idx)
				restest.AssertEqualJSON(t, "called", called, true)
			}))
			s.GetMsg().AssertEventName("test.collection", "remove")
		})
	}
}

// Test ApplyRemove error causes panic.
func TestApplyRemoveErrorCausesPanic(t *testing.T) {
	for _, l := range removeEventTestTbl {
		runTest(t, func(s *res.Service) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					restest.AssertPanicNoRecover(t, func() {
						r.RemoveEvent(l.Idx)
					})
				}),
				res.ApplyRemove(func(r res.Resource, idx int) (interface{}, error) {
					return nil, res.ErrTimeout
				}),
			)
		}, func(s *restest.Session) {
			s.Call("test.collection", "method", nil).
				Response().
				AssertError(res.ErrTimeout)
		})
	}
}

// Test ApplyCreate is called on create event.
func TestApplyCreateEvent(t *testing.T) {
	called := false
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.CreateEvent(mock.Model)
				restest.AssertEqualJSON(t, "called", called, true)
				r.OK(nil)
			}),
			res.ApplyCreate(func(r res.Resource, value interface{}) error {
				called = true
				restest.AssertEqualJSON(t, "value", value, mock.Model)
				restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.model")
				return nil
			}),
		)
	}, func(s *restest.Session) {
		req := s.Call("test.model", "method", nil)
		s.GetMsg().AssertEventName("test.model", "create")
		req.Response()
	})
}

// Test ApplyCreateEvent sends a create event, using With.
func TestApplyCreateEventUsingWith(t *testing.T) {
	called := false
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.GetResource(func(r res.GetRequest) { r.NotFound() }),
			res.ApplyCreate(func(r res.Resource, value interface{}) error {
				called = true
				restest.AssertEqualJSON(t, "value", value, mock.Model)
				restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.model")
				return nil
			}),
		)
	}, func(s *restest.Session) {
		restest.AssertNoError(t, s.Service().With("test.model", func(r res.Resource) {
			r.CreateEvent(mock.Model)
			restest.AssertEqualJSON(t, "called", called, true)
		}))
		s.GetMsg().AssertEventName("test.model", "create")
	})
}

// Test ApplyCreate error causes panic.
func TestApplyCreateErrorCausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				restest.AssertPanicNoRecover(t, func() {
					r.CreateEvent(mock.Model)
				})
			}),
			res.ApplyCreate(func(r res.Resource, value interface{}) error {
				return res.ErrTimeout
			}),
		)
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertError(res.ErrTimeout)
	})
}

// Test ApplyDeleteEvent sends a delete event.
func TestApplyDeleteEvent(t *testing.T) {
	called := false
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.DeleteEvent()
				restest.AssertEqualJSON(t, "called", called, true)
				r.OK(nil)
			}),
			res.ApplyDelete(func(r res.Resource) (interface{}, error) {
				called = true
				restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.model")
				return mock.Model, nil
			}),
		)
	}, func(s *restest.Session) {
		req := s.Call("test.model", "method", nil)
		s.GetMsg().AssertEventName("test.model", "delete")
		req.Response()
	})
}

// Test ApplyDeleteEvent sends a delete event, using With.
func TestApplyDeleteEventUsingWith(t *testing.T) {
	called := false
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.GetResource(func(r res.GetRequest) { r.NotFound() }),
			res.ApplyDelete(func(r res.Resource) (interface{}, error) {
				called = true
				restest.AssertEqualJSON(t, "ResourceName", r.ResourceName(), "test.model")
				return mock.Model, nil
			}),
		)
	}, func(s *restest.Session) {
		restest.AssertNoError(t, s.Service().With("test.model", func(r res.Resource) {
			r.DeleteEvent()
			restest.AssertEqualJSON(t, "called", called, true)
		}))
		s.GetMsg().AssertEventName("test.model", "delete")
	})
}

// Test ApplyDelete error causes panic.
func TestApplyDeleteErrorCausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				restest.AssertPanicNoRecover(t, func() {
					r.DeleteEvent()
				})
			}),
			res.ApplyDelete(func(r res.Resource) (interface{}, error) {
				return nil, res.ErrTimeout
			}),
		)
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertError(res.ErrTimeout)
	})
}
