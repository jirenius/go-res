package res

import (
	"encoding/json"
	"errors"
	"fmt"

	nats "github.com/nats-io/go-nats"
)

type work struct {
	s     *Service
	rname string   // Resource name for the work queue
	queue []func() // Callback queue
}

// startWorker starts a new resource worker that will listen for resources to
// process requests on.
func (s *Service) startWorker(ch chan *work) {
	for w := range ch {
		w.processQueue()
	}
}

func (w *work) processQueue() {
	var f func()
	idx := 0

	w.s.mu.Lock()
	for len(w.queue) > idx {
		f = w.queue[idx]
		w.s.mu.Unlock()
		idx++
		f()
		w.s.mu.Lock()
	}
	// Work complete
	delete(w.s.rwork, w.rname)
	w.s.mu.Unlock()
}

// processRequest is executed by the worker to process an incoming request.
func (s *Service) processRequest(m *nats.Msg, rtype, rname, method string, hs *Handlers, params map[string]string) {
	var r Request
	err := json.Unmarshal(m.Data, &r)
	r.s = s
	r.msg = m
	r.Type = rtype
	r.ResourceName = rname
	r.Method = method
	r.PathParams = params

	if err != nil {
		s.Logf("error unmarshalling incoming request: %s", err)
		r.error(ToError(err))
		return
	}

	r.executeHandler(hs)
}

func (r *Request) executeHandler(hs *Handlers) {
	// Recover from panics inside handlers
	defer func() {
		v := recover()
		if v == nil {
			return
		}

		var str string

		switch e := v.(type) {
		case *Error:
			str = e.Message
			if !r.replied {
				r.error(e)
			}
		case error:
			str = e.Error()
			if !r.replied {
				r.error(ToError(e))
			}
		case string:
			str = e
			if !r.replied {
				r.error(ToError(errors.New(e)))
			}
		default:
			str = fmt.Sprintf("%v", e)
			if !r.replied {
				r.error(ToError(errors.New(str)))
			}
		}

		r.s.Log("error handling request %s: %s", r.msg.Subject, str)
	}()

	switch r.Type {
	case "access":
		if hs.Access == nil {
			// No handling. Access requests might be handled by other services.
			return
		}
		hs.Access(r, (*AccessResponse)(r))
	case "get":
		if hs.Get == nil {
			r.reply(responseNotFound)
			return
		}
		hs.Get(r, (*GetResponse)(r))
	case "call":
		var h CallHandler
		if hs.Call != nil {
			h = hs.Call[r.Method]
		}
		if h == nil {
			r.reply(responseMethodNotFound)
			return
		}
		h(r, (*CallResponse)(r))
	case "auth":
		var h AuthHandler
		if hs.Auth != nil {
			h = hs.Auth[r.Method]
		}
		if h == nil {
			r.reply(responseMethodNotFound)
			return
		}
		h(r, (*AuthResponse)(r))
	default:
		r.s.Logf("unknown request type: %s", r.Type)
		return
	}

	if !r.replied {
		r.reply(responseMissingResponse)
	}
}
