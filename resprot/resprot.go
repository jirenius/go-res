package resprot

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/jirenius/go-res"
	"github.com/nats-io/nats.go"
)

var (
	errInvalidResponse           = errors.New("invalid response")
	errResourceResponse          = errors.New("response is a resource response")
	errInvalidModelResponse      = errors.New("invalid model response")
	errInvalidCollectionResponse = errors.New("invalid collection response")
)

var (
	emptyRequest = []byte(`{}`)
)

var (
	errUnexpectedEnd     = errors.New("unexpected end of JSON input")
	errUnexpectedBracket = errors.New("unexpected character '[' looking for beginning of primitive value or data value object")
	errMissingDataKey    = errors.New(`object is missing "data" key string`)
)

// Request represents the payload of a request.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#requests
type Request struct {

	// CID is the requesting clients connection ID.
	//
	// Valid for access, call, and auth requests. May be omitted on
	// inter-service requests.
	CID string `json:"cid,omitempty"`

	// Params is the requests parameters.
	//
	// Valid for call and auth requests. May be omitted.
	Params interface{} `json:"params,omitempty"`

	// Token is the RES client's access token.
	//
	// Valid for access, call, and auth requests. May be omitted.
	Token interface{} `json:"token,omitempty"`

	// Header is the request HTTP headers of the client, provided on connect.
	// connect.
	//
	// Valid for auth requests. May be omitted on inter-service requests.
	Header map[string][]string `json:"header,omitempty"`

	// Host is the host on which the URL is sought by the client. Per RFC 2616,
	// this is either the value of the "Host" header or the host name given in
	// the URL itself.
	//
	// Valid for auth requests. May be omitted on inter-service requests. The
	// format is not specified.
	Host string `json:"host,omitempty"`

	// RemoteAddr is the network address of the client, provided on connect.
	//
	// Valid for auth requests. May be omitted on inter-service requests.
	RemoteAddr string `json:"remoteAddr,omitempty"`

	// URI is the unmodified Request-URI of the Request-Line (RFC 2616,
	// Section 5.1) as provided by the client on connect.
	//
	// Valid for auth requests. May be omitted on inter-service requests.
	URI string `json:"uri,omitempty"`

	// Query is the query part of the resource ID without the question mark
	// separator.
	//
	// Valid for access, get, call, auth, and query requests. May be omitted,
	// except for on query requests.
	Query string `json:"query,omitempty"`
}

// Response represents the response to a request.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#response
type Response struct {

	// Result is the successful result of a request.
	Result json.RawMessage `json:"result"`

	// Resource is a reference to a resource.
	//
	// Valid for responses to call and auth requests.
	Resource res.Ref `json:"resource"`

	// Error is the request error.
	Error *res.Error `json:"error"`
}

// ParseResponse unmarshals a JSON encoded RES response.
//
// If the response is not valid, the Error field will be set to a *res.Error with code system.internalError.
func ParseResponse(data []byte) Response {
	var r Response
	if len(data) > 0 {
		err := json.Unmarshal(data, &r)
		if err != nil {
			r.Error = res.InternalError(err)
			// A valid response MUST have one of the members set
		} else if r.Error == nil && r.Resource == "" && r.Result == nil {
			r.Error = res.InternalError(errInvalidResponse)
		}
	} else {
		r.Error = res.InternalError(errInvalidResponse)
	}
	return r
}

// HasError returns true if the response has an error.
func (r Response) HasError() bool {
	return r.Error != nil
}

// HasResource returns true if the response is a resource response.
func (r Response) HasResource() bool {
	return r.Error == nil && r.Resource != ""
}

// HasResult returns true if the response is a successful a result response.
func (r Response) HasResult() bool {
	return r.Error == nil && r.Resource == ""
}

// ParseModel unmarshals the model from the response of a successful model
// get request.
//
// On success, the get response query value is returned, if one was set.
func (r Response) ParseModel(model interface{}) (string, error) {
	if r.Error != nil {
		return "", r.Error
	}

	if r.Resource != "" {
		return "", errResourceResponse
	}

	var result GetResult
	if len(r.Result) > 0 {
		err := json.Unmarshal(r.Result, &result)
		if err != nil {
			return "", err
		}
	}

	if result.Collection != nil || result.Model == nil {
		return "", errInvalidModelResponse
	}

	err := json.Unmarshal(result.Model, model)
	if err != nil {
		return "", err
	}

	return result.Query, nil
}

