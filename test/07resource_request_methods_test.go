package test

import (
	"encoding/json"
	"fmt"
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

var resourceRequestTestTbl = []struct {
	Pattern      string
	ResourceName string
	Query        string
}{
	// Simple RID
	{"model", "test.model", ""},
	{"model.foo", "test.model.foo", ""},
	{"model.foo.bar", "test.model.foo.bar", ""},
	// Pattern with placeholders
	{"model.$id", "test.model.42", ""},
	{"model.$id.bar", "test.model.foo.bar", ""},
	{"model.$id.bar.$type", "test.model.foo.bar.baz", ""},
	// Pattern with full wild card
	{"model.>", "test.model.42", ""},
	{"model.>", "test.model.foo.42", ""},
	{"model.$id.>", "test.model.foo.bar", ""},
	{"model.$id.>", "test.model.foo.bar.42", ""},
	{"model.foo.>", "test.model.foo.bar", ""},
	{"model.foo.>", "test.model.foo.bar.42", ""},
	// RID with query
	{"model", "test.model", "foo=bar"},
	{"model.foo", "test.model.foo", "bar.baz=zoo.42"},
	{"model.foo.bar", "test.model.foo.bar", "foo=?bar*.>zoo"},
}

var resourceRequestQueryTestTbl = []struct {
	Query         string
	ExpectedQuery json.RawMessage
}{
	{"foo=bar", json.RawMessage(`{"foo":["bar"]}`)},
	{"foo=bar&baz=42", json.RawMessage(`{"foo":["bar"],"baz":["42"]}`)},
	{"foo=bar&foo=baz", json.RawMessage(`{"foo":["bar","baz"]}`)},
	{"foo[0]=bar&foo[1]=baz", json.RawMessage(`{"foo[0]":["bar"],"foo[1]":["baz"]}`)},
}

// Test Service method returns the service instance
func TestServiceMethod(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			if r.Service() != s {
				t.Errorf("expected resource request Service() to return the service instance, but it didn't")
			}
			r.NotFound()
		}))
	}, func(s *restest.Session) {
		// Test getting the model
		s.Get("test.model").Response()
	})
}

// Test Service method returns the service instance using With
func TestServiceMethodUsingWith(t *testing.T) {
	runTestAsync(t, func(s *res.Service) {
		s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}, func(s *restest.Session, done func()) {
		restest.AssertNoError(t, s.Service().With("test.model", func(r res.Resource) {
			if r.Service() != s.Service() {
				t.Errorf("expected resource Service() to return the service instance, but it didn't")
			}
			done()
		}))
	})
}

// Test Resource and Query method returns the resource name and query.
func TestResourceNameAndQuery(t *testing.T) {
	for _, l := range resourceRequestTestTbl {
		runTest(t, func(s *res.Service) {
			s.Handle(l.Pattern, res.GetModel(func(r res.ModelRequest) {
				rid := l.ResourceName
				if l.Query != "" {
					rid += "?" + l.Query
				}
				rname := r.ResourceName()
				if rname != l.ResourceName {
					t.Errorf("expected ResourceName for RID %#v to be %#v, but got %#v", rid, l.ResourceName, rname)
				}
				q := r.Query()
				if q != l.Query {
					t.Errorf("expected Query for RID %#v to be %#v, but got %#v", rid, l.Query, q)
				}
				r.NotFound()
			}))
		}, func(s *restest.Session) {
			// Test getting the model
			s.Get(l.ResourceName + "?" + l.Query).
				Response().
				AssertError(res.ErrNotFound)
		})
	}
}

// Test Resource and Query method returns the resource name and query when using With
func TestResourceNameAndQueryUsingWith(t *testing.T) {
	for _, l := range resourceRequestTestTbl {
		runTestAsync(t, func(s *res.Service) {
			s.Handle(l.Pattern, res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		}, func(s *restest.Session, done func()) {
			rid := l.ResourceName
			if l.Query != "" {
				rid += "?" + l.Query
			}
			restest.AssertNoError(t, s.Service().With(rid, func(r res.Resource) {
				rname := r.ResourceName()
				if rname != l.ResourceName {
					t.Errorf("expected ResourceName for RID %#v to be %#v, but got %#v", rid, l.ResourceName, rname)
				}
				q := r.Query()
				if q != l.Query {
					t.Errorf("expected Query for RID %#v to be %#v, but got %#v", rid, l.Query, q)
				}
				done()
			}))
		})
	}
}

// Test ParseQuery method parses the query and returns the corresponding values.
func TestParseQuery(t *testing.T) {
	for _, l := range resourceRequestQueryTestTbl {
		runTestAsync(t, func(s *res.Service) {
			s.Handle("model", res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		}, func(s *restest.Session, done func()) {
			rid := "test.model?" + l.Query
			restest.AssertNoError(t, s.Service().With(rid, func(r res.Resource) {
				pq := r.ParseQuery()
				restest.AssertEqualJSON(t, fmt.Sprintf("Query for %#v", rid), pq, l.ExpectedQuery)
				done()
			}))
		})
	}
}
