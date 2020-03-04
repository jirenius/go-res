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

func newRIDStore() *mockstore.Store {
	return &mockstore.Store{
		Resources: map[string]interface{}{
			"test.model":      mock.Model,
			"test.collection": mock.Collection,
		},
		OnValue: func(st *mockstore.Store, id string) (interface{}, error) {
			if id == "test.error" {
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

func TestStoreHandler_GetModel_ReturnsModel(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Model,
			store.Handler{}.WithStore(newRIDStore()),
		)
	}, func(s *restest.Session) {
		s.Get("test.model").
			Response().
			AssertModel(mock.Model)
	})
}

func TestStoreHandler_GetCollection_ReturnsCollection(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("collection",
			res.Collection,
			store.Handler{}.WithStore(newRIDStore()),
		)
	}, func(s *restest.Session) {
		s.Get("test.collection").
			Response().
			AssertCollection(mock.Collection)
	})
}

func TestStoreHandler_GetMissingModel_ReturnsNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("missing",
			res.Model,
			store.Handler{}.WithStore(newRIDStore()),
		)
	}, func(s *restest.Session) {
		s.Get("test.missing").
			Response().
			AssertError(res.ErrNotFound)
	})
}

func TestStoreHandler_GetMissingCollection_ReturnsNotFound(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("missing",
			res.Collection,
			store.Handler{}.WithStore(newRIDStore()),
		)
	}, func(s *restest.Session) {
		s.Get("test.missing").
			Response().
			AssertError(res.ErrNotFound)
	})
}

func TestStoreHandler_GetErrorModel_ReturnsError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("error",
			res.Model,
			store.Handler{}.WithStore(newRIDStore()),
		)
	}, func(s *restest.Session) {
		s.Get("test.error").
			Response().
			AssertError(mock.CustomError)
	})
}

func TestStoreHandler_GetErrorCollection_ReturnsError(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("error",
			res.Collection,
			store.Handler{}.WithStore(newRIDStore()),
		)
	}, func(s *restest.Session) {
		s.Get("test.error").
			Response().
			AssertError(mock.CustomError)
	})
}

var testSetModelTbl = []struct {
	Model    interface{}
	Update   interface{}
	Expected interface{}
}{
	// Generic
	{mock.Model, json.RawMessage(`{"id":42,"foo":"bar"}`), nil},
	{mock.Model, json.RawMessage(`{"id":42,"foo":"baz"}`), json.RawMessage(`{"foo":"baz"}`)},
	{mock.Model, json.RawMessage(`{"id":43,"foo":"bar"}`), json.RawMessage(`{"id":43}`)},
	{mock.Model, json.RawMessage(`{"id":43,"foo":"baz"}`), json.RawMessage(`{"id":43,"foo":"baz"}`)},
	{mock.Model, json.RawMessage(`{"id":42,"foo":"bar","zoo":"baz"}`), json.RawMessage(`{"zoo":"baz"}`)},
	{mock.Model, json.RawMessage(`{"id":42}`), json.RawMessage(`{"foo":{"action":"delete"}}`)},
	// Modified fields of different types
	{json.RawMessage(`{"bool":false}`), json.RawMessage(`{"bool":true}`), json.RawMessage(`{"bool":true}`)},
	{json.RawMessage(`{"number":0}`), json.RawMessage(`{"number":1}`), json.RawMessage(`{"number":1}`)},
	{json.RawMessage(`{"string":""}`), json.RawMessage(`{"string":"a"}`), json.RawMessage(`{"string":"a"}`)},
	{json.RawMessage(`{"null":null}`), json.RawMessage(`{"null":false}`), json.RawMessage(`{"null":false}`)},
	{json.RawMessage(`{"ref":{"rid":"test.ref"}}`), json.RawMessage(`{"ref":{"rid":"test.mod"}}`), json.RawMessage(`{"ref":{"rid":"test.mod"}}`)},
	// New fields of different types
	{json.RawMessage(`{}`), json.RawMessage(`{"bool":true}`), json.RawMessage(`{"bool":true}`)},
	{json.RawMessage(`{}`), json.RawMessage(`{"number":1}`), json.RawMessage(`{"number":1}`)},
	{json.RawMessage(`{}`), json.RawMessage(`{"string":"a"}`), json.RawMessage(`{"string":"a"}`)},
	{json.RawMessage(`{}`), json.RawMessage(`{"null":false}`), json.RawMessage(`{"null":false}`)},
	{json.RawMessage(`{}`), json.RawMessage(`{"ref":{"rid":"test.ref"}}`), json.RawMessage(`{"ref":{"rid":"test.ref"}}`)},
	// Deleted fields of different types
	{json.RawMessage(`{"bool":false}`), json.RawMessage(`{}`), json.RawMessage(`{"bool":{"action":"delete"}}`)},
	{json.RawMessage(`{"number":0}`), json.RawMessage(`{}`), json.RawMessage(`{"number":{"action":"delete"}}`)},
	{json.RawMessage(`{"string":""}`), json.RawMessage(`{}`), json.RawMessage(`{"string":{"action":"delete"}}`)},
	{json.RawMessage(`{"null":null}`), json.RawMessage(`{}`), json.RawMessage(`{"null":{"action":"delete"}}`)},
	{json.RawMessage(`{"ref":{"rid":"test.ref"}}`), json.RawMessage(`{}`), json.RawMessage(`{"ref":{"action":"delete"}}`)},
}

