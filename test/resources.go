package test

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

var (
	defaultCID    = "testcid"
	defaultHeader = map[string][]string{
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
	}
	defaultHost       = "local"
	defaultRemoteAddr = "127.0.0.1"
	defaultURI        = "/ws"
)

func newDefaultRequest() *request {
	return &request{
		CID: defaultCID,
	}
}

func newRequest() *request {
	return &request{}
}

func newAuthRequest() *request {
	return &request{
		CID:        defaultCID,
		Header:     defaultHeader,
		Host:       defaultHost,
		RemoteAddr: defaultRemoteAddr,
		URI:        defaultURI,
	}
}

// Call responses
const (
	requestTimeout uint64 = iota
	noRequest
)
