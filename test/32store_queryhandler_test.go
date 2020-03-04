package test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
	"github.com/jirenius/go-res/store"
	"github.com/jirenius/go-res/store/mockstore"
)

func TestStoreQueryHandler_GetWithRequestHandlerNoPathParams_QueriesQueryStore(t *testing.T) {
	qst := mockstore.NewQueryStore(func(query url.Values) (interface{}, error) {
		restest.AssertNil(t, query)
		return mock.Model, nil
	})
	runTest(t, func(s *res.Service) {
		s.Handle("model.foo",
			res.Model,
			store.QueryHandler{}.
				WithQueryStore(qst).
				WithRequestHandler(func(resourceName string, pathParams map[string]string) (url.Values, error) {
					restest.AssertEqualJSON(t, "resourceName", resourceName, "test.model.foo")
					restest.AssertEqualJSON(t, "pathParams", pathParams, nil)
					return nil, nil
				}),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.foo").
			Response().
			AssertModel(mock.Model)
	})
}

func TestStoreQueryHandler_GetWithRequestHandlerAndPathParams_QueriesQueryStore(t *testing.T) {
	qst := mockstore.NewQueryStore(func(query url.Values) (interface{}, error) {
		restest.AssertEqualJSON(t, "query", query, url.Values{"id": {"foo"}})
		return mock.Model, nil
	})
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id",
			res.Model,
			store.QueryHandler{}.
				WithQueryStore(qst).
				WithRequestHandler(func(resourceName string, pathParams map[string]string) (url.Values, error) {
					restest.AssertEqualJSON(t, "resourceName", resourceName, "test.model.foo")
					restest.AssertEqualJSON(t, "pathParams", pathParams, map[string]string{"id": "foo"})
					return url.Values{"id": {"foo"}}, nil
				}).
				WithAffectedResources(func(_ res.Pattern, _ store.QueryChange) []string {
					return nil // dummy
				}),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.foo").
			Response().
			AssertModel(mock.Model)
	})
}

func TestStoreQueryHandler_GetWithQueryRequestHandler_QueriesQueryStore(t *testing.T) {
	qst := mockstore.NewQueryStore(func(query url.Values) (interface{}, error) {
		restest.AssertEqualJSON(t, "query", query, mock.URLValues)
		return mock.Model, nil
	})
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.Model,
			store.QueryHandler{}.
				WithQueryStore(qst).
				WithQueryRequestHandler(func(resourceName string, pathParams map[string]string, query url.Values) (url.Values, string, error) {
					restest.AssertEqualJSON(t, "resourceName", resourceName, "test.model")
					restest.AssertEqualJSON(t, "pathParams", pathParams, nil)
					restest.AssertEqualJSON(t, "query", query, mock.QueryValues)
					return mock.URLValues, mock.NormalizedQuery, nil
				}),
		)
	}, func(s *restest.Session) {
		s.Get("test.model?" + mock.Query).
			Response().
			AssertModel(mock.Model).
			AssertQuery(mock.NormalizedQuery)
	})
}

func TestStoreQueryHandler_GetWithQueryRequestHandlerWithParams_QueriesQueryStore(t *testing.T) {
	qst := mockstore.NewQueryStore(func(query url.Values) (interface{}, error) {
		restest.AssertEqualJSON(t, "query", query, mock.URLValues)
		return mock.Model, nil
	})
	runTest(t, func(s *res.Service) {
		s.Handle("model.$id",
			res.Model,
			store.QueryHandler{}.
				WithQueryStore(qst).
				WithQueryRequestHandler(func(resourceName string, pathParams map[string]string, query url.Values) (url.Values, string, error) {
					restest.AssertEqualJSON(t, "resourceName", resourceName, "test.model.foo")
					restest.AssertEqualJSON(t, "pathParams", pathParams, map[string]string{"id": "foo"})
					restest.AssertEqualJSON(t, "query", query, mock.QueryValues)
					return mock.URLValues, mock.NormalizedQuery, nil
				}).
				WithAffectedResources(func(_ res.Pattern, _ store.QueryChange) []string {
					return nil // dummy
				}),
		)
	}, func(s *restest.Session) {
		s.Get("test.model.foo?" + mock.Query).
			Response().
			AssertModel(mock.Model).
			AssertQuery(mock.NormalizedQuery)
	})
}

