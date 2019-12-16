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

type resourceResponse struct {
	Resource Ref `json:"resource"`
}

type errorResponse struct {
	Error *Error `json:"error"`
}

type accessResponse struct {
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

type tokenEvent struct {
	Token interface{} `json:"token"`
}

type changeEvent struct {
	Values map[string]interface{} `json:"values"`
}

type addEvent struct {
	Value interface{} `json:"value"`
	Idx   int         `json:"idx"`
}

type removeEvent struct {
	Idx int `json:"idx"`
}

type resQueryEvent struct {
	Subject string `json:"subject"`
}

type resQueryRequest struct {
	Query string `json:"query"`
}

type resEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type queryResponse struct {
	Events []resEvent `json:"events"`
}
