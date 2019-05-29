package test

import (
	"testing"

	res "github.com/jirenius/go-res"
)

// Test ApplyChange is called on change event.
func TestApplyChangeEvent(t *testing.T) {
	for _, l := range changeEventTestTbl {
		called := false

		runTest(t, func(s *Session) {
			s.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.ChangeEvent(l.Payload)
					AssertEqual(t, "called", called, true)
					r.OK(nil)
				}),
				res.ApplyChange(func(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
					called = true
					AssertEqual(t, "changes", changes, l.Payload)
					AssertEqual(t, "ResourceName", r.ResourceName(), "test.model")
					return map[string]interface{}{"foo": 12}, nil
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.model.method", nil)
			s.GetMsg(t).AssertSubject(t, "event.test.model.change")
			s.GetMsg(t).AssertSubject(t, inb)
		})
	}
}

// Test ApplyChange is not called when no properties has been changed.
func TestApplyEmptyChangeEvent(t *testing.T) {
	called := false

	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.NotFound()
			}),
			res.Call("method", func(r res.CallRequest) {
				r.ChangeEvent(nil)
				AssertEqual(t, "called", called, false)
				r.OK(nil)
			}),
			res.ApplyChange(func(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
				called = true
				return nil, nil
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test ApplyChange is called on change event using With
func TestApplyChangeEventUsingWith(t *testing.T) {
	for _, l := range changeEventTestTbl {
		called := false

		runTest(t, func(s *Session) {
			s.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.NotFound()
				}),
				res.ApplyChange(func(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
					called = true
					AssertEqual(t, "changes", changes, l.Payload)
					AssertEqual(t, "ResourceName", r.ResourceName(), "test.model")
					return map[string]interface{}{"foo": 12}, nil
				}),
			)
		}, func(s *Session) {
			AssertNoError(t, s.With("test.model", func(r res.Resource) {
				r.ChangeEvent(l.Payload)
				AssertEqual(t, "called", called, true)
			}))
			s.GetMsg(t).AssertSubject(t, "event.test.model.change")
		})
	}
}

// Test ApplyChange error causes panic
func TestApplyChangeErrorCausesPanic(t *testing.T) {
	for _, l := range changeEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					AssertPanicNoRecover(t, func() {
						r.ChangeEvent(l.Payload)
					})
				}),
				res.ApplyChange(func(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
					return nil, res.ErrTimeout
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.model.method", nil)
			s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrTimeout)
		})
	}
}

// Test ApplyAdd is called on add event.
func TestApplyAddEvent(t *testing.T) {
	for _, l := range addEventTestTbl {
		called := false
		runTest(t, func(s *Session) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.AddEvent(l.Value, l.Idx)
					AssertEqual(t, "called", called, true)
					r.OK(nil)
				}),
				res.ApplyAdd(func(r res.Resource, value interface{}, idx int) error {
					called = true
					AssertEqual(t, "value", value, l.Value)
					AssertEqual(t, "idx", idx, l.Idx)
					AssertEqual(t, "ResourceName", r.ResourceName(), "test.collection")
					return nil
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.collection.method", nil)
			s.GetMsg(t).AssertSubject(t, "event.test.collection.add")
			s.GetMsg(t).AssertSubject(t, inb)
		})
	}
}

// Test ApplyAddEvent is called on add event, using With
func TestApplyAddEventUsingWith(t *testing.T) {
	for _, l := range addEventTestTbl {
		called := false
		runTest(t, func(s *Session) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.ApplyAdd(func(r res.Resource, value interface{}, idx int) error {
					called = true
					AssertEqual(t, "value", value, l.Value)
					AssertEqual(t, "idx", idx, l.Idx)
					AssertEqual(t, "ResourceName", r.ResourceName(), "test.collection")
					return nil
				}),
			)
		}, func(s *Session) {
			AssertNoError(t, s.With("test.collection", func(r res.Resource) {
				r.AddEvent(l.Value, l.Idx)
				AssertEqual(t, "called", called, true)
			}))
			s.GetMsg(t).AssertSubject(t, "event.test.collection.add")
		})
	}
}

// Test ApplyAdd error causes panic
func TestApplyAddErrorCausesPanic(t *testing.T) {
	for _, l := range addEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					AssertPanicNoRecover(t, func() {
						r.AddEvent(l.Value, l.Idx)
					})
				}),
				res.ApplyAdd(func(r res.Resource, value interface{}, idx int) error {
					return res.ErrTimeout
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.collection.method", nil)
			s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrTimeout)
		})
	}
}

