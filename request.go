package res

import (
	"encoding/json"
	"strconv"
	"time"

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
	// Resource embedded. Value field is nil for get and access requests.
	Resource

	// Request type. May be "access", "get", "call", or "auth".
	Type string

	// Resource method.
	// For access and get requests it is unused.
	Method string

	// Connection ID of the requesting client connection.
	// For get requests it is unused.
	CID string `json:"cid"`

	// JSON encoded method parameters, or nil if the request had no parameters.
	// For access and get requests it is unused.
	RawParams json.RawMessage `json:"params"`

	// JSON encoded access token, or nil if the request had no token.
	// For get requests it is unused.
	RawToken json.RawMessage `json:"token"`

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

	msg     *nats.Msg
	replied bool // Flag telling if a reply has been made
}

type response Request

// AccessResponse has methods for responding to access requests.
type AccessResponse interface {
	Access(get bool, call string)
	AccessDenied()
	NotFound()
}

// GetModelResponse has methods for responding to model get requests.
type GetModelResponse interface {
	Model(model interface{})
	QueryModel(model interface{}, query string)
	NotFound()
}

// GetCollectionResponse has methods for responding to collection get requests.
type GetCollectionResponse interface {
	Collection(collection interface{})
	QueryCollection(collection interface{}, query string)
	NotFound()
}

// CallResponse has methods for responding to call requests.
type CallResponse interface {
	OK(result interface{})
	NotFound()
	MethodNotFound()
	InvalidParams(message string)
	Error(err *Error)
}

// NewResponse has methods for responding to new call requests.
type NewResponse interface {
	New(rid Ref)
	NotFound()
	MethodNotFound()
	InvalidParams(message string)
	Error(err *Error)
}

// AuthResponse has methods for responding to auth requests.
type AuthResponse interface {
	OK(result interface{})
	NotFound()
	MethodNotFound()
	InvalidParams(message string)
	Error(err *Error)
}

// Static responses and events
var (
	responseAccessDenied    = []byte(`{"error":{"code":"system.accessDenied","message":"Access denied"}}`)
	responseInternalError   = []byte(`{"error":{"code":"system.internalError","message":"Internal error"}}`)
	responseNotFound        = []byte(`{"error":{"code":"system.notFound","message":"Not found"}}`)
	responseMethodNotFound  = []byte(`{"error":{"code":"system.methodNotFound","message":"Method not found"}}`)
	responseInvalidParams   = []byte(`{"error":{"code":"system.invalidParams","message":"Invalid parameters"}}`)
	responseMissingResponse = []byte(`{"error":{"code":"system.internalError","message":"Internal error: missing response"}}`)
	responseAccessGranted   = []byte(`{"result":{"get":true,"call":"*"}}`)
)

// Predefined handlers
var (
	// Access handler that provides full get and call access.
	AccessGranted AccessHandler = func(w AccessResponse, r *Request) {
		r.reply(responseAccessGranted)
	}

	// Access handler that sends a system.accessDenied error response.
	AccessDenied AccessHandler = func(w AccessResponse, r *Request) {
		r.reply(responseAccessDenied)
	}
)

// Timeout attempts to set the timeout duration of the request.
// The call has no effect if the requester has already timed out the request.
func (r *Request) Timeout(d time.Duration) {
	if d < 0 {
		panic("res: negative timeout duration")
	}
	out := []byte(`timeout:"` + strconv.FormatInt(d.Nanoseconds()/1000000, 10) + `"`)
	r.s.Tracef("<-- %s: %s", r.msg.Subject, out)

	r.send(r.msg.Reply, out)
}

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
	r.s.Tracef("<== %s: %s", r.msg.Subject, data)
	r.send(r.msg.Reply, data)
}

// send publishes an encoded data payload on a subject.
func (r *Request) send(subj string, data []byte) {
	err := r.s.nc.Publish(subj, data)
	if err != nil {
		panic(err)
	}
}

// Access sends a successful response for the access request.
// The get flag tells if the client has access to get (read) the resource.
// The call string is a comma separated list of methods that the client can
// call. Eg. "set,foo,bar". A single asterisk character ("*") means the client
// is allowed to call any method. Empty string means no calls are allowed.
func (w *response) Access(get bool, call string) {
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
func (w *response) AccessDenied() {
	(*Request)(w).reply(responseAccessDenied)
}

// NotFound sends a system.notFound response for the access request.
func (w *response) NotFound() {
	(*Request)(w).reply(responseNotFound)
}

// Model sends a successful model response for the get request.
// The model must marshal into a JSON object.
func (w *response) Model(model interface{}) {
	w.model(model, "")
}

// QueryModel sends a successful query model response for the get request.
// The model must marshal into a JSON object.
func (w *response) QueryModel(model interface{}, query string) {
	w.model(model, query)
}

// model sends a successful model response for the get request.
func (w *response) model(model interface{}, query string) {
	type modelResponse struct {
		Model interface{} `json:"model"`
		Query string      `json:"query,omitempty"`
	}

	r := (*Request)(w)
	if query != "" && r.RawQuery == "" {
		panic("res: query model response on non-query request")
	}
	// [TODO] Marshal model to a json.RawMessage to see if it is a JSON object
	(*Request)(w).success(modelResponse{Model: model, Query: query})
}

// Collection sends a successful collection response for the get request.
// The collection must marshal into a JSON array.
func (w *response) Collection(collection interface{}) {
	w.collection(collection, "")
}

// QueryCollection sends a successful query collection response for the get request.
// The collection must marshal into a JSON array.
func (w *response) QueryCollection(collection interface{}, query string) {
	w.collection(collection, query)
}

// collection sends a successful collection response for the get request.
func (w *response) collection(collection interface{}, query string) {
	type collectionResponse struct {
		Collection interface{} `json:"collection"`
		Query      string      `json:"query,omitempty"`
	}

	r := (*Request)(w)
	if query != "" && r.RawQuery == "" {
		panic("res: query collection response on non-query request")
	}
	// [TODO] Marshal collection to a json.RawMessage to see if it is a JSON array
	(*Request)(w).success(collectionResponse{Collection: collection, Query: query})
}

// OK sends a successful response for the call request.
// The result may be nil.
func (w *response) OK(result interface{}) {
	(*Request)(w).success(result)
}

// MethodNotFound sends a system.methodNotFound response for the call request.
func (w *response) MethodNotFound() {
	(*Request)(w).reply(responseMethodNotFound)
}

// InvalidParams sends a system.invalidParams response for the call request.
// An empty message will be replaced will default to "Invalid parameters".
func (w *response) InvalidParams(message string) {
	if message == "" {
		(*Request)(w).reply(responseInvalidParams)
	} else {
		(*Request)(w).error(&Error{Code: CodeInvalidParams, Message: message})
	}
}

// Error sends a custom error response for the call request.
func (w *response) Error(err *Error) {
	(*Request)(w).error(err)
}

// New sends a successful response for the new call request.
func (w *response) New(rid Ref) {
	(*Request)(w).success(rid)
}

// UnmarshalParams parses the encoded parameters and stores the result in params.
// On any error, Unmarshal panics with a system.invalidParams *Error.
func (r *Request) UnmarshalParams(params interface{}) {
	err := json.Unmarshal(r.RawParams, params)
	if err != nil {
		panic(&Error{Code: CodeInvalidParams, Message: err.Error()})
	}
}
