package test

import (
	"errors"
	"testing"
	"time"

	res "github.com/jirenius/go-res"
)

// Test that Value gets the model as provided from the GetModel resource handler.
func TestModelValue(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(model)
				AssertEqual(t, "r.ForValue()", r.ForValue(), true)
			}),
			res.Call("method", func(r res.CallRequest) {
				v, err := r.Value()
				AssertNoError(t, err)
				if v != model {
					t.Errorf("expected Value() to return model, but it didn't")
				}
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test that RequireValue gets the model as provided from the GetModel resource handler.
func TestModelRequireValue(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(model)
				AssertEqual(t, "r.ForValue()", r.ForValue(), true)
			}),
			res.Call("method", func(r res.CallRequest) {
				v := r.RequireValue()
				if v != model {
					t.Errorf("expected Value() to return model, but it didn't")
				}
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertResult(t, nil)
	})
}

// Test that Value gets the model as provided from the GetModel resource handler, using With.
func TestModelValueUsingWith(t *testing.T) {
	model := resource["test.model"]

	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Model(model)
			AssertEqual(t, "r.ForValue()", r.ForValue(), true)
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertNoError(t, err)
			if v != model {
				t.Errorf("expected Value() to return model, but it didn't")
			}
			done()
		}))
	})
}

// Test that Value gets the collection as provided from the GetCollection resource handler.
func TestCollectionValue(t *testing.T) {
	collection := resource["test.collection"]

	runTest(t, func(s *Session) {
		s.Handle("collection",
			res.GetCollection(func(r res.CollectionRequest) {
				r.Collection(collection)
				AssertEqual(t, "r.ForValue()", r.ForValue(), true)
			}),
			res.Call("method", func(r res.CallRequest) {
				v, err := r.Value()
				AssertNoError(t, err)
				if v != collection {
					t.Errorf("expected Value() to return collection, but it didn't")
				}
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.collection.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test that Value gets the collection as provided from the GetCollection resource handler.
func TestCollectionRequireValue(t *testing.T) {
	collection := resource["test.collection"]

	runTest(t, func(s *Session) {
		s.Handle("collection",
			res.GetCollection(func(r res.CollectionRequest) {
				r.Collection(collection)
				AssertEqual(t, "r.ForValue()", r.ForValue(), true)
			}),
			res.Call("method", func(r res.CallRequest) {
				v := r.RequireValue()
				if v != collection {
					t.Errorf("expected Value() to return collection, but it didn't")
				}
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.collection.method", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertResult(t, nil)
	})
}

// Test that Value gets the collection as provided from the GetCollection resource handler, using With.
func TestCollectionValueUsingWith(t *testing.T) {
	collection := resource["test.collection"]

	runTestAsync(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Collection(collection)
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.collection", func(r res.Resource) {
			v, err := r.Value()
			AssertNoError(t, err)
			if v != collection {
				t.Errorf("expected Value() to return collection, but it didn't")
			}
			done()
		}))
	})
}

// Test that Value returns an error on missing get handler.
func TestValueWithoutHandler(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err, res.ErrNotFound)
			done()
		}))
	})
}

// Test that calling QueryModel within Value call gets the model.
func TestValueQueryModel(t *testing.T) {
	model := resource["test.model"]

	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.QueryModel(model, "foo=bar")
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model?foo=bar", func(r res.Resource) {
			v, err := r.Value()
			AssertNoError(t, err)
			if v != model {
				t.Errorf("expected Value() to return model, but it didn't")
			}
			done()
		}))
	})
}

// Test that calling QueryCollection within Value call gets the collection.
func TestValueQueryCollection(t *testing.T) {
	collection := resource["test.collection"]

	runTestAsync(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.QueryCollection(collection, "foo=bar")
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.collection?foo=bar", func(r res.Resource) {
			v, err := r.Value()
			AssertNoError(t, err)
			if v != collection {
				t.Errorf("expected Value() to return collection, but it didn't")
			}
			done()
		}))
	})
}

// Test that calling QueryModel for a non query model within Value call causes error.
func TestValueQueryModelOnNonQuery(t *testing.T) {
	model := resource["test.model"]

	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.QueryModel(model, "foo=bar")
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err.(*res.Error).Code, res.CodeInternalError)
			done()
		}))
	})
}

// Test that calling QueryCollection for a non query collection within Value call causes error.
func TestValueQueryCollectionOnNonQuery(t *testing.T) {
	collection := resource["test.collection"]

	runTestAsync(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.QueryCollection(collection, "foo=bar")
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.collection", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err.(*res.Error).Code, res.CodeInternalError)
			done()
		}))
	})
}

// Test that calling NotFound within Value call causes a system.notFound error.
func TestValueNotFound(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.NotFound()
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err, res.ErrNotFound)
			done()
		}))
	})
}

// Test that calling NotFound within Value call causes given error.
func TestValueError(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Error(res.ErrMethodNotFound)
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err, res.ErrMethodNotFound)
			done()
		}))
	})
}

// Test that calling Timeout within Value call has no effect.
func TestValueTimeout(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Timeout(time.Second * 12)
			r.NotFound()
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err, res.ErrNotFound)
			done()
		}))
	})
}

// Test that calling panicking with *Error within Value call causes given error.
func TestValuePanicWithError(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(res.ErrMethodNotFound)
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err, res.ErrMethodNotFound)
			done()
		}))
	})
}

// Test that calling panicking with error within Value call causes system.internalError.
func TestValuePanicWithOsError(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(errors.New("panic"))
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err.(*res.Error).Code, res.CodeInternalError)
			done()
		}))
	})
}

// Test that calling panicking with string within Value call causes system.internalError.
func TestValuePanicWithStringError(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic("panic")
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err.(*res.Error).Code, res.CodeInternalError)
			done()
		}))
	})
}

// Test that calling panicking with generic type within Value call causes system.internalError.
func TestValuePanicWithGenericTypeError(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(42)
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err.(*res.Error).Code, res.CodeInternalError)
			done()
		}))
	})
}

// Test that Value gets an error if the GetModel handler gives no response.
func TestModelValueWithoutResponse(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err.(*res.Error).Code, res.CodeInternalError)
			done()
		}))
	})
}

// Test that Value gets an error if the GetCollection handler gives no response.
func TestCollectionValueWithoutResponse(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetCollection(func(r res.CollectionRequest) {}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err.(*res.Error).Code, res.CodeInternalError)
			done()
		}))
	})
}

// Test that calling Value within GetModel handler causes Value to return system.internalError.
func TestModelValueWithinGetModelHandler(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Value()
			r.NotFound()
		}))
	}, func(s *Session, done func()) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			defer done()
			v, err := r.Value()
			AssertEqual(t, "value", v, nil)
			AssertEqual(t, "error", err.(*res.Error).Code, res.CodeInternalError)
		}))
	})
}
