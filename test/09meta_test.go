package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

var metaTestTbl = []struct {
	Name         string
	Status       int
	Header       map[string][]string
	ExpectedMeta interface{}
}{
	{
		Name:         "status code",
		Status:       402,
		ExpectedMeta: json.RawMessage(`{"status":402}`),
	},
	{
		Name:         "single header",
		Header:       map[string][]string{"Location": {"https://example.com"}},
		ExpectedMeta: json.RawMessage(`{"header":{"Location":["https://example.com"]}}`),
	},
	{
		Name:         "multiple headers",
		Header:       map[string][]string{"Set-Cookie": {"foo=bar", "zoo=baz"}},
		ExpectedMeta: json.RawMessage(`{"header":{"Set-Cookie":["foo=bar","zoo=baz"]}}`),
	},
	{
		Name:         "status and header",
		Status:       303,
		Header:       map[string][]string{"Location": {"https://example.com"}},
		ExpectedMeta: json.RawMessage(`{"status":303,"header":{"Location":["https://example.com"]}}`),
	},
}

// Test auth request with meta data and a successful response.
func TestMeta_AuthRequestWithSuccessResponse_MetaInResponse(t *testing.T) {
	for _, k := range []struct {
		Name   string
		Result interface{}
	}{
		{Name: "nil result", Result: nil},
		{Name: "custom result", Result: mock.Result},
	} {
		for _, l := range metaTestTbl {
			runTest(t, func(s *res.Service) {
				s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
					r.SetResponseStatus(l.Status)
					for k, v := range l.Header {
						r.ResponseHeader()[k] = v
					}
					r.OK(k.Result)
				}))
			}, func(s *restest.Session) {
				req := mock.DefaultRequest()
				req.IsHTTP = true
				s.Auth("test.model", "method", req).
					Response().
					AssertPayload(map[string]interface{}{
						"result": k.Result,
						"meta":   l.ExpectedMeta,
					})
			}, restest.WithTest(fmt.Sprintf("%s with %s", k.Name, l.Name)))
		}
	}
}

// Test auth request with meta data and an error response.
func TestMeta_AuthRequestWithErrorResponse_MetaInResponse(t *testing.T) {
	for _, k := range []struct {
		Name          string
		Respond       func(r res.AuthRequest)
		ExpectedError error
	}{
		{Name: "CustomError()", Respond: func(r res.AuthRequest) { r.Error(mock.CustomError) }, ExpectedError: mock.CustomError},
		{Name: "NotFound()", Respond: func(r res.AuthRequest) { r.NotFound() }, ExpectedError: res.ErrNotFound},
		{Name: "MethodNotFound()", Respond: func(r res.AuthRequest) { r.MethodNotFound() }, ExpectedError: res.ErrMethodNotFound},
		{Name: `InvalidParams("")`, Respond: func(r res.AuthRequest) { r.InvalidParams("") }, ExpectedError: res.ErrInvalidParams},
		{Name: `InvalidParams("foo")`, Respond: func(r res.AuthRequest) { r.InvalidParams("foo") }, ExpectedError: &res.Error{Code: res.CodeInvalidParams, Message: "foo"}},
		{Name: `InvalidQuery("")`, Respond: func(r res.AuthRequest) { r.InvalidQuery("") }, ExpectedError: res.ErrInvalidQuery},
		{Name: `InvalidQuery("foo")`, Respond: func(r res.AuthRequest) { r.InvalidQuery("foo") }, ExpectedError: &res.Error{Code: res.CodeInvalidQuery, Message: "foo"}},
	} {
		for _, l := range metaTestTbl {
			runTest(t, func(s *res.Service) {
				s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
					r.SetResponseStatus(l.Status)
					for k, v := range l.Header {
						r.ResponseHeader()[k] = v
					}
					k.Respond(r)
				}))
			}, func(s *restest.Session) {
				req := mock.DefaultRequest()
				req.IsHTTP = true
				s.Auth("test.model", "method", req).
					Response().
					AssertPayload(map[string]interface{}{
						"error": k.ExpectedError,
						"meta":  l.ExpectedMeta,
					})
			}, restest.WithTest(fmt.Sprintf("%s with %s", k.Name, l.Name)))
		}
	}
}

