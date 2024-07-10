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
	IsHTTP     bool                `json:"isHttp"`
}

type metaObject struct {
	Status int                 `json:"status,omitempty"`
	Header map[string][]string `json:"header,omitempty"`
}

type successResponse struct {
	Result interface{} `json:"result"`
	Meta   *metaObject `json:"meta,omitempty"`
}

type resourceResponse struct {
	Resource Ref         `json:"resource"`
	Meta     *metaObject `json:"meta,omitempty"`
}

type errorResponse struct {
	Error *Error      `json:"error"`
	Meta  *metaObject `json:"meta,omitempty"`
}

type accessResponse struct {
	Get  bool        `json:"get,omitempty"`
	Call string      `json:"call,omitempty"`
	Meta *metaObject `json:"meta,omitempty"`
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
	TID   string      `json:"tid,omitempty"`
}

type tokenResetEvent struct {
	TIDs    []string `json:"tids"`
	Subject string   `json:"subject"`
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
