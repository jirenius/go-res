package test

import (
	"encoding/json"
	"fmt"
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
	"github.com/jirenius/go-res/store"
	"github.com/jirenius/go-res/store/mockstore"
)

func newModelStore() *mockstore.Store {
	return &mockstore.Store{
		Resources: map[string]interface{}{
			"1": *mock.Model,
		},
		OnValue: func(st *mockstore.Store, id string) (interface{}, error) {
			if id == "error" {
				return nil, mock.CustomError
			}
			v, ok := st.Resources[id]
			if !ok {
				return nil, store.ErrNotFound
			}
			return v, nil
		},
	}
}

func newCollectionStore() *mockstore.Store {
	return &mockstore.Store{
		Resources: map[string]interface{}{
			"1": mock.Collection,
		},
		OnValue: func(st *mockstore.Store, id string) (interface{}, error) {
			if id == "error" {
				return nil, mock.CustomError
			}
			v, ok := st.Resources[id]
			if !ok {
				return nil, store.ErrNotFound
			}
			return v, nil
		},
	}
}

func TestStoreHandlerTransformer_GetModel_ReturnsModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id",
			res.Model,
			store.Handler{}.
				WithStore(newModelStore()).
				WithTransformer(store.IDTransformer("id", nil)),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.1").
			Response().
			AssertModel(mock.Model)
	})
}

func TestStoreHandlerTransformer_GetCollection_ReturnsCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.$id",
			res.Collection,
			store.Handler{}.
				WithStore(newCollectionStore()).
				WithTransformer(store.IDTransformer("id", nil)),
		)
	}, func(s *restest.Session) {
		s.Get("test.collection.1").
			Response().
			AssertCollection(mock.Collection)
	})
}

func TestStoreHandlerTransformer_GetMissingModel_ReturnsNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id",
			res.Model,
			store.Handler{}.
				WithStore(newModelStore()).
				WithTransformer(store.IDTransformer("id", nil)),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.3").
			Response().
			AssertError(res.ErrNotFound)
	})
}

func TestStoreHandlerTransformer_GetMissingCollection_ReturnsNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.$id",
			res.Collection,
			store.Handler{}.
				WithStore(newCollectionStore()).
				WithTransformer(store.IDTransformer("id", nil)),
		)
	}, func(s *restest.Session) {
		s.Get("test.collection.3").
			Response().
			AssertError(res.ErrNotFound)
	})
}

func TestStoreHandlerTransformer_GetErrorModel_ReturnsError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id",
			res.Model,
			store.Handler{}.
				WithStore(newModelStore()).
				WithTransformer(store.IDTransformer("id", nil)),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.error").
			Response().
			AssertError(mock.CustomError)
	})
}

func TestStoreHandlerTransformer_GetErrorCollection_ReturnsError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.$id",
			res.Collection,
			store.Handler{}.
				WithStore(newCollectionStore()).
				WithTransformer(store.IDTransformer("id", nil)),
		)
	}, func(s *restest.Session) {
		s.Get("test.collection.error").
			Response().
			AssertError(mock.CustomError)
	})
}

func TestStoreHandlerTransformer_UpdateModel_ExpectedEvent(t *testing.T) {
	for i, l := range testSetModelTbl {
		test := fmt.Sprintf("test #%d", i)
		st := mockstore.NewStore().Add("1", l.Model)

		runTest(t, func(s *res.Service) {
			s.Handle("model.$id",
				res.Model,
				store.Handler{}.
					WithStore(st).
					WithTransformer(store.IDTransformer("id", nil)),
			)
		}, func(s *restest.Session) {
			func() {
				txn := st.Write("1")
				defer txn.Close()
				restest.AssertNoError(t, txn.Update(l.Update), test)
				// Assert we get expected change event
				if l.Expected != nil {
					s.GetMsg().
						AssertChangeEvent("test.model.1", l.Expected)
				}
			}()
			// Assert there are no more events
			s.Get("test.model.1").
				Response().
				AssertModel(l.Update)
		}, restest.WithTest(test))
	}
}

func TestStoreHandlerTransformer_UpdateCollection_ExpectedEvents(t *testing.T) {
	for i, l := range testSetCollectionTbl {
		test := fmt.Sprintf("test #%d", i)
		st := mockstore.NewStore().Add("1", l.Collection)

		runTest(t, func(s *res.Service) {
			s.Handle("collection.$id",
				res.Collection,
				store.Handler{}.
					WithStore(st).
					WithTransformer(store.IDTransformer("id", nil)),
			)
		}, func(s *restest.Session) {
			func() {
				txn := st.Write("1")
				defer txn.Close()
				restest.AssertNoError(t, txn.Update(l.Update), test)
				// Assert we get expected add/remove events
				for _, ev := range l.ExpectedEvents {
					switch ev.Name {
					case "add":
						s.GetMsg().AssertAddEvent("test.collection.1", ev.Value, ev.Idx)
					case "remove":
						s.GetMsg().AssertRemoveEvent("test.collection.1", ev.Idx)
					default:
						panic("invalid test data: event name " + ev.Name)
					}
				}

			}()
			// Assert there are no more events
			s.Get("test.collection.1").
				Response().
				AssertCollection(l.Update)
		}, restest.WithTest(test))
	}
}

