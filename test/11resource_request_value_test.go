package test

import (
	"testing"
	"time"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

var testRequestValueTbl = []struct {
	Name   string
	Get    func(t *testing.T, r res.GetRequest)
	Assert func(t *testing.T, r res.Resource)
}{
	{
		// ForValue returns true
		"ForValue",
		func(t *testing.T, r res.GetRequest) {
			restest.AssertEqualJSON(t, "r.ForValue()", r.ForValue(), true)
			r.NotFound()
		},
		func(t *testing.T, r res.Resource) { r.Value() },
	},
	{
		// Model returns model
		"Model",
		func(t *testing.T, r res.GetRequest) { r.Model(mock.Model) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertNoError(t, err)
			restest.AssertEqualJSON(t, "r.Value()", v, mock.Model)
		},
	},
	{
		// Model with query returns model
		"ModelWithQuery",
		func(t *testing.T, r res.GetRequest) { r.QueryModel(mock.Model, mock.NormalizedQuery) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertNoError(t, err)
			restest.AssertEqualJSON(t, "r.Value()", v, mock.Model)
		},
	},
	{
		// Collection returns collection
		"Collection",
		func(t *testing.T, r res.GetRequest) { r.Collection(mock.Collection) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertNoError(t, err)
			restest.AssertEqualJSON(t, "r.Value()", v, mock.Collection)
		},
	},
	{
		// Collection with query returns collection
		"CollectionWithQuery",
		func(t *testing.T, r res.GetRequest) { r.QueryCollection(mock.Collection, mock.NormalizedQuery) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertNoError(t, err)
			restest.AssertEqualJSON(t, "r.Value()", v, mock.Collection)
		},
	},
	{
		// Error returns custom error
		"Error",
		func(t *testing.T, r res.GetRequest) { r.Error(mock.CustomError) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertResError(t, err, mock.CustomError)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// NotFound returns system.notFound error
		"NotFound",
		func(t *testing.T, r res.GetRequest) { r.NotFound() },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertResError(t, err, res.ErrNotFound)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// InvalidQuery without message returns system.invalidQuery error
		"InvalidQuery",
		func(t *testing.T, r res.GetRequest) { r.InvalidQuery("") },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertResError(t, err, res.ErrInvalidQuery)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// InvalidQuery with message returns system.invalidQuery error with message
		"InvalidQuery_WithMessage",
		func(t *testing.T, r res.GetRequest) { r.InvalidQuery(mock.ErrorMessage) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertResError(t, err, &res.Error{Code: res.CodeInvalidQuery, Message: mock.ErrorMessage})
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// Panic with *res.Error returns same error
		"Panic_WithResError",
		func(t *testing.T, r res.GetRequest) { panic(mock.CustomError) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertResError(t, err, mock.CustomError)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// Panic with os.Error returns system.internalError
		"Panic_WithError",
		func(t *testing.T, r res.GetRequest) { panic(mock.Error) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertErrorCode(t, err, res.CodeInternalError)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// Panic with string returns system.internalError
		"Panic_WithString",
		func(t *testing.T, r res.GetRequest) { panic(mock.ErrorMessage) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertErrorCode(t, err, res.CodeInternalError)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// Panic with int returns system.internalError
		"Panic_WithInt",
		func(t *testing.T, r res.GetRequest) { panic(42) },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertErrorCode(t, err, res.CodeInternalError)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// No response returns system.internalError
		"NoResponse",
		func(t *testing.T, r res.GetRequest) {},
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertErrorCode(t, err, res.CodeInternalError)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// Value inside GetRequest causes internal error
		"Value",
		func(t *testing.T, r res.GetRequest) { r.Value() },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertErrorCode(t, err, res.CodeInternalError)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// RequireValue inside GetRequest causes panic
		"RequireValue",
		func(t *testing.T, r res.GetRequest) { r.RequireValue() },
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertErrorCode(t, err, res.CodeInternalError)
			restest.AssertEqualJSON(t, "r.Value()", v, nil)
		},
	},
	{
		// Model with Timeout returns model
		"Model_WithTimeout",
		func(t *testing.T, r res.GetRequest) {
			r.Timeout(5 * time.Second)
			r.Model(mock.Model)
		},
		func(t *testing.T, r res.Resource) {
			v, err := r.Value()
			restest.AssertNoError(t, err)
			restest.AssertEqualJSON(t, "r.Value()", v, mock.Model)
		},
	},
}

func TestValue_UsingCall_ReturnsCorrectData(t *testing.T) {
	for _, l := range testRequestValueTbl {
		runTest(t, func(s *res.Service) {
			s.Handle("model",
				res.GetResource(func(r res.GetRequest) {
					l.Get(t, r)
				}),
				res.Call("method", func(r res.CallRequest) {
					l.Assert(t, r)
					r.OK(nil)
				}),
			)
		}, func(s *restest.Session) {
			s.Call("test.model", "method", nil).Response()
		}, restest.WithTest(l.Name))
	}
}

func TestValue_UsingWith_ReturnsCorrectData(t *testing.T) {
	for _, l := range testRequestValueTbl {
		runTestAsync(t, func(s *res.Service) {
			s.Handle("model",
				res.GetResource(func(r res.GetRequest) {
					l.Get(t, r)
				}),
			)
		}, func(s *restest.Session, done func()) {
			s.Service().With("test.model", func(r res.Resource) {
				l.Assert(t, r)
				done()
			})
		}, restest.WithTest(l.Name))
	}
}

var testRequestRequireValueTbl = []struct {
	Name   string
	Get    func(t *testing.T, r res.GetRequest)
	Assert func(t *testing.T, r res.Resource)
}{
	{
		// ForValue returns true
		"ForValue",
		func(t *testing.T, r res.GetRequest) {
			restest.AssertEqualJSON(t, "r.ForValue()", r.ForValue(), true)
			r.Model(mock.Model)
		},
		func(t *testing.T, r res.Resource) { r.RequireValue() },
	},
	{
		// Model returns model
		"Model",
		func(t *testing.T, r res.GetRequest) { r.Model(mock.Model) },
		func(t *testing.T, r res.Resource) {
			v := r.RequireValue()
			restest.AssertEqualJSON(t, "r.RequireValue()", v, mock.Model)
		},
	},
	{
		// Model with query returns model
		"ModelWithQuery",
		func(t *testing.T, r res.GetRequest) { r.QueryModel(mock.Model, mock.NormalizedQuery) },
		func(t *testing.T, r res.Resource) {
			v := r.RequireValue()
			restest.AssertEqualJSON(t, "r.RequireValue()", v, mock.Model)
		},
	},
	{
		// Collection returns collection
		"Collection",
		func(t *testing.T, r res.GetRequest) { r.Collection(mock.Collection) },
		func(t *testing.T, r res.Resource) {
			v := r.RequireValue()
			restest.AssertEqualJSON(t, "r.RequireValue()", v, mock.Collection)
		},
	},
	{
		// Collection with query returns collection
		"CollectionWithQuery",
		func(t *testing.T, r res.GetRequest) { r.QueryCollection(mock.Collection, mock.NormalizedQuery) },
		func(t *testing.T, r res.Resource) {
			v := r.RequireValue()
			restest.AssertEqualJSON(t, "r.RequireValue()", v, mock.Collection)
		},
	},
	{
		// Error returns custom error
		"Error",
		func(t *testing.T, r res.GetRequest) { r.Error(mock.CustomError) },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// NotFound returns system.notFound error
		"NotFound",
		func(t *testing.T, r res.GetRequest) { r.NotFound() },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// InvalidQuery without message returns system.invalidQuery error
		"InvalidQuery",
		func(t *testing.T, r res.GetRequest) { r.InvalidQuery("") },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// InvalidQuery with message returns system.invalidQuery error with message
		"InvalidQuery_WithMessage",
		func(t *testing.T, r res.GetRequest) { r.InvalidQuery(mock.ErrorMessage) },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// Panic with *res.Error returns same error
		"Panic_WithResError",
		func(t *testing.T, r res.GetRequest) { panic(mock.CustomError) },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// Panic with os.Error returns system.internalError
		"Panic_WithError",
		func(t *testing.T, r res.GetRequest) { panic(mock.Error) },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// Panic with string returns system.internalError
		"Panic_WithString",
		func(t *testing.T, r res.GetRequest) { panic(mock.ErrorMessage) },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// Panic with int returns system.internalError
		"Panic_WithInt",
		func(t *testing.T, r res.GetRequest) { panic(42) },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// No response returns system.internalError
		"NoResponse",
		func(t *testing.T, r res.GetRequest) {},
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// Value inside GetRequest causes internal error
		"Value",
		func(t *testing.T, r res.GetRequest) { r.Value() },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// RequireValue inside GetRequest causes panic
		"RequireValue",
		func(t *testing.T, r res.GetRequest) { r.RequireValue() },
		func(t *testing.T, r res.Resource) {
			restest.AssertPanic(t, func() { r.RequireValue() })
		},
	},
	{
		// Model with Timeout returns model
		"Model_WithTimeout",
		func(t *testing.T, r res.GetRequest) {
			r.Timeout(5 * time.Second)
			r.Model(mock.Model)
		},
		func(t *testing.T, r res.Resource) {
			v := r.RequireValue()
			restest.AssertEqualJSON(t, "r.RequireValue()", v, mock.Model)
		},
	},
}

func TestRequireValue_UsingCall_ReturnsCorrectData(t *testing.T) {
	for _, l := range testRequestRequireValueTbl {
		runTest(t, func(s *res.Service) {
			s.Handle("model",
				res.GetResource(func(r res.GetRequest) {
					l.Get(t, r)
				}),
				res.Call("method", func(r res.CallRequest) {
					l.Assert(t, r)
					r.OK(nil)
				}),
			)
		}, func(s *restest.Session) {
			s.Call("test.model", "method", nil).Response()
		}, restest.WithTest(l.Name))
	}
}

func TestRequireValue_UsingWith_ReturnsCorrectData(t *testing.T) {
	for _, l := range testRequestRequireValueTbl {
		runTestAsync(t, func(s *res.Service) {
			s.Handle("model",
				res.GetResource(func(r res.GetRequest) {
					l.Get(t, r)
				}),
			)
		}, func(s *restest.Session, done func()) {
			s.Service().With("test.model", func(r res.Resource) {
				l.Assert(t, r)
				done()
			})
		}, restest.WithTest(l.Name))
	}
}

// Test that Value returns an error on missing get handler.
func TestValueWithoutHandler(t *testing.T) {
	runTestAsync(t, func(s *res.Service) {
		s.Handle("model", res.Access(res.AccessGranted))
	}, func(s *restest.Session, done func()) {
		restest.AssertNoError(t, s.Service().With("test.model", func(r res.Resource) {
			v, err := r.Value()
			restest.AssertEqualJSON(t, "value", v, nil)
			restest.AssertEqualJSON(t, "error", err, res.ErrNotFound)
			done()
		}))
	})
}
