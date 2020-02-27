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
		Host:       "local",
		RemoteAddr: "127.0.0.1",
		URI:        "/ws",
	}
}