func TestStoreHandlerTransformer_GetModelWithTransform_ReturnsTransformedModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id",
			res.Model,
			store.Handler{}.
				WithStore(newCollectionStore()).
				WithTransformer(store.IDTransformer("id", func(id string, value interface{}) (interface{}, error) {
					restest.AssertEqualJSON(t, "transformer id", id, "1")
					restest.AssertEqualJSON(t, "transformer value", value, mock.Collection)
					return mock.Model, nil
				})),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.1").
			Response().
			AssertModel(mock.Model)
	})
}

func TestStoreHandlerTransformer_GetCollectionWithTransform_ReturnsTransformedCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.$id",
			res.Collection,
			store.Handler{}.
				WithStore(newModelStore()).
				WithTransformer(store.IDTransformer("id", func(id string, value interface{}) (interface{}, error) {
					restest.AssertEqualJSON(t, "transformer id", id, "1")
					restest.AssertEqualJSON(t, "transformer value", value, mock.Model)
					return mock.Collection, nil
				})),
		)
	}, func(s *restest.Session) {
		s.Get("test.collection.1").
			Response().
			AssertCollection(mock.Collection)
	})
}

func TestStoreHandlerTransformer_GetModelWithTransformError_ReturnsError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id",
			res.Model,
			store.Handler{}.
				WithStore(newCollectionStore()).
				WithTransformer(store.IDTransformer("id", func(id string, value interface{}) (interface{}, error) {
					return nil, mock.CustomError
				})),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.1").
			Response().
			AssertError(mock.CustomError)
	})
}

func TestStoreHandlerTransformer_GetCollectionWithTransformError_ReturnsError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection.$id",
			res.Collection,
			store.Handler{}.
				WithStore(newModelStore()).
				WithTransformer(store.IDTransformer("id", func(id string, value interface{}) (interface{}, error) {
					return nil, mock.CustomError
				})),
		)
	}, func(s *restest.Session) {
		s.Get("test.collection.1").
			Response().
			AssertError(mock.CustomError)
	})
}

func TestStoreHandlerTransformer_UpdateModelWithTransformer_ExpectedEvent(t *testing.T) {
	st := mockstore.NewStore().Add("1", map[string]interface{}{"id": 42, "firstName": "Foo", "lastName": "Bar", "deleted": false})

	runTest(t, func(s *res.Service) {
		s.Handle("model.$id",
			res.Model,
			store.Handler{}.
				WithStore(st).
				WithTransformer(store.IDTransformer("id", func(id string, value interface{}) (interface{}, error) {
					m := value.(map[string]interface{})
					return map[string]interface{}{"id": m["id"], "name": m["firstName"].(string) + " " + m["lastName"].(string)}, nil
				})),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.1").
			Response().
			AssertModel(json.RawMessage(`{"id":42,"name":"Foo Bar"}`))
		func() {
			txn := st.Write("1")
			defer txn.Close()
			restest.AssertNoError(t, txn.Update(map[string]interface{}{"id": 42, "firstName": "Zoo", "lastName": "Baz", "deleted": true}))
			// Assert we get expected change event
			s.GetMsg().
				AssertChangeEvent("test.model.1", json.RawMessage(`{"name":"Zoo Baz"}`))
		}()
		// Assert there are no more events
		s.Get("test.model.1").
			Response().
			AssertModel(json.RawMessage(`{"id":42,"name":"Zoo Baz"}`))
	})
}

func TestStoreHandlerTransformer_UpdateCollectionWithTransformer_ExpectedEvent(t *testing.T) {
	st := mockstore.NewStore().Add("1", []interface{}{42, "Foo", "Bar", false})

	runTest(t, func(s *res.Service) {
		s.Handle("collection.$id",
			res.Collection,
			store.Handler{}.
				WithStore(st).
				WithTransformer(store.IDTransformer("id", func(id string, value interface{}) (interface{}, error) {
					a := value.([]interface{})
					return []interface{}{a[0], a[1].(string) + " " + a[2].(string)}, nil
				})),
		)
	}, func(s *restest.Session) {
		s.Get("test.collection.1").
			Response().
			AssertCollection(json.RawMessage(`[42,"Foo Bar"]`))
		func() {
			txn := st.Write("1")
			defer txn.Close()
			restest.AssertNoError(t, txn.Update([]interface{}{42, "Zoo", "Baz", true}))
			// Assert we get expected remove and add events
			s.GetMsg().AssertRemoveEvent("test.collection.1", 1)
			s.GetMsg().AssertAddEvent("test.collection.1", "Zoo Baz", 1)
		}()
		// Assert there are no more events
		s.Get("test.collection.1").
			Response().
			AssertCollection(json.RawMessage(`[42,"Zoo Baz"]`))
	})
}
