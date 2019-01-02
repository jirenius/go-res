package test

import (
	"encoding/json"
	"testing"

	"github.com/jirenius/go-res"
)

// Test that the model is sent on get request
func TestGetModel(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Handle("model.foo", res.GetModel(func(r res.ModelRequest) {
			r.Model(json.RawMessage(model))
		}))
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.model.foo", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))

		// Test getting the model with missing part
		inb = s.Request("get.test.model", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)

		// Test getting the model with extra part
		inb = s.Request("get.test.model.foo.bar", nil)
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
			r.Collection(json.RawMessage(collection))
		}))
	}, func(s *Session) {
		// Test getting the collection
		inb := s.Request("get.test.collection.foo", nil)
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"collection":`+collection+`}}`))

		// Test getting the collection with missing part
		inb = s.Request("get.test.collection", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)

		// Test getting the collection with extra part
		inb = s.Request("get.test.collection.foo.bar", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrNotFound)
	})
}

// Test that calling NotFound on a model get request results in system.notFound
func TestGetModelNotFound(t *testing.T) {
	isCalled := false
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.NotFound()
			isCalled = true
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.model", nil)
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
			r.NotFound()
			isCalled = true
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.collection", nil)
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
			inb := s.Request("get.test.model", nil)
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
			inb := s.Request("get.test.collection", nil)
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
			panic("Panic!")
		}))
	}, func(s *Session) {
		for i := 0; i < 10; i++ {
			inb := s.Request("get.test.model", nil)
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
			panic("Panic!")
		}))
	}, func(s *Session) {
		for i := 0; i < 10; i++ {
			inb := s.Request("get.test.collection", nil)
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
		inb := s.Request("get.test.model", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
	})
}

// Test that panicing with an Error in a collection get request results in the given error
func TestPanicWithErrorOnGetCollection(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic(res.ErrMethodNotFound)
		}))
	}, func(s *Session) {
		inb := s.Request("get.test.collection", nil)
		s.GetMsg(t).
			AssertSubject(t, inb).
			AssertError(t, res.ErrMethodNotFound)
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
			inbs[i] = s.Request("get.test.model", nil)
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
			inbs[i] = s.Request("get.test.collection", nil)
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
			minbs[i] = s.Request("get.test.model", nil)
			cinbs[i] = s.Request("get.test.collection", nil)
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
