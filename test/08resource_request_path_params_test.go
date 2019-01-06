package test

import (
	"testing"

	res "github.com/jirenius/go-res"
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
		runTest(t, func(s *Session) {
			s.Handle(l.Pattern, res.GetModel(func(r res.ModelRequest) {
				pp := r.PathParams()
				AssertEqual(t, "PathParams", pp, l.Expected)
				r.NotFound()
			}))
		}, func(s *Session) {
			inb := s.Request("get."+l.ResourceName, newDefaultRequest())
			s.GetMsg(t).AssertSubject(t, inb).AssertError(t, res.ErrNotFound)
		})
	}
}

// Test PathParams method returns parameters derived from the resource ID using With.
func TestPathParamsUsingWith(t *testing.T) {
	for _, l := range resourceRequestPathParamsTestTbl {
		runTestAsync(t, func(s *Session) {
			s.Handle(l.Pattern)
		}, func(s *Session, done func()) {
			AssertNoError(t, s.With(l.ResourceName, func(r res.Resource) {
				pp := r.PathParams()
				AssertEqual(t, "PathParams", pp, l.Expected)
				done()
			}))
		})
	}
}
