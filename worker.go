package res

import (
	"encoding/json"
	"errors"
	"fmt"

	nats "github.com/nats-io/go-nats"
)

type work struct {
	s     *Service
	wid   string   // Worker ID for the work queue
	queue []func() // Callback queue
}

// startWorker starts a new resource worker that will listen for resources to
// process requests on.
func (s *Service) startWorker(ch chan *work) {
	for w := range ch {
		w.processQueue()
	}
	s.wg.Done()
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
	delete(w.s.rwork, w.wid)
	w.s.mu.Unlock()
}

// processRequest is executed by the worker to process an incoming request.
func (s *Service) processRequest(m *nats.Msg, rtype, rname, method string, hs *Handlers, pathParams map[string]string) {
	r := Request{
		Resource: Resource{
			rname:      rname,
			pathParams: pathParams,
			s:          s,
			hs:         hs,
		},
		rtype:  rtype,
		method: method,
		msg:    m,
	}

	if hs == nil {
		r.reply(responseNotFound)
		return
	}

	var rc resRequest
	err := json.Unmarshal(m.Data, &rc)
	if err != nil {
		s.Logf("error unmarshaling incoming request: %s", err)
		r.error(ToError(err))
		return
	}

	r.cid = rc.CID
	r.params = rc.Params
	r.token = rc.Token
	r.header = rc.Header
	r.host = rc.Host
	r.remoteAddr = rc.RemoteAddr
	r.uri = rc.URI
	r.query = rc.Query

	r.executeHandler()
}

func (r *Request) executeHandler() {
	// Recover from panics inside handlers
	defer func() {
		v := recover()
		if v == nil {
			return
		}

		var str string

		switch e := v.(type) {
		case *Error:
			if !r.replied {
				r.error(e)
				// Return without logging as panicing with a *Error is considered
				// a valid way of sending an error response.
				return
			}
			str = e.Message
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

		r.s.Logf("error handling request %s: %s", r.msg.Subject, str)
	}()

	hs := r.hs

	switch r.rtype {
	case "access":
		if hs.Access == nil {
			// No handling. Assume the access requests is handled by other services.
			return
		}
		hs.Access(r)
	case "get":
		switch hs.rtype {
		case rtypeUnset:
			r.reply(responseNotFound)
			return
		case rtypeModel:
			hs.GetModel(r)
		case rtypeCollection:
			hs.GetCollection(r)
		}
	case "call":
		if r.method == "new" {
			h := hs.New
			if h == nil {
				r.reply(responseMethodNotFound)
				return
			}
			h(r)
		} else {
			var h CallHandler
			if hs.Call != nil {
				h = hs.Call[r.method]
			}
			if h == nil {
				r.reply(responseMethodNotFound)
				return
			}
			h(r)
		}
	case "auth":
		var h AuthHandler
		if hs.Auth != nil {
			h = hs.Auth[r.method]
		}
		if h == nil {
			r.reply(responseMethodNotFound)
			return
		}
		h(r)
	default:
		r.s.Logf("unknown request type: %s", r.Type)
		return
	}

	if !r.replied {
		r.reply(responseMissingResponse)
	}
}
