# RES service package

A [Go](http://golang.org) package implementing the RES-Service protocol for [Resgate - Realtime API Gateway](https://github.com/jirenius/resgate).

[![GoDoc](https://godoc.org/github.com/jirenius/go-res?status.svg)](http://godoc.org/github.com/jirenius/go-res)

## Installation

```bash
# Service
go get github.com/jirenius/go-res
```

## Hello world example

Install and run [NATS server](https://nats.io/download/nats-io/gnatsd/) and [Resgate](https://github.com/jirenius/resgate):

```
go get github.com/nats-io/gnatsd
gnatsd
```

```
go get github.com/jirenius/resgate
resgate
```
**Service** 
```go
package main

import (
	"fmt"
	"os"

	"github.com/jirenius/go-res"
)

type Model struct {
	Message string `json:"message"`
}

var myModel = &Model{Message: "Hello Go World"}

func main() {
	// Enable debug logging
	res.SetDebug(true)

	// Create a new RES Service
	s := res.NewService("exampleService")

	// Add handlers for "exampleService.myModel" resource
	s.Handle("myModel",
		res.Access(func(r *res.Request, w *res.AccessResponse) {
			w.OK(true, "*")
		}),
		res.Get(func(r *res.Request, w *res.GetResponse) {
			w.Model(myModel)
		}),
		res.Call("set", func(r *res.Request, w *res.CallResponse) {
			var p struct {
				Message *string `json:"message,omitempty"`
			}
			r.UnmarshalParams(&p)

			// Check if the message property was changed
			if p.Message != nil && *p.Message != myModel.Message {
				// Update the model
				myModel.Message = *p.Message
				// Send a change event with updated fields
				r.Event("change", p)
			}

			// Send success response
			w.OK(nil)
		}),
	)

	// Start the service.
	err := s.Start("nats://localhost:4222")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
```

**Client**
* *Using Chrome* - Go to this [CodePen](https://codepen.io/sjirenius/pen/vraZPZ).  
* *Using some other browser*  
Some browsers won't allow accessing a non-encrypted websocket from an encrypted page.  
Run the [client javascript](https://github.com/jirenius/resgate#client) locally using a webpack server, or some other similar tool.