var testQueryChangeTbl = []struct {
	Type     res.ResourceType
	Resource interface{}
	Events   []restest.Event
	Reset    bool
	Error    error
}{
	// Collection
	{res.TypeCollection, mock.Collection, nil, false, nil},
	{res.TypeCollection, mock.Collection, nil, true, nil},
	{res.TypeCollection, mock.Collection, nil, false, mock.CustomError},
	{res.TypeCollection, json.RawMessage(`["foo",null]`), []restest.Event{
		{Name: "remove", Idx: 0},
	}, false, nil},
	{res.TypeCollection, json.RawMessage(`[true,42,"foo",null]`), []restest.Event{
		{Name: "add", Value: true, Idx: 0},
	}, false, nil},
	{res.TypeCollection, json.RawMessage(`[true,"foo",null]`), []restest.Event{
		{Name: "remove", Idx: 0},
		{Name: "add", Value: true, Idx: 0},
	}, false, nil},
	{res.TypeCollection, json.RawMessage(`["foo",null,42]`), []restest.Event{
		{Name: "remove", Idx: 0},
		{Name: "add", Value: 42, Idx: 2},
	}, false, nil},
	// Model
	{res.TypeModel, mock.Model, nil, false, nil},
	{res.TypeModel, mock.Model, nil, true, nil},
	{res.TypeModel, mock.Model, nil, false, mock.CustomError},
	{res.TypeModel, json.RawMessage(`{"id":42,"zoo":"baz"}`), []restest.Event{
		{Name: "change", Changed: map[string]interface{}{"foo": res.DeleteAction, "zoo": "baz"}},
	}, false, nil},
}

func TestStoreQueryHandler_QueryEventWithRequestHandler_ExpectedEvents(t *testing.T) {
	for i, l := range testQueryChangeTbl {
		l := l
		test := fmt.Sprintf("test #%d", i)
		// Create query store
		qst := mockstore.NewQueryStore(func(query url.Values) (interface{}, error) {
			restest.AssertNil(t, query)
			return l.Resource, nil
		})
		// Set resource type
		var typ res.Option
		if l.Type == res.TypeModel {
			typ = res.Model
		} else {
			typ = res.Collection
		}
		runTest(t, func(s *res.Service) {
			s.Handle("resource.foo",
				typ,
				store.QueryHandler{}.
					WithQueryStore(qst).
					WithRequestHandler(func(resourceName string, pathParams map[string]string) (url.Values, error) {
						return nil, nil
					}),
			)
		}, func(s *restest.Session) {
			qst.TriggerQueryChange(mockstore.QueryChange{
				OnEvents: func(q url.Values) ([]store.ResultEvent, bool, error) {
					return restest.ToResultEvents(l.Events), l.Reset, l.Error
				},
			})
			switch {
			case l.Error != nil:
			case l.Reset:
				s.GetMsg().AssertSystemReset([]string{"test.resource.foo"}, nil)
			default:
				for _, ev := range l.Events {
					switch ev.Name {
					case "add":
						s.GetMsg().AssertAddEvent("test.resource.foo", ev.Value, ev.Idx)
					case "remove":
						s.GetMsg().AssertRemoveEvent("test.resource.foo", ev.Idx)
					case "change":
						s.GetMsg().AssertChangeEvent("test.resource.foo", ev.Changed)
					default:
						panic("invalid test data: event name " + ev.Name)
					}
				}
			}
			resp := s.Get("test.resource.foo").Response()
			if l.Type == res.TypeModel {
				resp.AssertModel(l.Resource)
			} else {
				resp.AssertCollection(l.Resource)
			}
		}, restest.WithTest(test))
	}
}

