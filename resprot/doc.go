/*
Package resprot provides low level structs and methods for communicating with
services using the RES Service Protocol over NATS server.

https://github.com/resgateio/resgate/blob/master/docs/res-service-protocol.md


Usage

Make a request:

	conn, _ := nats.Connect("nats://127.0.0.1:4222")
	response := resprot.SendRequest(conn, "call.example.ping", nil, time.Second)

Get a model:

	response := resprot.SendRequest(conn, "get.example.model", nil, time.Second)

	var model struct {
		Message string `json:"message"`
	}
	_, err := response.ParseModel(&model)

Call a method:

	response := resprot.SendRequest(conn, "call.math.add", resprot.Request{Params: struct {
		A float64 `json:"a"`
		B float64 `json:"a"`
	}{5, 6}}, time.Second)

	var result struct {
		Sum float64 `json:"sum"`
	}
	err := response.ParseResult(&result)

*/
package resprot
