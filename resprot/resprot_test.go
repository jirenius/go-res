package resprot_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/resprot"
	"github.com/jirenius/go-res/restest"
)

// Test disabled as it might connect to an actual nats instance
func TestUsage_SendRequest(t *testing.T) {
	// conn, _ := nats.Connect("nats://127.0.0.1:4222")
	// response := resprot.SendRequest(conn, "call.example.ping", nil, time.Second)

	// // ------------------

	// if !response.HasError() {
	// 	t.Errorf("expected error, but found none")
	// }
}

func TestUsage_SendModelRequest(t *testing.T) {
	conn := restest.NewMockConn(t, nil)
	go func() {
		msg := conn.GetMsg().
			AssertSubject("get.example.model").
			AssertPayload(json.RawMessage(`{}`))
		conn.RequestRaw(msg.Reply, []byte(`{"result":{"model":{"message":"foo"}}}`))
	}()

	// ---

	response := resprot.SendRequest(conn, "get.example.model", nil, time.Second)

	var model struct {
		Message string `json:"message"`
	}
	_, err := response.ParseModel(&model)

	// ------------------

	restest.AssertNoError(t, err)
	restest.AssertEqualJSON(t, "model", model, json.RawMessage(`{"message":"foo"}`))
}

func TestUsage_SendCallRequest(t *testing.T) {
	conn := restest.NewMockConn(t, nil)
	go func() {
		msg := conn.GetMsg().
			AssertSubject("call.math.add").
			AssertPayload(json.RawMessage(`{"params":{"a":5,"b":6}}`))
		conn.RequestRaw(msg.Reply, []byte(`{"result":{"sum":11}}`))
	}()

	// ---

	response := resprot.SendRequest(conn, "call.math.add", resprot.Request{Params: struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}{5, 6}}, time.Second)

	var result struct {
		Sum float64 `json:"sum"`
	}
	err := response.ParseResult(&result)

	// ---

	restest.AssertNoError(t, err)
	restest.AssertEqualJSON(t, "result", result, json.RawMessage(`{"sum":11}`))
}

func TestSendRequest_DifferentPayload_SendsExpectedPayload(t *testing.T) {
	table := []struct {
		Payload  interface{}
		Expected interface{}
	}{
		{nil, json.RawMessage(`{}`)},
		{json.RawMessage(`{}`), json.RawMessage(`{}`)},
		{json.RawMessage(`{"foo":"bar"}`), json.RawMessage(`{"foo":"bar"}`)},
		{json.RawMessage(`"foo"`), json.RawMessage(`"foo"`)},
	}

	for _, l := range table {
		l := l
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg().
				AssertSubject("call.test.method").
				AssertPayload(l.Expected)
			conn.RequestRaw(msg.Reply, []byte(`{"result":null}`))
		}()

		resprot.SendRequest(conn, "call.test.method", l.Payload, time.Second)
	}
}

func TestSendRequest_WithBrokenJSON_ReturnsError(t *testing.T) {
	conn := restest.NewMockConn(t, nil)
	response := resprot.SendRequest(conn, "call.test.method", json.RawMessage(`[broken}`), time.Second)
	restest.AssertError(t, response.Error)
}

func TestSendRequest_WithFailedSubscribe_ReturnsError(t *testing.T) {
	conn := restest.NewMockConn(t, nil)
	conn.FailNextSubscription()
	response := resprot.SendRequest(conn, "call.test.method", nil, time.Second)
	restest.AssertError(t, response.Error)
}

func TestSendRequest_WithoutResponse_ReturnsErrTimeout(t *testing.T) {
	conn := restest.NewMockConn(t, nil)
	response := resprot.SendRequest(conn, "call.test.method", nil, time.Millisecond)
	restest.AssertResError(t, response.Error, res.ErrTimeout)
}

