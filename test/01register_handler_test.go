package test

import (
	"testing"

	"github.com/jirenius/go-res"
)

// Test that the service can be served without error
func TestRegisterHandler(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model")
	}, func(s *Session) {
		s.AssertSubscription(t, "get.test.>")
		s.AssertSubscription(t, "call.test.>")
		s.AssertSubscription(t, "auth.test.>")
		s.AssertNoSubscription(t, "access.test.>")
	})
}

// Test that the access methods are subscribed to when handler
// with an access handler function is registered
func TestRegisterHandlerWithAccess(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *Session) {
		s.AssertSubscription(t, "get.test.>")
		s.AssertSubscription(t, "call.test.>")
		s.AssertSubscription(t, "auth.test.>")
		s.AssertSubscription(t, "access.test.>")
	})
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