// Test ApplyRemove is called on remove event.
func TestApplyRemoveEvent(t *testing.T) {
	for _, l := range removeEventTestTbl {
		called := false
		runTest(t, func(s *Session) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.RemoveEvent(l.Idx)
					AssertEqual(t, "called", called, true)
					r.OK(nil)
				}),
				res.ApplyRemove(func(r res.Resource, idx int) (interface{}, error) {
					called = true
					AssertEqual(t, "idx", idx, l.Idx)
					AssertEqual(t, "ResourceName", r.ResourceName(), "test.collection")
					return 42, nil
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.collection.method", nil)
			s.GetMsg(t).AssertSubject(t, "event.test.collection.remove")
			s.GetMsg(t).AssertSubject(t, inb)
		})
	}
}

// Test ApplyRemoveEvent sends an remove event with idx, using With
func TestApplyRemoveEventUsingWith(t *testing.T) {
	for _, l := range removeEventTestTbl {
		called := false
		runTest(t, func(s *Session) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.ApplyRemove(func(r res.Resource, idx int) (interface{}, error) {
					called = true
					AssertEqual(t, "idx", idx, l.Idx)
					AssertEqual(t, "ResourceName", r.ResourceName(), "test.collection")
					return 42, nil
				}),
			)
		}, func(s *Session) {
			AssertNoError(t, s.With("test.collection", func(r res.Resource) {
				r.RemoveEvent(l.Idx)
				AssertEqual(t, "called", called, true)
			}))
			s.GetMsg(t).AssertSubject(t, "event.test.collection.remove")
		})
	}
}

// Test ApplyRemove error causes panic.
func TestApplyRemoveErrorCausesPanic(t *testing.T) {
	for _, l := range removeEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					AssertPanicNoRecover(t, func() {
						r.RemoveEvent(l.Idx)
					})
				}),
				res.ApplyRemove(func(r res.Resource, idx int) (interface{}, error) {
					return nil, res.ErrTimeout
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.collection.method", nil)
			s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrTimeout)
		})
	}
}

// Test ApplyCreate is called on create event.
func TestApplyCreateEvent(t *testing.T) {
	called := false
	model := map[string]interface{}{"foo": "bar"}
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.CreateEvent(model)
				AssertEqual(t, "called", called, true)
				r.OK(nil)
			}),
			res.ApplyCreate(func(r res.Resource, value interface{}) error {
				called = true
				AssertEqual(t, "value", value, model)
				AssertEqual(t, "ResourceName", r.ResourceName(), "test.model")
				return nil
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, "event.test.model.create")
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test ApplyCreateEvent sends a create event, using With.
func TestApplyCreateEventUsingWith(t *testing.T) {
	called := false
	model := map[string]interface{}{"foo": "bar"}
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetResource(func(r res.GetRequest) { r.NotFound() }),
			res.ApplyCreate(func(r res.Resource, value interface{}) error {
				called = true
				AssertEqual(t, "value", value, model)
				AssertEqual(t, "ResourceName", r.ResourceName(), "test.model")
				return nil
			}),
		)
	}, func(s *Session) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			r.CreateEvent(map[string]interface{}{"foo": "bar"})
			AssertEqual(t, "called", called, true)
		}))
		s.GetMsg(t).AssertSubject(t, "event.test.model.create")
	})
}

// Test ApplyCreate error causes panic.
func TestApplyCreateErrorCausesPanic(t *testing.T) {
	model := map[string]interface{}{"foo": "bar"}
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				AssertPanicNoRecover(t, func() {
					r.CreateEvent(model)
				})
			}),
			res.ApplyCreate(func(r res.Resource, value interface{}) error {
				return res.ErrTimeout
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrTimeout)
	})
}

// Test ApplyDeleteEvent sends a delete event.
func TestApplyDeleteEvent(t *testing.T) {
	called := false
	model := map[string]interface{}{"foo": "bar"}
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				r.DeleteEvent()
				AssertEqual(t, "called", called, true)
				r.OK(nil)
			}),
			res.ApplyDelete(func(r res.Resource) (interface{}, error) {
				called = true
				AssertEqual(t, "ResourceName", r.ResourceName(), "test.model")
				return model, nil
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, "event.test.model.delete")
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test ApplyDeleteEvent sends a delete event, using With.
func TestApplyDeleteEventUsingWith(t *testing.T) {
	called := false
	model := map[string]interface{}{"foo": "bar"}
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetResource(func(r res.GetRequest) { r.NotFound() }),
			res.ApplyDelete(func(r res.Resource) (interface{}, error) {
				called = true
				AssertEqual(t, "ResourceName", r.ResourceName(), "test.model")
				return model, nil
			}),
		)
	}, func(s *Session) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			r.DeleteEvent()
			AssertEqual(t, "called", called, true)
		}))
		s.GetMsg(t).AssertSubject(t, "event.test.model.delete")
	})
}

// Test ApplyDelete error causes panic.
func TestApplyDeleteErrorCausesPanic(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.Call("method", func(r res.CallRequest) {
				AssertPanicNoRecover(t, func() {
					r.DeleteEvent()
				})
			}),
			res.ApplyDelete(func(r res.Resource) (interface{}, error) {
				return nil, res.ErrTimeout
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrTimeout)
	})
}
