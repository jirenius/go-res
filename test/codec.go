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

// type response struct {
// 	Result interface{} `json:"result"`
// 	Error  *res.Error  `json:"error"`
// }

// type accessResponse struct {
// 	Get  bool   `json:"get,omitempty"`
// 	Call string `json:"call,omitempty"`
// }

// type modelResponse struct {
// 	Model interface{} `json:"model"`
// 	Query string      `json:"query,omitempty"`
// }

// type collectionResponse struct {
// 	Collection interface{} `json:"collection"`
// 	Query      string      `json:"query,omitempty"`
// }

// type resetEvent struct {
// 	Resources []string `json:"resources,omitempty"`
// 	Access    []string `json:"access,omitempty"`
// }
