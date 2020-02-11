package res

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	nats "github.com/nats-io/nats.go"
)

const queryEventChannelSize = 10

// QueryRequest has methods for responding to query requests.
type QueryRequest interface {
	Resource
	Model(model interface{})
	Collection(collection interface{})
	NotFound()
	InvalidQuery(message string)
	Error(err error)
	Timeout(d time.Duration)
}

type queryRequest struct {
	resource
	msg     *nats.Msg
	events  []resEvent
	replied bool // Flag telling if a reply has been made
}

type queryEvent struct {
	r   resource
	sub *nats.Subscription
	ch  chan *nats.Msg
	cb  func(r QueryRequest)
}

// Model sends a model response for the query request.
// The model represents the current state of query model
// for the given query.
// Only valid for a query model resource.
func (qr *queryRequest) Model(model interface{}) {
	if qr.h.Type == TypeCollection {
		panic("res: model response not allowed on query collections")
	}
	qr.success(modelResponse{Model: model})
}

// Collection sends a collection response for the query request.
// The collection represents the current state of query collection
// for the given query.
// Only valid for a query collection resource.
func (qr *queryRequest) Collection(collection interface{}) {
	if qr.h.Type == TypeModel {
		panic("res: collection response not allowed on query models")
	}
	qr.success(collectionResponse{Collection: collection})
}

// ChangeEvent adds a change event to the query response.
// If ev is empty, no event is added.
// Only valid for a query model resource.
func (qr *queryRequest) ChangeEvent(ev map[string]interface{}) {
	if qr.h.Type == TypeCollection {
		panic("res: change event not allowed on query collections")
	}
	if len(ev) == 0 {
		return
	}
	qr.events = append(qr.events, resEvent{Event: "change", Data: changeEvent{Values: ev}})
}

// AddEvent adds an add event to the query response,
// adding the value v at index idx.
// Only valid for a query collection resource.
func (qr *queryRequest) AddEvent(v interface{}, idx int) {
	if qr.h.Type == TypeModel {
		panic("res: add event not allowed on query models")
	}
	if idx < 0 {
		panic("res: add event idx less than zero")
	}
	qr.events = append(qr.events, resEvent{Event: "add", Data: addEvent{Value: v, Idx: idx}})
}

// RemoveEvent adds a remove event to the query response,
// removing the value at index idx.
// Only valid for a query collection resource.
func (qr *queryRequest) RemoveEvent(idx int) {
	if qr.h.Type == TypeModel {
		panic("res: remove event not allowed on query models")
	}
	if idx < 0 {
		panic("res: remove event idx less than zero")
	}
	qr.events = append(qr.events, resEvent{Event: "remove", Data: removeEvent{Idx: idx}})
}

// NotFound sends a system.notFound response for the query request.
func (qr *queryRequest) NotFound() {
	qr.reply(responseNotFound)
}

// InvalidQuery sends a system.invalidQuery response for the query request.
// An empty message will default to "Invalid query".
func (qr *queryRequest) InvalidQuery(message string) {
	if message == "" {
		qr.reply(responseInvalidQuery)
	} else {
		qr.error(&Error{Code: CodeInvalidQuery, Message: message})
	}
}

// Error sends a custom error response for the query request.
func (qr *queryRequest) Error(err error) {
	qr.error(ToError(err))
}

// Timeout attempts to set the timeout duration of the query request.
// The call has no effect if the requester has already timed out the request.
func (qr *queryRequest) Timeout(d time.Duration) {
	if d < 0 {
		panic("res: negative timeout duration")
	}
	out := []byte(`timeout:"` + strconv.FormatInt(int64(d/time.Millisecond), 10) + `"`)
	qr.s.rawEvent(qr.msg.Reply, out)
}

// startQueryListener listens for query requests and passes them on to a worker.
func (qe *queryEvent) startQueryListener() {
	for m := range qe.ch {
		m := m
		qe.r.s.runWith(qe.r.Group(), func() {
			qe.handleQueryRequest(m)
		})
	}
}

// handleQueryRequest is called by the query listener on incoming query requests.
func (qe *queryEvent) handleQueryRequest(m *nats.Msg) {
	s := qe.r.s
	s.tracef("Q=> %s: %s", qe.r.rname, m.Data)

	qr := &queryRequest{
		resource: qe.r,
		msg:      m,
	}

	var rqr resQueryRequest
	err := json.Unmarshal(m.Data, &rqr)
	if err != nil {
		s.errorf("Error unmarshaling incoming query request: %s", err)
		qr.error(ToError(err))
		return
	}

	if rqr.Query == "" {
		s.errorf("Missing query on incoming query request: %s", err)
		qr.reply(responseMissingQuery)
		return
	}

	qr.query = rqr.Query

	qr.executeCallback(qe.cb)
	if qr.replied {
		return
	}

	var data []byte
	if len(qr.events) == 0 {
		data = responseNoQueryEvents
	} else {
		data, err = json.Marshal(successResponse{Result: queryResponse{Events: qr.events}})
		if err != nil {
			data = responseInternalError
		}
	}
	qr.reply(data)
}

func (qr *queryRequest) executeCallback(cb func(QueryRequest)) {
	// Recover from panics inside query event callback
	defer func() {
		v := recover()
		if v == nil {
			return
		}

		var str string

		switch e := v.(type) {
		case *Error:
			if !qr.replied {
				qr.error(e)
				// Return without logging, as panicing with an *Error is considered
				// a valid way of sending an error response.
				return
			}
			str = e.Message
		case error:
			str = e.Error()
			if !qr.replied {
				qr.error(ToError(e))
			}
		case string:
			str = e
			if !qr.replied {
				qr.error(ToError(errors.New(e)))
			}
		default:
			str = fmt.Sprintf("%v", e)
			if !qr.replied {
				qr.error(ToError(errors.New(str)))
			}
		}

		qr.s.errorf("Error handling query request %s: %s", qr.rname, str)
	}()

	cb(qr)
}

// error sends an error response as a reply.
func (qr *queryRequest) error(e *Error) {
	data, err := json.Marshal(errorResponse{Error: e})
	if err != nil {
		data = responseInternalError
	}
	qr.reply(data)
}

// success sends a successful response as a reply.
func (qr *queryRequest) success(result interface{}) {
	data, err := json.Marshal(successResponse{Result: result})
	if err != nil {
		qr.error(ToError(err))
		return
	}

	qr.reply(data)
}

// reply sends an encoded payload to as a reply.
// If a reply is already sent, reply will log an error.
func (qr *queryRequest) reply(payload []byte) {
	if qr.replied {
		qr.s.errorf("Response already sent on query request %s", qr.rname)
		return
	}
	qr.replied = true

	qr.s.tracef("<=Q %s: %s", qr.rname, payload)
	err := qr.s.nc.Publish(qr.msg.Reply, payload)
	if err != nil {
		qr.s.errorf("Error sending query reply %s: %s", qr.rname, err)
	}
}
