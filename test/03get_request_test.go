package test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jirenius/go-res"
)

// Test that the model is sent on get request
func TestGetModel(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Handle("model.foo", res.GetModel(func(r res.ModelRequest) {
			AssertEqual(t, "r.ForValue()", r.ForValue(), false)
			AssertEqual(t, "r.ResourceType()", r.ResourceType(), res.TypeModel)
			r.Model(json.RawMessage(model))
		}))
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.model.foo", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))

		// Test getting the model with missing part
		inb = s.Request("get.test.model", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)

		// Test getting the model with extra part
		inb = s.Request("get.test.model.foo.bar", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test that the collection is sent on get request
func TestGetCollection(t *testing.T) {
	collection := resource["test.collection"]

	runTest(t, func(s *Session) {
		s.Handle("collection.foo", res.GetCollection(func(r res.CollectionRequest) {
			AssertEqual(t, "r.ForValue()", r.ForValue(), false)
			AssertEqual(t, "r.ResourceType()", r.ResourceType(), res.TypeCollection)
			r.Collection(json.RawMessage(collection))
		}))
	}, func(s *Session) {
		// Test getting the collection
		inb := s.Request("get.test.collection.foo", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"collection":`+collection+`}}`))

		// Test getting the collection with missing part
		inb = s.Request("get.test.collection", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)

		// Test getting the collection with extra part
		inb = s.Request("get.test.collection.foo.bar", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test that the model is sent on get request
func TestGetResource(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Handle("model.foo", res.GetResource(func(r res.GetRequest) {
			AssertEqual(t, "r.ForValue()", r.ForValue(), false)
			AssertEqual(t, "r.ResourceType()", r.ResourceType(), res.TypeUnset)
			r.Model(json.RawMessage(model))
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.model.foo", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test ResourceType returns TypeModel when using res.Model
func TestGetResourceTypedModel(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Handle("model.foo",
			res.Model,
			res.GetResource(func(r res.GetRequest) {
				AssertEqual(t, "r.ForValue()", r.ForValue(), false)
				AssertEqual(t, "r.ResourceType()", r.ResourceType(), res.TypeModel)
				r.Model(json.RawMessage(model))
			}),
		)
	}, func(s *Session) {
		inb := s.Request("get.test.model.foo", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test ResourceType returns TypeCollection when using res.Collection
func TestGetResourceTypedCollection(t *testing.T) {
	collection := resource["test.collection"]

	runTest(t, func(s *Session) {
		s.Handle("collection.foo",
			res.Collection,
			res.GetResource(func(r res.GetRequest) {
				AssertEqual(t, "r.ForValue()", r.ForValue(), false)
				AssertEqual(t, "r.ResourceType()", r.ResourceType(), res.TypeCollection)
				r.Collection(json.RawMessage(collection))
			}),
		)
	}, func(s *Session) {
		inb := s.Request("get.test.collection.foo", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"collection":`+collection+`}}`))
	})
}

// Test that calling NotFound on a model get request results in system.notFound
func TestGetModelNotFound(t *testing.T) {
	isCalled := false
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			isCalled = true
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.model", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)

		if !isCalled {
			t.Fatalf("expected handler to be called, but it wasn't")
		}
	})
}

// Test that calling NotFound on a collection get request results in system.notFound
func TestGetCollectionNotFound(t *testing.T) {
	isCalled := false
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			isCalled = true
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.collection", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)

		if !isCalled {
			t.Fatalf("expected handler to be called, but it wasn't")
		}
	})
}

// Test that calling Error on a model get request results in given error
func TestGetModelError(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Error(res.ErrMethodNotFound)
		}))
	}, func(s *Session) {
		for i := 0; i < 10; i++ {
			inb := s.Request("get.test.model", newRequest())
			s.GetMsg(t).
				AssertSubject(t, inb).
				AssertError(t, res.ErrMethodNotFound)
		}
	})
}

// Test that calling Error on a collection get request results in given error
func TestGetCollectionError(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Error(res.ErrMethodNotFound)
		}))
	}, func(s *Session) {
		for i := 0; i < 10; i++ {
			inb := s.Request("get.test.collection", newRequest())
			s.GetMsg(t).
				AssertSubject(t, inb).
				AssertError(t, res.ErrMethodNotFound)
		}
	})
}

// Test that panicing in a model get request results in system.internalError
func TestPanicOnGetModel(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic("panic")
		}))
	}, func(s *Session) {
		for i := 0; i < 10; i++ {
			inb := s.Request("get.test.model", newRequest())
			s.GetMsg(t).
				AssertSubject(t, inb).
				AssertErrorCode(t, "system.internalError")
		}
	})
}

