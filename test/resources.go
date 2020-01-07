package test

import (
	"encoding/json"
	"errors"

	res "github.com/jirenius/go-res"
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
	QueryModelResponse      json.RawMessage
	Collection              []interface{}
	CollectionResponse      json.RawMessage
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
	IntValue        int
}

var mock = mockData{
	// Request info
	"testcid",   // CID
	"local",     // Host
	"127.0.0.1", // RemoteAddr
	"/ws",       // URI
	map[string][]string{ // Header
		"Accept-Encoding":          []string{"gzip, deflate, br"},
		"Accept-Language":          []string{"*"},
		"Cache-Control":            []string{"no-cache"},
		"Connection":               []string{"Upgrade"},
		"Origin":                   []string{"http://localhost"},
		"Pragma":                   []string{"no-cache"},
		"Sec-Websocket-Extensions": []string{"permessage-deflate; client_max_window_bits"},
		"Sec-Websocket-Key":        []string{"dGhlIHNhbXBsZSBub25jZQ=="},
		"Sec-Websocket-Version":    []string{"13"},
		"Upgrade":                  []string{"websocket"},
		"User-Agent":               []string{"GolangTest/1.0 (Test)"},
	},
	json.RawMessage(`{"foo":"bar","baz":42}`), // Params
	json.RawMessage(`{"user":"foo","id":42}`), // Token
	// Resources
	&modelDto{ID: 42, Foo: "bar"},                                                                    // Model
	json.RawMessage(`{"result":{"model":{"id":42,"foo":"bar"}}}`),                                    // ModelResponse
	json.RawMessage(`{"result":{"model":{"id":42,"foo":"bar"},"query":"foo=bar&zoo=baz&limit=10"}}`), // QueryModelResponse
	[]interface{}{42, "foo", nil},                                                                    // Collection
	json.RawMessage(`{"result":{"collection":[42,"foo",null]}}`),                                     // CollectionResponse
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
	"Custom error",             // ErrorMessage
	"test.custom",              // CustomErrorCode
	"zoo=baz&foo=bar",          // Query
	"foo=bar&zoo=baz&limit=10", // NormalizedQuery
	42,                         // IntValue
}

func (m *mockData) DefaultRequest() *request {
	return &request{
		CID: m.CID,
	}
}

func (m *mockData) QueryRequest() *request {
	return &request{
		Query: m.Query,
	}
}

func (m *mockData) Request() *request {
	return &request{}
}

func (m *mockData) AuthRequest() *request {
	return &request{
		CID:        m.CID,
		Header:     m.Header,
		Host:       m.Host,
		RemoteAddr: m.RemoteAddr,
		URI:        m.URI,
	}
}
