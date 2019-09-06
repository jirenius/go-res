package res

import (
	"errors"
	"fmt"
	"time"
)

// getRequest implements the ModelRequest and CollectionRequest interfaces.
// Instead of sending the response to NATS (like Request), getRequest stores
// the reply values in memory.
type getRequest struct {
	*resource
	replied bool // Flag telling if a reply has been made
	value   interface{}
	err     error
}

func (r *getRequest) Model(model interface{}) {
	r.reply()
	r.value = model
}

func (r *getRequest) QueryModel(model interface{}, query string) {
	if query != "" && r.query == "" {
		panic("res: query model response on non-query get request")
	}
	r.reply()
	r.value = model
}

func (r *getRequest) Collection(collection interface{}) {
	r.reply()
	r.value = collection
}

func (r *getRequest) QueryCollection(collection interface{}, query string) {
	if query != "" && r.query == "" {
		panic("res: query model response on non-query get request")
	}
	r.reply()
	r.value = collection
}

func (r *getRequest) NotFound() {
	r.Error(ErrNotFound)
}

func (r *getRequest) Error(err *Error) {
	r.reply()
	r.err = err
}

func (r *getRequest) Timeout(d time.Duration) {
	// Implement once an internal timeout for requests is implemented
}

func (r *getRequest) ForValue() bool {
	return true
}

func (r *getRequest) reply() {
	if r.replied {
		panic("res: response already sent on get request")
	}
	r.replied = true
}

func (r *getRequest) executeHandler() {
	r.inGet = true
	// Recover from panics inside handlers
	defer func() {
		r.inGet = false

		v := recover()
		if v == nil {
			return
		}

		var str string

		switch e := v.(type) {
		case *Error:
			if !r.replied {
				r.Error(e)
				// Return without logging as panicing with a *Error is considered
				// a valid way of sending an error response.
				return
			}
			str = e.Message
		case error:
			str = e.Error()
			if !r.replied {
				r.Error(ToError(e))
			}
		case string:
			str = e
			if !r.replied {
				r.Error(ToError(errors.New(e)))
			}
		default:
			str = fmt.Sprintf("%v", e)
			if !r.replied {
				r.Error(ToError(errors.New(str)))
			}
		}

		r.s.Logf("error handling get request %#v: %s", r.rname, str)
	}()

	h := r.h
	if h.Get == nil {
		r.Error(ErrNotFound)
		return
	}
	h.Get(r)

	if !r.replied {
		r.Error(InternalError(fmt.Errorf("missing response on get request for %#v", r.rname)))
	}
}
