package test

import (
	"errors"
	"testing"
	"time"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test that the model is sent on get request
func TestGetModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo", res.GetModel(func(r res.ModelRequest) {
			restest.AssertEqualJSON(t, "r.ForValue()", r.ForValue(), false)
			restest.AssertEqualJSON(t, "r.ResourceType()", r.ResourceType(), res.TypeModel)
			r.Model(mock.Model)
		}))
	}, func(s *restest.Session) {
		// Test getting the model
		s.Get("test.model.foo").
			Response().
			AssertModel(mock.Model)

		// Test getting the model with missing part
		s.Get("test.model").
			Response().
			AssertError(res.ErrNotFound)

		// Test getting the model with extra part
		s.Get("test.model.foo.bar").
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test that the collection is sent on get request
func TestGetCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.foo", res.GetCollection(func(r res.CollectionRequest) {
			restest.AssertEqualJSON(t, "r.ForValue()", r.ForValue(), false)
			restest.AssertEqualJSON(t, "r.ResourceType()", r.ResourceType(), res.TypeCollection)
			r.Collection(mock.Collection)
		}))
	}, func(s *restest.Session) {
		// Test getting the collection
		s.Get("test.collection.foo").
			Response().
			AssertCollection(mock.Collection)

		// Test getting the collection with missing part
		s.Get("test.collection").
			Response().
			AssertError(res.ErrNotFound)

		// Test getting the collection with extra part
		s.Get("test.collection.foo.bar").
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test that the model is sent on get request
func TestGetResource(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo", res.GetResource(func(r res.GetRequest) {
			restest.AssertEqualJSON(t, "r.ForValue()", r.ForValue(), false)
			restest.AssertEqualJSON(t, "r.ResourceType()", r.ResourceType(), res.TypeUnset)
			r.Model(mock.Model)
		}))
	}, func(s *restest.Session) {
		s.Get("test.model.foo").
			Response().
			AssertModel(mock.Model)
	})
}

// Test ResourceType returns TypeModel when using res.Model
func TestGetResourceTypedModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo",
			res.Model,
			res.GetResource(func(r res.GetRequest) {
				restest.AssertEqualJSON(t, "r.ForValue()", r.ForValue(), false)
				restest.AssertEqualJSON(t, "r.ResourceType()", r.ResourceType(), res.TypeModel)
				r.Model(mock.Model)
			}),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.foo").
			Response().
			AssertModel(mock.Model)
	})
}

// Test ResourceType returns TypeCollection when using res.Collection
func TestGetResourceTypedCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.foo",
			res.Collection,
			res.GetResource(func(r res.GetRequest) {
				restest.AssertEqualJSON(t, "r.ForValue()", r.ForValue(), false)
				restest.AssertEqualJSON(t, "r.ResourceType()", r.ResourceType(), res.TypeCollection)
				r.Collection(mock.Collection)
			}),
		)
	}, func(s *restest.Session) {
		s.Get("test.collection.foo").
			Response().
			AssertCollection(mock.Collection)
	})
}

// Test that calling NotFound on a model get request results in system.notFound
func TestGetModelNotFound(t *testing.T) {
	isCalled := false
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			isCalled = true
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Get("test.model").
			Response().
			AssertError(res.ErrNotFound)

		if !isCalled {
			t.Fatalf("expected handler to be called, but it wasn't")
		}
	})
}

// Test that calling NotFound on a collection get request results in system.notFound
func TestGetCollectionNotFound(t *testing.T) {
	isCalled := false
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			isCalled = true
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		s.Get("test.collection").
			Response().
			AssertError(res.ErrNotFound)

		if !isCalled {
			t.Fatalf("expected handler to be called, but it wasn't")
		}
	})
}

// Test that calling Error on a model get request results in given error
func TestGetModelError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Error(res.ErrMethodNotFound)
		}))
	}, func(s *restest.Session) {
		for i := 0; i < 10; i++ {
			s.Get("test.model").
				Response().
				AssertError(res.ErrMethodNotFound)
		}
	})
}

// Test that calling Error on a collection get request results in given error
func TestGetCollectionError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Error(res.ErrMethodNotFound)
		}))
	}, func(s *restest.Session) {
		for i := 0; i < 10; i++ {
			s.Get("test.collection").
				Response().
				AssertError(res.ErrMethodNotFound)
		}
	})
}

