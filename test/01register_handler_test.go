package test

import (
	"testing"

	"github.com/jirenius/go-res"
)

// Test that the service can serve a handler without error
func TestRegisterModelHandler(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) { r.NotFound() }))
	}, func(s *Session) {
		s.AssertSubscription(t, "get.test.>")
		s.AssertSubscription(t, "call.test.>")
		s.AssertSubscription(t, "auth.test.>")
		s.AssertNoSubscription(t, "access.test.>")
	}, withResources([]string{"test.>"}))
}

// Test that the access methods are subscribed to when handler
// with an access handler function is registered
func TestRegisterAccessHandler(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *Session) {
		s.AssertNoSubscription(t, "get.test.>")
		s.AssertNoSubscription(t, "call.test.>")
		s.AssertNoSubscription(t, "auth.test.>")
		s.AssertSubscription(t, "access.test.>")
	}, withAccess([]string{"test.>"}))
}

// Test that the resource and access methods are subscribed to when
// both resource and access handler function is registered
func TestRegisterModelAndAccessHandler(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) { r.NotFound() }),
			res.Access(res.AccessGranted),
		)
	}, func(s *Session) {
		s.AssertSubscription(t, "get.test.>")
		s.AssertSubscription(t, "call.test.>")
		s.AssertSubscription(t, "auth.test.>")
		s.AssertSubscription(t, "access.test.>")
	}, withResources([]string{"test.>"}), withAccess([]string{"test.>"}))
}

// Test that registering both a model and collection handler results
// in a panic
func TestPanicOnMultipleGetHandlers(t *testing.T) {
	defer func() {
		v := recover()
		if v == nil {
			t.Fatalf("expected a panic, but nothing happened")
		}
	}()

	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.NotFound()
			}),
			res.GetCollection(func(r res.CollectionRequest) {
				r.NotFound()
			}),
		)
	}, nil)
}

// Test that making invalid pattern registration causes panic
func TestPanicOnInvalidPatternRegistration(t *testing.T) {

	tbl := [][]string{
		{"model.$id.type.$id"},
		{"model.foo", "model.foo"},
		{"model..foo"},
		{"model.$"},
		{"model.$.foo"},
		{"model.>.foo"},
		{"model.foo.>bar"},
		{"model.>.foo"},
	}

	for _, l := range tbl {
		runTest(t, func(s *Session) {
			defer func() {
				v := recover()
				if v == nil {
					t.Fatalf("expected a panic, but nothing happened")
				}
			}()

			for _, p := range l {
				s.Handle(p)
			}
		}, nil, withoutReset)
	}
}
