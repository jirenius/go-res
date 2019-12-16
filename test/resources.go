package test

import (
	"encoding/json"
	"errors"

	res "github.com/jirenius/go-res"
)

type ModelDto struct {
	Id  int    `json:"id"`
	Foo string `json:"foo"`
}

type Mock struct {
	// Request info
	CID        string
	Host       string
	RemoteAddr string
	URI        string
	Header     map[string][]string
	Params     json.RawMessage
	Token      json.RawMessage
	// Resources
	Model                   *ModelDto
	ModelResponse           json.RawMessage
	QueryModelResponse      json.RawMessage
	Collection              []interface{}
	CollectionResponse      json.RawMessage
	QueryCollectionResponse json.RawMessage
	Result                  json.RawMessage
	ResultResponse          json.RawMessage
	CustomError             *res.Error
	Error                   error
	// Consts
	ErrorMessage    string
	CustomErrorCode string
	Query           string
	NormalizedQuery string
}

var mock = Mock{
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
	&ModelDto{Id: 42, Foo: "bar"},                                                                    // Model
	json.RawMessage(`{"result":{"model":{"id":42,"foo":"bar"}}}`),                                    // ModelResponse
	json.RawMessage(`{"result":{"model":{"id":42,"foo":"bar"},"query":"foo=bar&zoo=baz&limit=10"}}`), // QueryModelResponse
	[]interface{}{42, "foo", nil},                                                                    // Collection
	json.RawMessage(`{"result":{"collection":[42,"foo",null]}}`),                                     // CollectionResponse
	json.RawMessage(`{"result":{"collection":[42,"foo",null],"query":"foo=bar&zoo=baz&limit=10"}}`),  // QueryCollectionResponse
	json.RawMessage(`{"foo":"bar","zoo":42}`),                                                        // Result
	json.RawMessage(`{"result":{"foo":"bar","zoo":42}}`),                                             // ResultResponse
	&res.Error{Code: "test.custom", Message: "Custom error", Data: map[string]string{"foo": "bar"}},  // CustomError
	errors.New("custom error"),                                                                       // Error
	// Consts
	"Custom error",             // ErrorMessage
	"test.custom",              // CustomErrorCode
	"zoo=baz&foo=bar",          // Query
	"foo=bar&zoo=baz&limit=10", // NormalizedQuery
}

var resource = map[string]string{
	"test.model":                   `{"string":"foo","int":42,"bool":true,"null":null}`,
	"test.model.parent":            `{"name":"parent","child":{"rid":"test.model"}}`,
	"test.model.secondparent":      `{"name":"secondparent","child":{"rid":"test.model"}}`,
	"test.model.grandparent":       `{"name":"grandparent","child":{"rid":"test.model.parent"}}`,
	"test.model.a":                 `{"bref":{"rid":"test.model.b"}}`,
	"test.model.b":                 `{"aref":{"rid":"test.model.a"},"bref":{"rid":"test.model.b"}}`,
	"test.collection":              `["foo",42,true,null]`,
	"test.collection.parent":       `["parent",{"rid":"test.collection"}]`,
	"test.collection.secondparent": `["secondparent",{"rid":"test.collection"}]`,
	"test.collection.grandparent":  `["grandparent",{"rid":"test.collection.parent"},null]`,
	"test.collection.a":            `[{"rid":"test.collection.b"}]`,
	"test.collection.b":            `[{"rid":"test.collection.a"},{"rid":"test.collection.b"}]`,
}

func (m *Mock) DefaultRequest() *request {
	return &request{
		CID: m.CID,
	}
}

func (m *Mock) QueryRequest() *request {
	return &request{
		Query: m.Query,
	}
}

func (m *Mock) Request() *request {
	return &request{}
}

func (m *Mock) AuthRequest() *request {
	return &request{
		CID:        m.CID,
		Header:     m.Header,
		Host:       m.Host,
		RemoteAddr: m.RemoteAddr,
		URI:        m.URI,
	}
}