// Test that panicing with an Error in a model get request results in the given error
func TestPanicWithErrorOnGetModel(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(res.ErrMethodNotFound)
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.model", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test that panicing with an error in a model get request results in the given error
func TestPanicWithOsErrorOnGetModel(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(errors.New("panic"))
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.model", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertErrorCode(t, res.CodeInternalError)
	})
}

// Test that panicing with a generic value in a model get request results in system.internalError
func TestPanicWithGenericValueOnGetModel(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(42)
		}))
	}, func(s *Session) {
		for i := 0; i < 10; i++ {
			inb := s.Request("get.test.model", newRequest())
			s.GetMsg(t).
				AssertSubject(t, inb).
				AssertErrorCode(t, "system.internalError")
		}
	})
}

// Test that panicing in a collection get request results in system.internalError
func TestPanicOnGetCollection(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic("panic")
		}))
	}, func(s *Session) {
		for i := 0; i < 10; i++ {
			inb := s.Request("get.test.collection", newRequest())
			s.GetMsg(t).
				AssertSubject(t, inb).
				AssertErrorCode(t, "system.internalError")
		}
	})
}

// Test that panicing with an Error in a collection get request results in the given error
func TestPanicWithErrorOnGetCollection(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic(res.ErrMethodNotFound)
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.collection", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test that panicing with an error in a collection get request results in the given error
func TestPanicWithOsErrorOnGetCollection(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic(errors.New("panic"))
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.collection", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertErrorCode(t, res.CodeInternalError)
	})
}

// Test that panicing with a generic value in a collection get request results in the given error
func TestPanicWithGenericValueOnGetCollection(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic(42)
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.collection", newRequest())
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertErrorCode(t, res.CodeInternalError)
	})
}

// Test sending multiple get model requests for the same resource
// and assert they are handled in order
func TestMultipleGetModel(t *testing.T) {
	model := resource["test.model"]
	const requestCount = 100

	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Model(json.RawMessage(model))
		}))
	}, func(s *Session) {
		inbs := make([]string, requestCount)

		// Test getting the model
		for i := 0; i < requestCount; i++ {
			inbs[i] = s.Request("get.test.model", newRequest())
		}

		for _, inb := range inbs {
			s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
		}
	})
}

// Test sending multiple get collection requests for the same resource
// and assert they are handled in order
func TestMultipleGetCollection(t *testing.T) {
	collection := resource["test.collection"]
	const requestCount = 100

	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Collection(json.RawMessage(collection))
		}))
	}, func(s *Session) {
		inbs := make([]string, requestCount)

		// Test getting the collection
		for i := 0; i < requestCount; i++ {
			inbs[i] = s.Request("get.test.collection", newRequest())
		}

		for _, inb := range inbs {
			s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"collection":`+collection+`}}`))
		}
	})
}

// Test sending multiple get model requests for the same resource
// and assert they are handled in order
func TestMultipleGetDifferentResources(t *testing.T) {
	model := resource["test.model"]
	collection := resource["test.collection"]
	const requestCount = 50

	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Model(json.RawMessage(model))
		}))
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Collection(json.RawMessage(collection))
		}))
	}, func(s *Session) {
		minbs := make([]string, requestCount)
		cinbs := make([]string, requestCount)

		// Test getting the resources
		for i := 0; i < requestCount; i++ {
			minbs[i] = s.Request("get.test.model", newRequest())
			cinbs[i] = s.Request("get.test.collection", newRequest())
		}

		var mi, ci int
		for i := 0; i < requestCount*2; i++ {
			m := s.GetMsg(t)
			switch {
			case mi < requestCount && minbs[mi] == m.Subject:
				m.AssertResult(t, json.RawMessage(`{"model":`+model+`}`))
				mi++
			case ci < requestCount && cinbs[ci] == m.Subject:
				m.AssertResult(t, json.RawMessage(`{"collection":`+collection+`}`))
				ci++
			default:
				t.Fatalf("expected message subject to be a for a collection or model requestion, but got %#v", m.Subject)
			}
		}
	})
}

// Test that Timeout sends the pre-response with timeout on a model get request.
func TestGetModelRequestTimeout(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.model", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertRawPayload(t, []byte(`timeout:"42000"`))
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero on a model get request.
func TestGetModelRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
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
		inb := s.Request("get.test.model", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, "system.internalError")
	})
}

// Test that Timeout sends the pre-response with timeout on a collection get request.
func TestGetCollectionRequestTimeout(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.collection", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertRawPayload(t, []byte(`timeout:"42000"`))
		s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero on a collection get request.
func TestGetCollectionRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
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
		inb := s.Request("get.test.collection", nil)
		s.GetMsg(t).AssertSubject(t, inb).AssertErrorCode(t, "system.internalError")
	})
}