// ParseCollection unmarshals the collection from the response of a
// successful collection get request.
//
// On success, the get response query value is returned, if one was set.
func (r Response) ParseCollection(collection interface{}) (string, error) {
	if r.Error != nil {
		return "", r.Error
	}

	if r.Resource != "" {
		return "", errResourceResponse
	}

	var result GetResult
	if len(r.Result) > 0 {
		err := json.Unmarshal(r.Result, &result)
		if err != nil {
			return "", err
		}
	}

	if result.Model != nil || result.Collection == nil {
		return "", errInvalidCollectionResponse
	}

	err := json.Unmarshal(result.Collection, collection)
	if err != nil {
		return "", err
	}

	return result.Query, nil
}

// AccessResult returns the get and call values from the response of a successful access request.
func (r Response) AccessResult() (bool, string, error) {
	if r.Error != nil {
		return false, "", r.Error
	}

	if r.Resource != "" {
		return false, "", errResourceResponse
	}

	var result AccessResult
	if len(r.Result) > 0 {
		err := json.Unmarshal(r.Result, &result)
		if err != nil {
			return false, "", err
		}
	}

	return result.Get, result.Call, nil
}

// ParseResult unmarshals the result from the response of a successful
// request.
func (r Response) ParseResult(v interface{}) error {
	if r.Error != nil {
		return r.Error
	}

	if r.Resource != "" {
		return errResourceResponse
	}

	if len(r.Result) > 0 {
		err := json.Unmarshal(r.Result, &v)
		if err != nil {
			return err
		}
	}

	return nil
}

// AccessResult is the result of an access request.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#access-request
type AccessResult struct {
	Get  bool   `json:"get,omitempty"`
	Call string `json:"call,omitempty"`
}

// GetResult is the result of a get request.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#get-request
type GetResult struct {
	Model      json.RawMessage `json:"model,omitempty"`
	Collection json.RawMessage `json:"collection,omitempty"`
	Query      string          `json:"query,omitempty"`
}

// ResetEvent is the payload of a system reset event.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#system-reset-event
type ResetEvent struct {
	Resources []string `json:"resources,omitempty"`
	Access    []string `json:"access,omitempty"`
}

// TokenEvent is the payload of a connection token event.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#connection-token-event
type TokenEvent struct {
	Token interface{} `json:"token"`
}

// ChangeEvent is the payload of a model change event.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#model-change-event
type ChangeEvent struct {
	Values map[string]interface{} `json:"values"`
}

// AddEvent is the payload of a collection add event.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#collection-add-event
type AddEvent struct {
	Value interface{} `json:"value"`
	Idx   int         `json:"idx"`
}

// RemoveEvent is the payload of a collection remove event.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#collection-remove-event
type RemoveEvent struct {
	Idx int `json:"idx"`
}

// QueryEvent is the payload of a query event.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#query-event
type QueryEvent struct {
	Subject string `json:"subject"`
}

// EventEntry is a single event entry in a response to a query request.
type EventEntry struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// QueryRequest is a query request.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#query-request
type QueryRequest struct {
	Query string `json:"query"`
}

// QueryResult is the result of a query request.
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#query-request
type QueryResult struct {
	Events []EventEntry `json:"events"`
}

