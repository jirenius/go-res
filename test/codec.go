package test

import (
	"encoding/json"
)

type request struct {
	CID        string              `json:"cid,omitempty"`
	Params     json.RawMessage     `json:"params,omitempty"`
	Token      json.RawMessage     `json:"token,omitempty"`
	Header     map[string][]string `json:"header,omitempty"`
	Host       string              `json:"host,omitempty"`
	RemoteAddr string              `json:"remoteAddr,omitempty"`
	URI        string              `json:"uri,omitempty"`
	Query      string              `json:"query,omitempty"`
}