func TestStoreQueryHandler_QueryEventWithQueryRequestHandler_ExpectedEvents(t *testing.T) {
	for i, l := range testQueryChangeTbl {
		l := l
		test := fmt.Sprintf("test #%d", i)
		// Create query store
		qst := mockstore.NewQueryStore(func(query url.Values) (interface{}, error) {
			restest.AssertNil(t, query)
			return l.Resource, nil
		})
		// Set resource type
		var typ res.Option
		if l.Type == res.TypeModel {
			typ = res.Model
		} else {
			typ = res.Collection
		}
		runTest(t, func(s *res.Service) {
			s.Handle("resource.foo",
				typ,
				store.QueryHandler{}.
					WithQueryStore(qst).
					WithQueryRequestHandler(func(resourceName string, pathParams map[string]string, query url.Values) (url.Values, string, error) {
						return nil, mock.NormalizedQuery, nil
					}),
			)
		}, func(s *restest.Session) {
			qst.TriggerQueryChange(mockstore.QueryChange{
				OnEvents: func(q url.Values) ([]store.ResultEvent, bool, error) {
					return restest.ToResultEvents(l.Events), l.Reset, l.Error
				},
			})
			// Assert a query event is sent
			var subj string
			s.GetMsg().AssertQueryEvent("test.resource.foo", &subj)
			// Return with a query request
			resp := s.QueryRequest(subj, mock.Query).Response()
			switch {
			case l.Error != nil:
				resp.AssertError(res.ToError(l.Error))
			case l.Reset:
				if l.Type == res.TypeModel {
					resp.AssertModel(l.Resource)
				} else {
					resp.AssertCollection(l.Resource)
				}
			default:
				resp.AssertEvents(l.Events...)
			}
		}, restest.WithTest(test))
	}
}

func TestStoreQueryHandler_QueryEventWithAffectedResources_ExpectedEvents(t *testing.T) {
	// Create query store
	qst := mockstore.NewQueryStore(func(query url.Values) (interface{}, error) {
		restest.AssertNil(t, query)
		return mock.Collection, nil
	})
	events := []restest.Event{{Name: "remove", Idx: 0}, {Name: "add", Value: 42, Idx: 0}}
	runTest(t, func(s *res.Service) {
		s.Handle("collection.$type",
			res.Collection,
			store.QueryHandler{}.
				WithQueryStore(qst).
				WithQueryRequestHandler(func(resourceName string, pathParams map[string]string, query url.Values) (url.Values, string, error) {
					return nil, mock.NormalizedQuery, nil
				}).
				WithAffectedResources(func(pattern res.Pattern, qc store.QueryChange) []string {
					return []string{string(pattern.ReplaceTag("type", "foo")), string(pattern.ReplaceTag("type", "bar"))}
				}),
		)
	}, func(s *restest.Session) {
		qst.TriggerQueryChange(mockstore.QueryChange{
			OnEvents: func(q url.Values) ([]store.ResultEvent, bool, error) {
				return restest.ToResultEvents(events), false, nil
			},
		})
		// Assert two query events are sent
		var fooSubj string
		var barSubj string
		msgs := s.GetParallelMsgs(2)
		msgs.GetMsg("event.test.collection.foo.query").AssertQueryEvent("test.collection.foo", &fooSubj)
		msgs.GetMsg("event.test.collection.bar.query").AssertQueryEvent("test.collection.bar", &barSubj)
		if fooSubj == barSubj {
			t.Errorf("expected event subjects not to be equal, but they were")
		}
		// Assert we can send query events on both
		s.QueryRequest(fooSubj, mock.Query).Response().AssertEvents(events...)
		s.QueryRequest(barSubj, mock.Query).Response().AssertEvents(events...)
	})
}