func TestSendRequest_WithInvalidOrErrorResponse_ReturnsError(t *testing.T) {
	table := []struct {
		ResponsePayload   []byte
		ExpectedErrorCode string
	}{
		// Invalid response
		{nil, "system.internalError"},
		{[]byte(``), "system.internalError"},
		{[]byte(`[broken}`), "system.internalError"},
		{[]byte(`{}`), "system.internalError"},
		{[]byte(`{"foo":"bar"}`), "system.internalError"},
		{[]byte(`{"error":"foo"}`), "system.internalError"},
		{[]byte(`{"resource":"foo"}`), "system.internalError"},
		// Error response
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"}}`), "custom.error"},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"}}`), "custom.error"},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"result":"foo"}`), "custom.error"},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"},"result":"foo"}`), "custom.error"},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "call.test.method", nil, time.Second)

		restest.AssertErrorCode(t, response.Error, l.ExpectedErrorCode, ctx)
		restest.AssertTrue(t, "HasError() to return true", response.HasError(), ctx)
		restest.AssertTrue(t, "HasResource() to return false", !response.HasResource(), ctx)
		restest.AssertTrue(t, "HasResult() to return false", !response.HasResult(), ctx)
	}
}

func TestSendRequest_WithResourceResponse_ReturnsResource(t *testing.T) {
	table := []struct {
		ResponsePayload []byte
		Expected        res.Ref
	}{
		// Resource response
		{[]byte(`{"resource":{"rid":"test.model"}}`), "test.model"},
		{[]byte(`{"resource":{"rid":"test.model"},"result":"foo"}`), "test.model"},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "call.test.method", nil, time.Second)

		restest.AssertTrue(t, "HasError() to return false", !response.HasError(), ctx)
		restest.AssertTrue(t, "HasResource() to return true", response.HasResource(), ctx)
		restest.AssertTrue(t, "HasResult() to return false", !response.HasResult(), ctx)
		restest.AssertEqualJSON(t, "response.Resource", response.Resource, l.Expected, ctx)
	}
}

func TestParseResult_WithResultResponse_ReturnsResult(t *testing.T) {
	table := []struct {
		ResponsePayload []byte
		Expected        interface{}
	}{
		// Invalid response
		{[]byte(`{"result":null}`), nil},
		{[]byte(`{"result":"foo"}`), "foo"},
		{[]byte(`{"result":["foo"]}`), json.RawMessage(`["foo"]`)},
		{[]byte(`{"result":{"foo":42}}`), json.RawMessage(`{"foo":42}`)},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "call.test.method", nil, time.Second)

		restest.AssertTrue(t, "HasError() to return false", !response.HasError(), ctx)
		restest.AssertTrue(t, "HasResource() to return false", !response.HasResource(), ctx)
		restest.AssertTrue(t, "HasResult() to return true", response.HasResult(), ctx)
		var result interface{}
		restest.AssertNoError(t, response.ParseResult(&result))
		restest.AssertEqualJSON(t, "parsed result", result, l.Expected, ctx)
	}
}

func TestParseResult_WithInvalidResultResponse_ReturnsError(t *testing.T) {
	table := []struct {
		ResponsePayload []byte
	}{
		// Invalid response
		{nil},
		{[]byte(``)},
		{[]byte(`[broken}`)},
		{[]byte(`{}`)},
		{[]byte(`{"foo":"bar"}`)},
		{[]byte(`{"error":"foo"}`)},
		{[]byte(`{"resource":"foo"}`)},
		// Error response
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"result":"foo"}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"},"result":"foo"}`)},
		// Resource response
		{[]byte(`{"resource":{"rid":"test.model"}}`)},
		{[]byte(`{"resource":{"rid":"test.model"},"result":"foo"}`)},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "call.test.method", nil, time.Second)
		var result interface{}
		restest.AssertError(t, response.ParseResult(&result), ctx)
		restest.AssertTrue(t, "result is nil", result == nil)
	}
}

func TestParseModel_WithModelResponse_ReturnsModelAndQuery(t *testing.T) {
	table := []struct {
		ResponsePayload []byte
		ExpectedModel   interface{}
		ExpectedQuery   string
	}{
		// Model response
		{[]byte(`{"result":{"model":{}}}`), json.RawMessage(`{}`), ""},
		{[]byte(`{"result":{"model":{"foo":"bar"}}}`), json.RawMessage(`{"foo":"bar"}`), ""},
		{[]byte(`{"result":{"model":{"foo":"bar"},"query":"q=test&limit=5"}}`), json.RawMessage(`{"foo":"bar"}`), "q=test&limit=5"},
		{[]byte(`{"result":{"model":{"ref":{"rid":"test.model"}}}}`), map[string]interface{}{"ref": res.Ref("test.model")}, ""},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "get.test.model", nil, time.Second)

		restest.AssertTrue(t, "HasError() to return false", !response.HasError(), ctx)
		restest.AssertTrue(t, "HasResource() to return false", !response.HasResource(), ctx)
		restest.AssertTrue(t, "HasResult() to return true", response.HasResult(), ctx)
		var model interface{}
		query, err := response.ParseModel(&model)
		restest.AssertNoError(t, err)
		restest.AssertEqualJSON(t, "model", model, l.ExpectedModel, ctx)
		restest.AssertEqualJSON(t, "query", query, l.ExpectedQuery, ctx)
	}
}

