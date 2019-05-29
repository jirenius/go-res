package test

import (
	"encoding/json"
	"testing"

	res "github.com/jirenius/go-res"
)

// Test Route adds to the path of the the parent
func TestRoute(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Route("foo", func(m *res.Mux) {
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(json.RawMessage(model))
				}),
			)
		})
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.foo.model", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test Mount Mux to service
func TestMount(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		m := res.NewMux("")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
		)
		s.Mount("foo", m)
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.foo.model", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test Mount Mux to service root
func TestMountToRoot(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		m := res.NewMux("foo")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
		)
		s.Mount("", m)
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.foo.model", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test Mount root Mux to serviceyou
func TestMountRootMux(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		m := res.NewMux("")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
		)
		s.Mount("foo", m)
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.foo.model", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test Mount root Mux to service root panics
func TestMountRootMuxToRoot(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		AssertPanic(t, func() {
			m := res.NewMux("")
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(json.RawMessage(model))
				}),
			)
			s.Mount("", m)
		})
	}, nil, withoutReset)
}

// Test Mount Mux twice panics
func TestMountMuxTwice(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		AssertPanic(t, func() {
			m := res.NewMux("")
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(json.RawMessage(model))
				}),
			)
			s.Mount("foo", m)
			s.Mount("bar", m)
		})
	}, nil, withoutReset)
}