// Test auth request with meta data and a resource response.
func TestMeta_AuthRequestWithResourceResponse_MetaInResponse(t *testing.T) {
	rid := "test.foo"
	for _, l := range metaTestTbl {
		runTest(t, func(s *res.Service) {
			s.Handle("model", res.Auth("method", func(r res.AuthRequest) {
				r.SetResponseStatus(l.Status)
				for k, v := range l.Header {
					r.ResponseHeader()[k] = v
				}
				r.Resource(rid)
			}))
		}, func(s *restest.Session) {
			req := mock.DefaultRequest()
			req.IsHTTP = true
			s.Auth("test.model", "method", req).
				Response().
				AssertPayload(map[string]interface{}{
					"resource": res.Ref(rid),
					"meta":     l.ExpectedMeta,
				})
		}, restest.WithTest(l.Name))
	}
}

// Test call request with meta data and a successful response.
func TestMeta_CallRequestWithSuccessResponse_MetaInResponse(t *testing.T) {
	for _, k := range []struct {
		Name   string
		Result interface{}
	}{
		{Name: "nil result", Result: nil},
		{Name: "custom result", Result: mock.Result},
	} {
		for _, l := range metaTestTbl {
			runTest(t, func(s *res.Service) {
				s.Handle("model", res.Call("method", func(r res.CallRequest) {
					r.SetResponseStatus(l.Status)
					for k, v := range l.Header {
						r.ResponseHeader()[k] = v
					}
					r.OK(k.Result)
				}))
			}, func(s *restest.Session) {
				req := mock.DefaultRequest()
				req.IsHTTP = true
				s.Call("test.model", "method", req).
					Response().
					AssertPayload(map[string]interface{}{
						"result": k.Result,
						"meta":   l.ExpectedMeta,
					})
			}, restest.WithTest(fmt.Sprintf("%s with %s", k.Name, l.Name)))
		}
	}
}

// Test call request with meta data and an error response.
func TestMeta_CallRequestWithErrorResponse_MetaInResponse(t *testing.T) {
	for _, k := range []struct {
		Name          string
		Respond       func(r res.CallRequest)
		ExpectedError error
	}{
		{Name: "CustomError()", Respond: func(r res.CallRequest) { r.Error(mock.CustomError) }, ExpectedError: mock.CustomError},
		{Name: "NotFound()", Respond: func(r res.CallRequest) { r.NotFound() }, ExpectedError: res.ErrNotFound},
		{Name: "MethodNotFound()", Respond: func(r res.CallRequest) { r.MethodNotFound() }, ExpectedError: res.ErrMethodNotFound},
		{Name: `InvalidParams("")`, Respond: func(r res.CallRequest) { r.InvalidParams("") }, ExpectedError: res.ErrInvalidParams},
		{Name: `InvalidParams("foo")`, Respond: func(r res.CallRequest) { r.InvalidParams("foo") }, ExpectedError: &res.Error{Code: res.CodeInvalidParams, Message: "foo"}},
		{Name: `InvalidQuery("")`, Respond: func(r res.CallRequest) { r.InvalidQuery("") }, ExpectedError: res.ErrInvalidQuery},
		{Name: `InvalidQuery("foo")`, Respond: func(r res.CallRequest) { r.InvalidQuery("foo") }, ExpectedError: &res.Error{Code: res.CodeInvalidQuery, Message: "foo"}},
	} {
		for _, l := range metaTestTbl {
			runTest(t, func(s *res.Service) {
				s.Handle("model", res.Call("method", func(r res.CallRequest) {
					r.SetResponseStatus(l.Status)
					for k, v := range l.Header {
						r.ResponseHeader()[k] = v
					}
					r.Error(mock.CustomError)
				}))
			}, func(s *restest.Session) {
				req := mock.DefaultRequest()
				req.IsHTTP = true
				s.Call("test.model", "method", req).
					Response().
					AssertPayload(map[string]interface{}{
						"error": mock.CustomError,
						"meta":  l.ExpectedMeta,
					})
			}, restest.WithTest(fmt.Sprintf("%s with %s", k.Name, l.Name)))
		}
	}
}

// Test call request with meta data and a resource response.
func TestMeta_CallRequestWithResourceResponse_MetaInResponse(t *testing.T) {
	rid := "test.foo"
	for _, l := range metaTestTbl {
		runTest(t, func(s *res.Service) {
			s.Handle("model", res.Call("method", func(r res.CallRequest) {
				r.SetResponseStatus(l.Status)
				for k, v := range l.Header {
					r.ResponseHeader()[k] = v
				}
				r.Resource(rid)
			}))
		}, func(s *restest.Session) {
			req := mock.DefaultRequest()
			req.IsHTTP = true
			s.Call("test.model", "method", req).
				Response().
				AssertPayload(map[string]interface{}{
					"resource": res.Ref(rid),
					"meta":     l.ExpectedMeta,
				})
		}, restest.WithTest(l.Name))
	}
}