func TestParseModel_WithInvalidModelResponse_ReturnsError(t *testing.T) {
	table := []struct {
		ResponsePayload []byte
	}{
		// Invalid response
		{nil},
		{[]byte(``)},
		{[]byte(`[broken}`)},
		{[]byte(`{}`)},
		{[]byte(`{"foo":"bar"}`)},
		{[]byte(`{"error":"foo"}`)},
		{[]byte(`{"resource":"foo"}`)},
		// Error response
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"result":{"model":{"foo":"bar"}}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"},"result":{"model":{"foo":"bar"}}}`)},
		// Resource response
		{[]byte(`{"resource":{"rid":"test.model"}}`)},
		{[]byte(`{"resource":{"rid":"test.model"},"result":{"model":{"foo":"bar"}}}`)},
		// Result response
		{[]byte(`{"result":null}`)},
		{[]byte(`{"result":"foo"}`)},
		{[]byte(`{"result":{"foo":"bar"}}`)},
		// Collection response
		{[]byte(`{"result":{"collection":[]}}`)},
		{[]byte(`{"result":{"collection":["foo","bar"]}}`)},
		{[]byte(`{"result":{"collection":["foo","bar"],"query":"q=test&limit=5"}}`)},
		{[]byte(`{"result":{"collection":[{"rid":"test.model"}]}}`)},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "get.test.model", nil, time.Second)
		var model interface{}
		query, err := response.ParseModel(&model)
		restest.AssertError(t, err, ctx)
		restest.AssertTrue(t, "model is nil", model == nil)
		restest.AssertTrue(t, "query is empty", query == "")
	}
}

func TestParseCollection_WithCollectionResponse_ReturnsCollectionAndQuery(t *testing.T) {
	table := []struct {
		ResponsePayload    []byte
		ExpectedCollection interface{}
		ExpectedQuery      string
	}{
		// Collection response
		{[]byte(`{"result":{"collection":[]}}`), json.RawMessage(`[]`), ""},
		{[]byte(`{"result":{"collection":["foo","bar"]}}`), json.RawMessage(`["foo","bar"]`), ""},
		{[]byte(`{"result":{"collection":["foo","bar"],"query":"q=test&limit=5"}}`), json.RawMessage(`["foo","bar"]`), "q=test&limit=5"},
		{[]byte(`{"result":{"collection":[{"rid":"test.model"}]}}`), []res.Ref{res.Ref("test.model")}, ""},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "get.test.collection", nil, time.Second)

		restest.AssertTrue(t, "HasError() to return false", !response.HasError(), ctx)
		restest.AssertTrue(t, "HasResource() to return false", !response.HasResource(), ctx)
		restest.AssertTrue(t, "HasResult() to return true", response.HasResult(), ctx)
		var collection interface{}
		query, err := response.ParseCollection(&collection)
		restest.AssertNoError(t, err)
		restest.AssertEqualJSON(t, "collection", collection, l.ExpectedCollection, ctx)
		restest.AssertEqualJSON(t, "query", query, l.ExpectedQuery, ctx)
	}
}

