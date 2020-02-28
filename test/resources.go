package test

import (
	"encoding/json"
	"errors"
	"net/url"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

type modelDto struct {
	ID  int    `json:"id"`
	Foo string `json:"foo"`
}

type mockData struct {
	// Request info
	CID        string
	Host       string
	RemoteAddr string
	URI        string
	Header     map[string][]string
	Params     json.RawMessage
	Token      json.RawMessage
	// Resources
	Model                   *modelDto
	ModelResponse           json.RawMessage
	ModelResult             json.RawMessage
	QueryModelResponse      json.RawMessage
	Collection              []interface{}
	CollectionResponse      json.RawMessage
	CollectionResult        json.RawMessage
	QueryCollectionResponse json.RawMessage
	Result                  json.RawMessage
	ResultResponse          json.RawMessage
	CustomError             *res.Error
	Error                   error
	AccessGrantedResponse   json.RawMessage
	BrokenJSON              []byte
	// Unserializables
	UnserializableValue interface{}
	UnserializableError *res.Error
	// Consts
	ErrorMessage    string
	CustomErrorCode string
	Query           string
	NormalizedQuery string
	QueryValues     url.Values
	URLValues       url.Values
	IntValue        int
}

var mock = mockData{
	// Request info
	"testcid",   // CID
	"local",     // Host
	"127.0.0.1", // RemoteAddr
	"/ws",       // URI
	map[string][]string{ // Header
		"Accept-Encoding":          {"gzip, deflate, br"},
		"Accept-Language":          {"*"},
		"Cache-Control":            {"no-cache"},
		"Connection":               {"Upgrade"},
		"Origin":                   {"http://localhost"},
		"Pragma":                   {"no-cache"},
		"Sec-Websocket-Extensions": {"permessage-deflate; client_max_window_bits"},
		"Sec-Websocket-Key":        {"dGhlIHNhbXBsZSBub25jZQ=="},
		"Sec-Websocket-Version":    {"13"},
		"Upgrade":                  {"websocket"},
		"User-Agent":               {"GolangTest/1.0 (Test)"},
	},
	json.RawMessage(`{"foo":"bar","baz":42}`), // Params
	json.RawMessage(`{"user":"foo","id":42}`), // Token
	// Resources
	&modelDto{ID: 42, Foo: "bar"},                                                                    // Model
	json.RawMessage(`{"result":{"model":{"id":42,"foo":"bar"}}}`),                                    // ModelResponse
	json.RawMessage(`{"model":{"id":42,"foo":"bar"}}`),                                               // ModelResult
	json.RawMessage(`{"result":{"model":{"id":42,"foo":"bar"},"query":"foo=bar&zoo=baz&limit=10"}}`), // QueryModelResponse
	[]interface{}{42, "foo", nil},                                                                    // Collection
	json.RawMessage(`{"result":{"collection":[42,"foo",null]}}`),                                     // CollectionResponse
	json.RawMessage(`{"collection":[42,"foo",null]}`),                                                // CollectionResult
	json.RawMessage(`{"result":{"collection":[42,"foo",null],"query":"foo=bar&zoo=baz&limit=10"}}`),  // QueryCollectionResponse
	json.RawMessage(`{"foo":"bar","zoo":42}`),                                                        // Result
	json.RawMessage(`{"result":{"foo":"bar","zoo":42}}`),                                             // ResultResponse
	&res.Error{Code: "test.custom", Message: "Custom error", Data: map[string]string{"foo": "bar"}},  // CustomError
	errors.New("custom error"),                                                                       // Error
	json.RawMessage(`{"result":{"get":true,"call":"*"}}`),                                            // AccessGrantedResponse
	[]byte(`{]`), // BrokenJSON
	// Unserializables
	func() {}, // UnserializableValue
	&res.Error{Code: "test.unserializable", Message: "Unserializable", Data: func() {}}, // UnserializableError
	// Consts
	"Custom error",                                  // ErrorMessage
	"test.custom",                                   // CustomErrorCode
	"zoo=baz&foo=bar",                               // Query
	"foo=bar&zoo=baz&limit=10",                      // NormalizedQuery
	url.Values{"zoo": {"baz"}, "foo": {"bar"}},      // QueryValues
	url.Values{"id": {"42"}, "foo": {"bar", "baz"}}, // URLValues
	42, // IntValue
}

func (m *mockData) DefaultRequest() *restest.Request {
	return &restest.Request{
		CID: m.CID,
	}
}

func (m *mockData) QueryRequest() *restest.Request {
	return &restest.Request{
		Query: m.Query,
	}
}

func (m *mockData) Request() *restest.Request {
	return &restest.Request{}
}

func (m *mockData) AuthRequest() *restest.Request {
	return &restest.Request{
		CID:        m.CID,
		Header:     m.Header,
		Host:       m.Host,
		RemoteAddr: m.RemoteAddr,
		URI:        m.URI,
	}
}