// Test access request with meta data and a successful response.
func TestMeta_AccessRequestWithSuccessResponse_MetaInResponse(t *testing.T) {
	for _, k := range []struct {
		Name           string
		Respond        func(r res.AccessRequest)
		ExpectedResult interface{}
	}{
		{Name: "AccessGranted()", Respond: func(r res.AccessRequest) { r.AccessGranted() }, ExpectedResult: json.RawMessage(`{"get":true,"call":"*"}`)},
		{Name: `Access(true, "foo")`, Respond: func(r res.AccessRequest) { r.Access(true, "foo") }, ExpectedResult: json.RawMessage(`{"get":true,"call":"foo"}`)},
	} {
		for _, l := range metaTestTbl {
			runTest(t, func(s *res.Service) {
				s.Handle("model", res.Access(func(r res.AccessRequest) {
					r.SetResponseStatus(l.Status)
					for k, v := range l.Header {
						r.ResponseHeader()[k] = v
					}
					k.Respond(r)
				}))
			}, func(s *restest.Session) {
				req := mock.DefaultRequest()
				req.IsHTTP = true
				s.Access("test.model", req).
					Response().
					AssertPayload(map[string]interface{}{
						"result": k.ExpectedResult,
						"meta":   l.ExpectedMeta,
					})
			}, restest.WithTest(fmt.Sprintf("%s with %s", k.Name, l.Name)))
		}
	}
}

// Test access request with meta data and an error response.
func TestMeta_AccessRequestWithErrorResponse_MetaInResponse(t *testing.T) {
	for _, k := range []struct {
		Name          string
		Respond       func(r res.AccessRequest)
		ExpectedError error
	}{
		{Name: "AccessDenied()", Respond: func(r res.AccessRequest) { r.AccessDenied() }, ExpectedError: res.ErrAccessDenied},
		{Name: `Access(false, "")`, Respond: func(r res.AccessRequest) { r.Access(false, "") }, ExpectedError: res.ErrAccessDenied},
		{Name: "CustomError()", Respond: func(r res.AccessRequest) { r.Error(mock.CustomError) }, ExpectedError: mock.CustomError},
		{Name: "NotFound()", Respond: func(r res.AccessRequest) { r.NotFound() }, ExpectedError: res.ErrNotFound},
		{Name: `InvalidQuery("")`, Respond: func(r res.AccessRequest) { r.InvalidQuery("") }, ExpectedError: res.ErrInvalidQuery},
		{Name: `InvalidQuery("foo")`, Respond: func(r res.AccessRequest) { r.InvalidQuery("foo") }, ExpectedError: &res.Error{Code: res.CodeInvalidQuery, Message: "foo"}},
	} {
		for _, l := range metaTestTbl {
			runTest(t, func(s *res.Service) {
				s.Handle("model", res.Access(func(r res.AccessRequest) {
					r.SetResponseStatus(l.Status)
					for k, v := range l.Header {
						r.ResponseHeader()[k] = v
					}
					k.Respond(r)
				}))
			}, func(s *restest.Session) {
				req := mock.DefaultRequest()
				req.IsHTTP = true
				s.Access("test.model", req).
					Response().
					AssertPayload(map[string]interface{}{
						"error": k.ExpectedError,
						"meta":  l.ExpectedMeta,
					})
			}, restest.WithTest(fmt.Sprintf("%s with %s", k.Name, l.Name)))
		}
	}
}

// Test using SetResponseStatus on call request, when IsHTTP is false, causes panic.
func TestMeta_SetResponseStatusWhenIsHTTPIsFalse_Panics(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.SetResponseStatus(402)
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertErrorCode(res.CodeInternalError)
	})
}

// Test using ResponseHeader on call request, when IsHTTP is false, causes panic.
func TestMeta_ResponseHeaderWhenIsHTTPIsFalse_Panics(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			_ = r.ResponseHeader()
			r.OK(nil)
		}))
	}, func(s *restest.Session) {
		s.Call("test.model", "method", nil).
			Response().
			AssertErrorCode(res.CodeInternalError)
	})
}