// Test calling InvalidQuery with no message on an model get request results in system.invalidQuery
func TestModelInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *restest.Session) {
		s.Get("test.model?zoo=baz&foo=bar").
			Response().
			AssertError(res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on an model get request results in system.invalidQuery
func TestModelInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *restest.Session) {
		s.Get("test.model?zoo=baz&foo=bar").
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test calling InvalidQuery with no message on an collection get request results in system.invalidQuery
func TestCollectionInvalidQuery_EmptyMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.InvalidQuery("")
		}))
	}, func(s *restest.Session) {
		s.Get("test.collection?zoo=baz&foo=bar").
			Response().
			AssertError(res.ErrInvalidQuery)
	})
}

// Test calling InvalidQuery on an collection get request results in system.invalidQuery
func TestCollectionInvalidQuery_CustomMessage(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.InvalidQuery(mock.ErrorMessage)
		}))
	}, func(s *restest.Session) {
		s.Get("test.collection?zoo=baz&foo=bar").
			Response().
			AssertError(&res.Error{
				Code:    res.CodeInvalidQuery,
				Message: mock.ErrorMessage,
			})
	})
}

// Test that panicing in a model get request results in system.internalError
func TestPanicOnGetModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic("panic")
		}))
	}, func(s *restest.Session) {
		for i := 0; i < 10; i++ {
			s.Get("test.model").
				Response().
				AssertErrorCode("system.internalError")
		}
	})
}

// Test that panicing with an Error in a model get request results in the given error
func TestPanicWithErrorOnGetModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(res.ErrMethodNotFound)
		}))
	}, func(s *restest.Session) {
		s.Get("test.model").
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test that panicing with an error in a model get request results in the given error
func TestPanicWithOsErrorOnGetModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(errors.New("panic"))
		}))
	}, func(s *restest.Session) {
		s.Get("test.model").
			Response().
			AssertErrorCode(res.CodeInternalError)
	})
}

// Test that panicing with a generic value in a model get request results in system.internalError
func TestPanicWithGenericValueOnGetModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			panic(42)
		}))
	}, func(s *restest.Session) {
		for i := 0; i < 10; i++ {
			s.Get("test.model").
				Response().
				AssertErrorCode("system.internalError")
		}
	})
}

// Test that panicing in a collection get request results in system.internalError
func TestPanicOnGetCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic("panic")
		}))
	}, func(s *restest.Session) {
		for i := 0; i < 10; i++ {
			s.Get("test.collection").
				Response().
				AssertErrorCode("system.internalError")
		}
	})
}

// Test that panicing with an Error in a collection get request results in the given error
func TestPanicWithErrorOnGetCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic(res.ErrMethodNotFound)
		}))
	}, func(s *restest.Session) {
		s.Get("test.collection").
			Response().
			AssertError(res.ErrMethodNotFound)
	})
}

// Test that panicing with an error in a collection get request results in the given error
func TestPanicWithOsErrorOnGetCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic(errors.New("panic"))
		}))
	}, func(s *restest.Session) {
		s.Get("test.collection").
			Response().
			AssertErrorCode(res.CodeInternalError)
	})
}

// Test that panicing with a generic value in a collection get request results in the given error
func TestPanicWithGenericValueOnGetCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			panic(42)
		}))
	}, func(s *restest.Session) {
		s.Get("test.collection").
			Response().
			AssertErrorCode(res.CodeInternalError)
	})
}

// Test sending multiple get model requests for the same resource
// and assert they are handled in order
func TestMultipleGetModel(t *testing.T) {
	const requestCount = 100

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Model(mock.Model)
		}))
	}, func(s *restest.Session) {
		reqs := make([]*restest.NATSRequest, requestCount)

		// Test getting the model
		for i := 0; i < requestCount; i++ {
			reqs[i] = s.Get("test.model")
		}

		for _, req := range reqs {
			req.Response().AssertModel(mock.Model)
		}
	})
}

// Test sending multiple get collection requests for the same resource
// and assert they are handled in order
func TestMultipleGetCollection(t *testing.T) {
	const requestCount = 100

	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Collection(mock.Collection)
		}))
	}, func(s *restest.Session) {
		reqs := make([]*restest.NATSRequest, requestCount)

		// Test getting the collection
		for i := 0; i < requestCount; i++ {
			reqs[i] = s.Get("test.collection")
		}

		for _, req := range reqs {
			req.Response().AssertCollection(mock.Collection)
		}
	})
}