// SendRequest sends a request over NATS and unmarshals the response before
// returning it.
//
// If any error is encountered, the Error field will be set.
//
// if req is nil, an empty json object, {}, will be sent as payload instead.
//
// SendRequest handles pre-responses that may extend timeout. See:
// https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md#pre-response
func SendRequest(nc res.Conn, subject string, req interface{}, timeout time.Duration, onTimeoutExtend ...func(time.Duration)) Response {
	var r Response

	// Marshal the request
	var data []byte
	if req != nil {
		dta, err := json.Marshal(req)
		if err != nil {
			r.Error = res.InternalError(err)
			return r
		}
		data = dta
	} else {
		data = emptyRequest
	}

	// Manually create a response inbox
	inbox := nats.NewInbox()

	// Subscribe to response inbox
	ch := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe(inbox, ch)
	if err != nil {
		r.Error = res.InternalError(err)
		return r
	}
	defer sub.Unsubscribe()

	// Publish request
	err = nc.PublishRequest(subject, inbox, data)
	if err != nil {
		r.Error = res.InternalError(err)
		return r
	}

	// Set timeout timer
	timer := time.NewTimer(timeout)

	for {
		select {
		case <-timer.C:
			r.Error = res.ErrTimeout
			return r
		case msg := <-ch:
			// Is the first character a-z or A-Z?
			// Then it is a pre-response.
			if len(msg.Data) == 0 || (msg.Data[0]|32) < 'a' || (msg.Data[0]|32) > 'z' {
				return ParseResponse(msg.Data)
			}

			// Parse pre-response using reflect.StructTag
			// as it uses the same format.
			tag := reflect.StructTag(msg.Data)

			if v, ok := tag.Lookup("timeout"); ok {
				if ms, err := strconv.Atoi(v); err == nil {
					// Stop previous timer and make a new one.
					timer.Stop()
					d := time.Duration(ms) * time.Millisecond
					timer = time.NewTimer(d)
					for _, f := range onTimeoutExtend {
						f(d)
					}
				}
			}
		}
	}
}

// UnmarshalDataValue parses the JSON-encoded data and stores the result in the value pointed to by v, similar to json.Unmarshal.
//
// If the JSON data starts with an object, UnmarshalDataValue will use the value of the object key "data" to store in v, or will return an error if the object key "data" does not exist.
//
// If the JSON data start with an array, UnmarshalDataValue will return an error.
//
// 	UnmarshalDataValue([]byte(`42`), v)                     // sets v to 42
// 	UnmarshalDataValue([]byte(`"foo"`), v)                  // sets v to "foo"
// 	UnmarshalDataValue([]byte(`{"data":true}`), v)          // sets v to true
// 	UnmarshalDataValue([]byte(`{"data":["foo","bar"]}`), v) // sets v to []string{"foo", "bar"}
// 	UnmarshalDataValue([]byte(`{"foo":"bar"}`), v)          // returns error
// 	UnmarshalDataValue([]byte(`[1,2,3]`), v)                // returns error
//
// UnmarshalDataValue can be used to implement the json.Unmarshaler interface:
//
// 	func (t *T) UnmarshalJSON([]byte) error
// 		return UnmarshalDataValue(data, t)
// 	}
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-protocol.md#data-values
func UnmarshalDataValue(data []byte, v interface{}) error {
	// Get first non-whitespace character
	var c byte
	i := 0
	for {
		if i == len(data) {
			return errUnexpectedEnd
		}
		c = data[i]
		if c != 0x20 && c != 0x09 && c != 0x0A && c != 0x0D {
			break
		}
		i++
	}

	// Data values cannot be arrays
	if c == '[' {
		return errUnexpectedBracket
	}

	// Data value object start
	if c == '{' {
		var dv struct {
			Data json.RawMessage `json:"data"`
		}
		err := json.Unmarshal(data, &dv)
		if err != nil {
			return err
		}
		if dv.Data == nil {
			return errMissingDataKey
		}
		data = dv.Data
	}

	return json.Unmarshal(data, v)
}

// MarshalDataValue returns the JSON encoding of v, similar to json.Marshal.
//
// If v encodes into a JSON object or array, MarshalDataValue will wrap the value in a data object, where the value is stored under the key "data".
//
// 	MarshalDataValue(42)                        // Returns []byte(`42`)
// 	MarshalDataValue("foo"), v)                 // Returns []byte(`"foo"`)
// 	MarshalDataValue([]string{"foo", "bar"})    // Returns []byte(`{"data":["foo","bar"]}`)
// 	MarshalDataValue(map[string]int{"foo": 42}) // Returns []byte(`{"data":{"foo":42}}`)
//
// MarshalDataValue can be used to implement the json.Marshaler interface:
//
// 	func (t T) MarshalJSON() ([]byte, error)
// 		return MarshalDataValue(t)
// 	}
//
// See:
// https://github.com/resgateio/resgate/blob/master/docs/res-protocol.md#data-values
func MarshalDataValue(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	c := data[0]
	// Wrap in data value object if the value is an object or array
	if c == '[' || c == '{' {
		o := make([]byte, len(data)+9) // 9 = len(`{"data":}`)
		copy(o, `{"data":`)
		copy(o[8:], data)
		o[len(o)-1] = '}'
		return o, nil
	}

	return data, nil
}
