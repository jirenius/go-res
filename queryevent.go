package res

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	nats "github.com/nats-io/go-nats"
)

const queryEventChannelSize = 10

// QueryRequest has methods for responding to query requests.
type QueryRequest interface {
	Resource
	NotFound()
	Error(err *Error)
	Timeout(d time.Duration)
}

type queryRequest struct {
	resource
	msg     *nats.Msg
	events  []resEvent
	err     *Error
	replied bool // Flag telling if a reply has been made
}

type queryEvent struct {
	r   resource
	sub *nats.Subscription
	ch  chan *nats.Msg
	cb  func(r QueryRequest)
}

// ChangeEvent adds a change event to the query response
// If ev is empty, no event is added.
func (qr *queryRequest) ChangeEvent(ev map[string]interface{}) {
	if len(ev) == 0 {
		return
	}
	qr.events = append(qr.events, resEvent{Event: "change", Data: changeEvent{Values: ev}})
}

// AddEvent adds an add event to the query response,
// adding the value v at index idx.
func (qr *queryRequest) AddEvent(v interface{}, idx int) {
	if idx < 0 {
		panic("res: add event idx less than zero")
	}
	qr.events = append(qr.events, resEvent{Event: "add", Data: addEvent{Value: v, Idx: idx}})
}

// RemoveEvent adds a remove event to the query response,
// removing the value at index idx.
func (qr *queryRequest) RemoveEvent(idx int) {
	if idx < 0 {
		panic("res: remove event idx less than zero")
	}
	qr.events = append(qr.events, resEvent{Event: "remove", Data: removeEvent{Idx: idx}})
}

// NotFound sends a system.notFound response for the query request.
func (qr *queryRequest) NotFound() {
	qr.reply(responseNotFound)
}

// Error sends a custom error response for the query request.
func (qr *queryRequest) Error(err *Error) {
	qr.error(err)
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
