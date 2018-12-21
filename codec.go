package res

import "encoding/json"

type resRequest struct {
	CID        string              `json:"cid"`
	Params     json.RawMessage     `json:"params"`
	Token      json.RawMessage     `json:"token"`
	Header     map[string][]string `json:"header"`
	Host       string              `json:"host"`
	RemoteAddr string              `json:"remoteAddr"`
	URI        string              `json:"uri"`
	Query      string              `json:"query"`
}

type successResponse struct {
	Result interface{} `json:"result"`
}

type errorResponse struct {
	Error *Error `json:"error"`
}

type okResponse struct {
	Get  bool   `json:"get,omitempty"`
	Call string `json:"call,omitempty"`
}

type modelResponse struct {
	Model interface{} `json:"model"`
	Query string      `json:"query,omitempty"`
}

type collectionResponse struct {
	Collection interface{} `json:"collection"`
	Query      string      `json:"query,omitempty"`
}

type resetEvent struct {
	Resources []string `json:"resources,omitempty"`
	Access    []string `json:"access,omitempty"`
}
