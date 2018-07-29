package res

import (
	"encoding/json"

	nats "github.com/nats-io/go-nats"
)

// Request types
const (
	RequestTypeAccess = "access"
	RequestTypeGet    = "get"
	RequestTypeCall   = "call"
	RequestTypeAuth   = "auth"
)

// Request represent a RES request
type Request struct {
	// Request type. May be "access", "get", "call", or "auth".
	Type string

	// Resource name. The name is the resource ID without the query.
	ResourceName string

	// Resource method.
	// For access and get requests it is unused.
	Method string

	// Path parameters parsed from the resource name
	PathParams map[string]string

	// Connection ID of the requesting client connection.
	// For get requests it is unused.
	CID string `json:"cid"`

	// JSON encoded method parameters, or nil if the request had no parameters.
	// For access and get requests it is unused.
	RawParams json.RawMessage `json:"params"`

	// JSON encoded access token, or nil if the request had no token.
	// For get requests it is unused.
	RawToken json.RawMessage `json:"token"`

	// Query part of the resource ID without the question mark separator.
	Query string `json:"query"`

	// HTTP headers sent by client on connect.
	// This field is only populated for auth requests.
	Header map[string][]string `json:"header"`

	// The host on which the URL is sought by the client. Per RFC 2616,
	// this is either the value of the "Host" header or the host name given
	// in the URL itself.
	// This field is only populated for auth requests.
	Host string `json:"host"`

	// The network address of the client sent on connect.
	// The format is not specified.
	// This field is only populated for auth requests.
	RemoteAddr string `json:"remoteAddr"`

	// The unmodified Request-URI of the Request-Line (RFC 2616, Section 5.1)
	// as sent by the client when on connect.
	// This field is only populated for auth requests.
	URI string `json:"uri"`

	s       *Service
	msg     *nats.Msg
	replied bool // Flag telling if a reply has been made
}

// AccessResponse has methods for responding to access requests.
type AccessResponse Request

// GetResponse has methods for responding to get requests.
type GetResponse Request

// CallResponse has methods for responding to call requests.
type CallResponse Request

// AuthResponse has methods for responding to auth requests.
type AuthResponse Request

// Static responses and events
var (
	responseAccessDenied    = []byte(`{"error":{"code":"system.accessDenied","message":"Access denied"}}`)
	responseInternalError   = []byte(`{"error":{"code":"system.internalError","message":"Internal error"}}`)
	responseNotFound        = []byte(`{"error":{"code":"system.notFound","message":"Not found"}}`)
	responseMethodNotFound  = []byte(`{"error":{"code":"system.methodNotFound","message":"Method not found"}}`)
	responseInvalidParams   = []byte(`{"error":{"code":"system.invalidParams","message":"Invalid parameters"}}`)
	responseMissingResponse = []byte(`{"error":{"code":"system.internalError","message":"Internal error: missing response"}}`)
)

// success sends a successful response as a reply.
func (r *Request) success(result interface{}) {
	type successResponse struct {
		Result interface{} `json:"result"`
	}

	data, err := json.Marshal(successResponse{Result: result})
	if err != nil {
		r.error(ToError(err))
		return
	}

	r.reply(data)
}

// error sends an error response as a reply.
func (r *Request) error(e *Error) {
	type errorResponse struct {
		Error *Error `json:"error"`
	}

	data, err := json.Marshal(errorResponse{Error: e})
	if err != nil {
		data = responseInternalError
	}

	r.reply(data)
}

// reply sends an encoded payload to as a reply.
// If a reply is already sent, reply will panic.
func (r *Request) reply(data []byte) {
	if r.replied {
		panic("res: response already sent on request")
	}
	r.replied = true
	if debug {
		r.s.Logf("<== %s: %s", r.msg.Subject, data)
	}
	r.send(r.msg.Reply, data)
}

// send publishes an encoded data payload on a subject.
func (r *Request) send(subj string, data []byte) {
	err := r.s.nc.Publish(subj, data)
	if err != nil {
		panic(err)
	}
}

// OK sends a successful response for the access request.
// The get flag tells if the client has access to get (read) the resource.
// The call string is a comma separated list of methods that the client can
// call. Eg. "set,foo,bar". A single asterisk character ("*") means the client
// is allowed to call any method. Empty string means no calls are allowed.
func (w *AccessResponse) OK(get bool, call string) {
	type okResponse struct {
		Get  bool   `json:"get,omitempty"`
		Call string `json:"call,omitempty"`
	}

	if !get && call == "" {
		(*Request)(w).reply(responseAccessDenied)
	} else {
		(*Request)(w).success(okResponse{Get: get, Call: call})
	}
}

// AccessDenied sends a system.accessDenied response for the access request.
func (w *AccessResponse) AccessDenied() {
	(*Request)(w).reply(responseAccessDenied)
}