func TestParseCollection_WithInvalidCollectionResponse_ReturnsError(t *testing.T) {
	table := []struct {
		ResponsePayload []byte
	}{
		// Invalid response
		{nil},
		{[]byte(``)},
		{[]byte(`[broken}`)},
		{[]byte(`{}`)},
		{[]byte(`{"foo":"bar"}`)},
		{[]byte(`{"error":"foo"}`)},
		{[]byte(`{"resource":"foo"}`)},
		// Error response
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"result":{"model":{"foo":"bar"}}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"},"result":{"model":{"foo":"bar"}}}`)},
		// Resource response
		{[]byte(`{"resource":{"rid":"test.model"}}`)},
		{[]byte(`{"resource":{"rid":"test.model"},"result":{"model":{"foo":"bar"}}}`)},
		// Result response
		{[]byte(`{"result":null}`)},
		{[]byte(`{"result":"foo"}`)},
		{[]byte(`{"result":{"foo":"bar"}}`)},
		// Model response
		{[]byte(`{"result":{"model":{}}}`)},
		{[]byte(`{"result":{"model":{"foo":"bar"}}}`)},
		{[]byte(`{"result":{"model":{"foo":"bar"},"query":"q=test&limit=5"}}`)},
		{[]byte(`{"result":{"model":{"ref":{"rid":"test.model"}}}}`)},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "get.test.collection", nil, time.Second)
		var collection interface{}
		query, err := response.ParseCollection(&collection)
		restest.AssertError(t, err, ctx)
		restest.AssertTrue(t, "collection is nil", collection == nil)
		restest.AssertTrue(t, "query is empty", query == "")
	}
}

func TestAccessResult_WithAccessResponse_ReturnsAccessValues(t *testing.T) {
	table := []struct {
		ResponsePayload []byte
		ExpectedGet     bool
		ExpectedCall    string
	}{
		// Access response
		{[]byte(`{"result":null}`), false, ""},
		{[]byte(`{"result":{}}`), false, ""},
		{[]byte(`{"result":{"get":false,"call":""}}`), false, ""},
		{[]byte(`{"result":{"get":true}}`), true, ""},
		{[]byte(`{"result":{"call":"foo,bar"}}`), false, "foo,bar"},
		{[]byte(`{"result":{"get":true,"call":"*"}}`), true, "*"},
		{[]byte(`{"result":{"get":true,"call":"*","foo":"bar"}}`), true, "*"},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "access.test.model", nil, time.Second)

		restest.AssertTrue(t, "HasError() to return false", !response.HasError(), ctx)
		restest.AssertTrue(t, "HasResource() to return false", !response.HasResource(), ctx)
		restest.AssertTrue(t, "HasResult() to return true", response.HasResult(), ctx)
		get, call, err := response.AccessResult()
		restest.AssertNoError(t, err)
		restest.AssertEqualJSON(t, "get", get, l.ExpectedGet, ctx)
		restest.AssertEqualJSON(t, "call", call, l.ExpectedCall, ctx)
	}
}

func TestAccessResult_WithInvalidAccessResponse_ReturnsError(t *testing.T) {
	table := []struct {
		ResponsePayload []byte
	}{
		// Invalid response
		{nil},
		{[]byte(``)},
		{[]byte(`[broken}`)},
		{[]byte(`{}`)},
		{[]byte(`{"foo":"bar"}`)},
		{[]byte(`{"error":"foo"}`)},
		{[]byte(`{"resource":"foo"}`)},
		// Error response
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"result":{"model":{"foo":"bar"}}}`)},
		{[]byte(`{"error":{"code":"custom.error","message":"Custom error"},"resource":{"rid":"test.model"},"result":{"model":{"foo":"bar"}}}`)},
		// Resource response
		{[]byte(`{"resource":{"rid":"test.model"}}`)},
		{[]byte(`{"resource":{"rid":"test.model"},"result":{"model":{"foo":"bar"}}}`)},
		// Result response
		{[]byte(`{"result":["foo"]}`)},
		{[]byte(`{"result":{"get":"yes"}}`)},
		{[]byte(`{"result":{"call":true}}`)},
	}

	for i, l := range table {
		l := l
		ctx := fmt.Sprintf("test #%d", i+1)
		conn := restest.NewMockConn(t, nil)
		go func() {
			msg := conn.GetMsg()
			conn.RequestRaw(msg.Reply, l.ResponsePayload)
		}()

		response := resprot.SendRequest(conn, "access.test.model", nil, time.Second)
		get, call, err := response.AccessResult()
		restest.AssertError(t, err, ctx)
		restest.AssertTrue(t, "get is false", !get)
		restest.AssertTrue(t, "call is empty", call == "")
	}
}
