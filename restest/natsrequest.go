package restest

import "strings"

// NATSRequest represents a requests sent over NATS to the service.
type NATSRequest struct {
	c   *MockConn
	inb string
}

// Response gets the next pending message that is published to NATS by the
// service.
//
// If no message is received within a set amount of time, or if the message is
// not a response to the request, it will log it as a fatal error.
func (nr *NATSRequest) Response() *Msg {
	m := nr.c.GetMsg()
	m.AssertSubject(nr.inb)
	return m
}

// Get sends a get request to the service.
//
// The resource ID, rid, may contain a query part:
// 	test.model?q=foo
func (c *MockConn) Get(rid string) *NATSRequest {
	rname, q := parseRID(rid)
	return c.Request("get."+rname, Request{Query: q})
}

// Call sends a call request to the service.
//
// A nil req value sends a DefaultCallRequest.
//
// The resource ID, rid, may contain a query part:
// 	test.model?q=foo
func (c *MockConn) Call(rid string, method string, req *Request) *NATSRequest {
	if req == nil {
		req = DefaultCallRequest()
	}
	r := *req
	rname, q := parseRID(rid)
	if q != "" {
		r.Query = q
	}
	return c.Request("call."+rname+"."+method, r)
}

// Auth sends an auth request to the service.
//
// A nil req value sends a DefaultAuthRequest.
//
// The resource ID, rid, may contain a query part:
// 	test.model?q=foo
func (c *MockConn) Auth(rid string, method string, req *Request) *NATSRequest {
	if req == nil {
		req = DefaultAuthRequest()
	}
	r := *req
	rname, q := parseRID(rid)
	if q != "" {
		r.Query = q
	}
	return c.Request("auth."+rname+"."+method, r)
}

// Access sends an access request to the service.
//
// A nil req value sends a DefaultAccessRequest.
//
// The resource ID, rid, may contain a query part:
// 	test.model?q=foo
func (c *MockConn) Access(rid string, req *Request) *NATSRequest {
	if req == nil {
		req = DefaultAuthRequest()
	}
	r := *req
	rname, q := parseRID(rid)
	if q != "" {
		r.Query = q
	}
	return c.Request("access."+rname, r)
}

func parseRID(rid string) (name string, query string) {
	i := strings.IndexByte(rid, '?')
	if i == -1 {
		return rid, ""
	}

	return rid[:i], rid[i+1:]
}
