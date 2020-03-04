package test

import (
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

var resourceRequestPathParamsTestTbl = []struct {
	Pattern      string
	ResourceName string
	Expected     map[string]string
}{
	{"model", "test.model", nil},
	{"model.$id", "test.model.42", map[string]string{"id": "42"}},
	{"model.$type.$id.foo", "test.model.user.42.foo", map[string]string{"type": "user", "id": "42"}},
	{"model.$id.bar", "test.model.foo.bar", map[string]string{"id": "foo"}},
}

// Test PathParams method returns parameters derived from the resource ID.
func TestPathParams(t *testing.T) {
	for _, l := range resourceRequestPathParamsTestTbl {
		runTest(t, func(s *res.Service) {
			s.Handle(l.Pattern, res.GetModel(func(r res.ModelRequest) {
				pp := r.PathParams()
				restest.AssertEqualJSON(t, "PathParams", pp, l.Expected)
				r.NotFound()
			}))
		}, func(s *restest.Session) {
			s.Get(l.ResourceName).
				Response().
				AssertError(res.ErrNotFound)
		})
	}
}

// Test PathParams method returns parameters derived from the resource ID using With.
func TestPathParamsUsingWith(t *testing.T) {
	for _, l := range resourceRequestPathParamsTestTbl {
		runTestAsync(t, func(s *res.Service) {
			s.Handle(l.Pattern, res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		}, func(s *restest.Session, done func()) {
			restest.AssertNoError(t, s.Service().With(l.ResourceName, func(r res.Resource) {
				pp := r.PathParams()
				restest.AssertEqualJSON(t, "PathParams", pp, l.Expected)
				done()
			}))
		})
	}
}