// NotFound sends a system.notFound response for the access request.
func (w *AccessResponse) NotFound() {
	(*Request)(w).reply(responseNotFound)
}

// Model sends a successful model response for the get request.
// The model must marshal into a JSON object.
func (w *GetResponse) Model(model interface{}) {
	w.model(model, "")
}

// QueryModel sends a successful query model response for the get request.
// The model must marshal into a JSON object.
func (w *GetResponse) QueryModel(model interface{}, query string) {
	w.model(model, query)
}

// Collection sends a successful collection response for the get request.
// The collection must marshal into a JSON array.
func (w *GetResponse) Collection(collection interface{}) {
	w.collection(collection, "")
}

// QueryCollection sends a successful query collection response for the get request.
// The collection must marshal into a JSON array.
func (w *GetResponse) QueryCollection(collection interface{}, query string) {
	w.collection(collection, query)
}

// NotFound sends a system.notFound response for the get request.
func (w *GetResponse) NotFound() {
	(*Request)(w).reply(responseNotFound)
}

// model sends a successful model response for the get request.
func (w *GetResponse) model(model interface{}, query string) {
	type modelResponse struct {
		Model interface{} `json:"model"`
		Query string      `json:"query,omitempty"`
	}

	r := (*Request)(w)
	if query != "" && r.Query == "" {
		panic("res: query model response on non-query request")
	}
	// [TODO] Marshal model to a json.RawMessage to see if it is a JSON object
	(*Request)(w).success(modelResponse{Model: model, Query: query})
}

// collection sends a successful collection response for the get request.
func (w *GetResponse) collection(collection interface{}, query string) {
	type collectionResponse struct {
		Collection interface{} `json:"collection"`
		Query      string      `json:"query,omitempty"`
	}

	r := (*Request)(w)
	if query != "" && r.Query == "" {
		panic("res: query collection response on non-query request")
	}
	// [TODO] Marshal collection to a json.RawMessage to see if it is a JSON array
	(*Request)(w).success(collectionResponse{Collection: collection, Query: query})
}

// OK sends a successful response for the call request.
// The result may be nil.
func (w *CallResponse) OK(result interface{}) {
	(*Request)(w).success(result)
}

// NotFound sends a system.notFound response for the call request.
func (w *CallResponse) NotFound() {
	(*Request)(w).reply(responseNotFound)
}

// MethodNotFound sends a system.methodNotFound response for the call request.
func (w *CallResponse) MethodNotFound() {
	(*Request)(w).reply(responseMethodNotFound)
}

// InvalidParams sends a system.invalidParams response for the call request.
// An empty message will be replaced will default to "Invalid parameters".
func (w *CallResponse) InvalidParams(message string) {
	if message == "" {
		(*Request)(w).reply(responseInvalidParams)
	} else {
		(*Request)(w).error(&Error{Code: CodeInvalidParams, Message: message})
	}
}

// Error sends a custom error response for the call request.
func (w *CallResponse) Error(err *Error) {
	(*Request)(w).error(err)
}

// OK sends a successful response for the auth request.
// The result may be nil.
func (w *AuthResponse) OK(result interface{}) {
	(*Request)(w).success(result)
}

// NotFound sends a system.notFound response for the auth request.
func (w *AuthResponse) NotFound() {
	(*Request)(w).reply(responseNotFound)
}

// MethodNotFound sends a system.methodNotFound response for the auth request.
func (w *AuthResponse) MethodNotFound() {
	(*Request)(w).reply(responseMethodNotFound)
}

// InvalidParams sends a system.invalidParams response for the auth request.
// An empty message will be replaced will default to "Invalid parameters".
func (w *AuthResponse) InvalidParams(message string) {
	if message == "" {
		(*Request)(w).reply(responseInvalidParams)
	} else {
		(*Request)(w).error(&Error{Code: CodeInvalidParams, Message: message})
	}
}

// Error sends a custom error response for the auth request.
func (w *AuthResponse) Error(err *Error) {
	(*Request)(w).error(err)
}

// UnmarshalParams parses the encoded parameters and stores the result in params.
// On any error, Unmarshal panics with a system.invalidParams error.
func (r *Request) UnmarshalParams(params interface{}) {
	err := json.Unmarshal(r.RawParams, params)
	if err != nil {
		// [TODO] Maybe should return err instead. These panics can be triggered
		// by clients, cluttering the log with errors.
		panic(&Error{Code: CodeInvalidParams, Message: err.Error()})
	}
}

// Event sends a resource event on the requested resource name.
// If the event is "change", "add", "remove", or "reaccess", the payload must
// contain the parameters required for the event, as specified in:
// https://github.com/jirenius/resgate/blob/master/docs/res-service-protocol.md#events
func (r *Request) Event(event string, payload interface{}) {
	r.s.send("event."+r.ResourceName+"."+event, payload)
}
