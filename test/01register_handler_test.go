package test

import (
	"testing"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test that the service can serve a handler without error
func TestRegisterModelHandler(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) { r.NotFound() }))
	}, func(s *restest.Session) {
		s.AssertSubscription("get.test")
		s.AssertSubscription("get.test.>")
		s.AssertSubscription("call.test.>")
		s.AssertNoSubscription("call.test")
		s.AssertSubscription("auth.test.>")
		s.AssertNoSubscription("auth.test")
		s.AssertNoSubscription("access.test.>")
		s.AssertNoSubscription("access.test")
	}, restest.WithReset([]string{"test", "test.>"}, nil))
}

// Test that the access methods are subscribed to when handler
// with an access handler function is registered
func TestRegisterAccessHandler(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *restest.Session) {
		s.AssertNoSubscription("get.test")
		s.AssertNoSubscription("get.test.>")
		s.AssertNoSubscription("call.test.>")
		s.AssertNoSubscription("call.test")
		s.AssertNoSubscription("auth.test.>")
		s.AssertNoSubscription("auth.test")
		s.AssertSubscription("access.test.>")
		s.AssertSubscription("access.test")
	}, restest.WithReset(nil, []string{"test", "test.>"}))
}

// Test that the resource and access methods are subscribed to when
// both resource and access handler function is registered
func TestRegisterModelAndAccessHandler(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) { r.NotFound() }),
			res.Access(res.AccessGranted),
		)
	}, func(s *restest.Session) {
		s.AssertSubscription("get.test")
		s.AssertSubscription("get.test.>")
		s.AssertSubscription("call.test.>")
		s.AssertNoSubscription("call.test")
		s.AssertSubscription("auth.test.>")
		s.AssertNoSubscription("auth.test")
		s.AssertSubscription("access.test.>")
		s.AssertSubscription("access.test")
	}, restest.WithReset([]string{"test", "test.>"}, []string{"test", "test.>"}))
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

	runTest(t, func(s *res.Service) {
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
	}

	for _, l := range tbl {
		runTest(t, func(s *res.Service) {
			defer func() {
				v := recover()
				if v == nil {
					t.Fatalf("expected a panic, but nothing happened")
				}
			}()

			for _, p := range l {
				s.Handle(p)
			}
		}, nil, restest.WithoutReset)
	}
}

func TestHandler_InvalidHandlerOption_CausesPanic(t *testing.T) {
	tbl := []func(){
		func() { res.Call("foo.bar", func(r res.CallRequest) {}) },
		func() { res.Auth("foo.bar", func(r res.AuthRequest) {}) },
	}

	for _, l := range tbl {
		runTest(t, func(s *res.Service) {
			restest.AssertPanic(t, func() {
				l()
			})
		}, nil, restest.WithoutReset)
	}
}

func TestHandler_InvalidHandlerOptions_CausesPanic(t *testing.T) {
	tbl := [][]res.Option{
		{res.Model, res.Model},
		{res.Collection, res.Collection},
		{res.Model, res.Collection},
		{res.Collection, res.Model},
		{res.ApplyChange(func(r res.Resource, c map[string]interface{}) (map[string]interface{}, error) { return nil, nil }), res.ApplyChange(func(r res.Resource, c map[string]interface{}) (map[string]interface{}, error) { return nil, nil })},
		{res.ApplyAdd(func(r res.Resource, v interface{}, idx int) error { return nil }), res.ApplyAdd(func(r res.Resource, v interface{}, idx int) error { return nil })},
		{res.ApplyRemove(func(r res.Resource, idx int) (interface{}, error) { return nil, nil }), res.ApplyRemove(func(r res.Resource, idx int) (interface{}, error) { return nil, nil })},
		{res.ApplyCreate(func(r res.Resource, v interface{}) error { return nil }), res.ApplyCreate(func(r res.Resource, v interface{}) error { return nil })},
		{res.ApplyDelete(func(r res.Resource) (interface{}, error) { return nil, nil }), res.ApplyDelete(func(r res.Resource) (interface{}, error) { return nil, nil })},
	}

	for _, l := range tbl {
		runTest(t, func(s *res.Service) {
			restest.AssertPanic(t, func() {
				s.Handle("model", l...)
			})
		}, nil, restest.WithoutReset)
	}
}