func TestStoreHandler_UpdateModel_ExpectedEvent(t *testing.T) {
	for i, l := range testSetModelTbl {
		test := fmt.Sprintf("test #%d", i)
		st := mockstore.NewStore().Add("test.model", l.Model)

		runTest(t, func(s *res.Service) {
			s.Handle("model",
				res.Model,
				store.Handler{}.WithStore(st),
			)
		}, func(s *restest.Session) {
			func() {
				txn := st.Write("test.model")
				defer txn.Close()
				restest.AssertNoError(t, txn.Update(l.Update), test)
				// Assert we get expected change event
				if l.Expected != nil {
					s.GetMsg().
						AssertChangeEvent("test.model", l.Expected)
				}
			}()
			// Assert there are no more events
			s.Get("test.model").
				Response().
				AssertModel(l.Update)
		}, restest.WithTest(test))
	}
}

var testSetCollectionTbl = []struct {
	Collection     interface{}
	Update         interface{}
	ExpectedEvents []store.ResultEvent
}{
	// No change
	{mock.Collection, json.RawMessage(`[42,"foo",null]`), nil},
	// Remove
	{mock.Collection, json.RawMessage(`["foo",null]`), []store.ResultEvent{
		{Name: "remove", Idx: 0},
	}},
	{mock.Collection, json.RawMessage(`[42,null]`), []store.ResultEvent{
		{Name: "remove", Idx: 1},
	}},
	{mock.Collection, json.RawMessage(`[42,"foo"]`), []store.ResultEvent{
		{Name: "remove", Idx: 2},
	}},
	// Add
	{mock.Collection, json.RawMessage(`[true,42,"foo",null]`), []store.ResultEvent{
		{Name: "add", Value: true, Idx: 0},
	}},
	{mock.Collection, json.RawMessage(`[42,false,"foo",null]`), []store.ResultEvent{
		{Name: "add", Value: false, Idx: 1},
	}},
	{mock.Collection, json.RawMessage(`[42,"foo",null,{"rid":"test.ref"}]`), []store.ResultEvent{
		{Name: "add", Value: res.Ref("test.ref"), Idx: 3},
	}},
	// Replace
	{mock.Collection, json.RawMessage(`[true,"foo",null]`), []store.ResultEvent{
		{Name: "remove", Idx: 0},
		{Name: "add", Value: true, Idx: 0},
	}},
	{mock.Collection, json.RawMessage(`[42,false,null]`), []store.ResultEvent{
		{Name: "remove", Idx: 1},
		{Name: "add", Value: false, Idx: 1},
	}},
	{mock.Collection, json.RawMessage(`[42,"foo",{"rid":"test.ref"}]`), []store.ResultEvent{
		{Name: "remove", Idx: 2},
		{Name: "add", Value: res.Ref("test.ref"), Idx: 2},
	}},
	// Move
	{mock.Collection, json.RawMessage(`["foo",null,42]`), []store.ResultEvent{
		{Name: "remove", Idx: 0},
		{Name: "add", Value: 42, Idx: 2},
	}},
	{mock.Collection, json.RawMessage(`["foo",42,null]`), []store.ResultEvent{
		{Name: "remove", Idx: 0},
		{Name: "add", Value: 42, Idx: 1},
	}},
	{mock.Collection, json.RawMessage(`[42,null,"foo"]`), []store.ResultEvent{
		{Name: "remove", Idx: 1},
		{Name: "add", Value: "foo", Idx: 2},
	}},
}

func TestStoreHandler_UpdateCollection_ExpectedEvents(t *testing.T) {
	for i, l := range testSetCollectionTbl {
		test := fmt.Sprintf("test #%d", i)
		st := mockstore.NewStore().Add("test.collection", l.Collection)

		runTest(t, func(s *res.Service) {
			s.Handle("collection",
				res.Collection,
				store.Handler{}.WithStore(st),
			)
		}, func(s *restest.Session) {
			func() {
				txn := st.Write("test.collection")
				defer txn.Close()
				restest.AssertNoError(t, txn.Update(l.Update), test)
				// Assert we get expected add/remove events
				for _, ev := range l.ExpectedEvents {
					switch ev.Name {
					case "add":
						s.GetMsg().AssertAddEvent("test.collection", ev.Value, ev.Idx)
					case "remove":
						s.GetMsg().AssertRemoveEvent("test.collection", ev.Idx)
					default:
						panic("invalid test data: event name " + ev.Name)
					}
				}

			}()
			// Assert there are no more events
			s.Get("test.collection").
				Response().
				AssertCollection(l.Update)
		}, restest.WithTest(test))
	}
}
