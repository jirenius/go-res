package restest

import (
	"encoding/json"
)

// Request represents a request payload.
type Request struct {
	CID        string              `json:"cid,omitempty"`
	Params     json.RawMessage     `json:"params,omitempty"`
	Token      json.RawMessage     `json:"token,omitempty"`
	Header     map[string][]string `json:"header,omitempty"`
	Host       string              `json:"host,omitempty"`
	RemoteAddr string              `json:"remoteAddr,omitempty"`
	URI        string              `json:"uri,omitempty"`
	Query      string              `json:"query,omitempty"`
}

// DefaultCallRequest returns a default call request.
func DefaultCallRequest() *Request {
	return &Request{CID: "testcid"}
}

// DefaultAuthRequest returns a default auth request.
func DefaultAuthRequest() *Request {
	return &Request{
		CID: "testcid",
		Header: map[string][]string{
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
		Host:       "local",
		RemoteAddr: "127.0.0.1",
		URI:        "/ws",
	}
}