// Test sending multiple get model requests for the same resource
// and assert they are handled in order
func TestMultipleGetDifferentResources(t *testing.T) {
	const requestCount = 50

	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Model(mock.Model)
		}))
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Collection(mock.Collection)
		}))
	}, func(s *restest.Session) {
		minbs := make([]string, requestCount)
		cinbs := make([]string, requestCount)

		// Test getting the resources
		for i := 0; i < requestCount; i++ {
			minbs[i] = s.RequestRaw("get.test.model", nil)
			cinbs[i] = s.RequestRaw("get.test.collection", nil)
		}

		var mi, ci int
		for i := 0; i < requestCount*2; i++ {
			m := s.GetMsg()
			switch {
			case mi < requestCount && minbs[mi] == m.Subject:
				m.AssertModel(mock.Model)
				mi++
			case ci < requestCount && cinbs[ci] == m.Subject:
				m.AssertCollection(mock.Collection)
				ci++
			default:
				t.Fatalf("expected message subject to be a for a collection or model request, but got %#v", m.Subject)
			}
		}
	})
}

// Test that Timeout sends the pre-response with timeout on a model get request.
func TestGetModelRequestTimeout(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := s.Get("test.model")
		req.Response().AssertRawPayload([]byte(`timeout:"42000"`))
		req.Response().AssertError(res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero on a model get request.
func TestGetModelRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *res.Service) {
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
	}, func(s *restest.Session) {
		s.Get("test.model").
			Response().
			AssertErrorCode("system.internalError")
	})
}

// Test that Timeout sends the pre-response with timeout on a collection get request.
func TestGetCollectionRequestTimeout(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.Timeout(time.Second * 42)
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		req := s.Get("test.collection")
		req.Response().AssertRawPayload([]byte(`timeout:"42000"`))
		req.Response().AssertError(res.ErrNotFound)
	})
}

// Test that Timeout panics if duration is less than zero on a collection get request.
func TestGetCollectionRequestTimeoutWithDurationLessThanZero(t *testing.T) {
	runTest(t, func(s *res.Service) {
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
	}, func(s *restest.Session) {
		s.Get("test.collection").
			Response().
			AssertErrorCode("system.internalError")
	})
}

func TestGetModelQuery_WithQueryModel_SendsQueryModelResponse(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo", res.GetModel(func(r res.ModelRequest) {
			r.QueryModel(mock.Model, mock.NormalizedQuery)
		}))
	}, func(s *restest.Session) {
		s.Get("test.model.foo?" + mock.Query).
			Response().
			AssertPayload(mock.QueryModelResponse)
	})
}

func TestGetRequest_InvalidJSON_RespondsWithInternalError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo", res.GetModel(func(r res.ModelRequest) {
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		inb := s.RequestRaw("get.test.model.foo", mock.BrokenJSON)
		s.GetMsg().
			AssertSubject(inb).
			AssertErrorCode(res.CodeInternalError)
	})
}

func TestGetCollectionQuery_WithQueryCollection_SendsQueryCollectionResponse(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.foo", res.GetCollection(func(r res.CollectionRequest) {
			r.QueryCollection(mock.Collection, mock.NormalizedQuery)
		}))
	}, func(s *restest.Session) {
		s.Get("test.collection.foo?" + mock.Query).
			Response().
			AssertPayload(mock.QueryCollectionResponse)
	})
}

func TestGetModelQuery_WithoutQueryModel_SendsQueryModelResponse(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo", res.GetModel(func(r res.ModelRequest) {
			r.QueryModel(mock.Model, mock.NormalizedQuery)
		}))
	}, func(s *restest.Session) {
		s.Get("test.model.foo").
			Response().
			AssertPayload(mock.QueryModelResponse)
	})
}

func TestGetCollectionQuery_WithoutQueryCollection_SendsQueryCollectionResponse(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.foo", res.GetCollection(func(r res.CollectionRequest) {
			r.QueryCollection(mock.Collection, mock.NormalizedQuery)
		}))
	}, func(s *restest.Session) {
		s.Get("test.collection.foo").
			Response().
			AssertPayload(mock.QueryCollectionResponse)
	})
}

// Test that a get request without any get handler gives error system.notFound
func TestGet_WithoutGetHandler_SendsNotFoundError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("dummy", func(r res.CallRequest) {
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		s.Get("test.model").
			Response().
			AssertError(res.ErrNotFound)
	})
}

// Test that multiple responses to get request causes panic
func TestGet_WithMultipleResponses_CausesPanic(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.Model(mock.Model)
			restest.AssertPanic(t, func() {
				r.NotFound()
			})
		}))
	}, func(s *restest.Session) {
		s.Get("test.model").
			Response().
			AssertModel(mock.Model)
	})
}
