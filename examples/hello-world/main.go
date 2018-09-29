/*
This is an example of a simple Hello World RES service written in Go.
* It exposes a single resource: "exampleService.myModel".
* It allows setting the resource's Message property through the "set" method.

Visit https://github.com/jirenius/resgate#client for the matching client.
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/jirenius/go-res"
)

// Model example
type Model struct {
	Message string `json:"message"`
}

var myModel = &Model{Message: "Hello Go World"}

func main() {
	// Create a new RES Service
	s := res.NewService("exampleService")

	// Add handlers for "exampleService.myModel" resource
	s.Handle("myModel",
		res.Access(res.AccessGranted),
		res.GetModel(func(w res.GetModelResponse, r *res.Request) {
			w.Model(myModel)
		}),
		res.Call("set", func(w res.CallResponse, r *res.Request) {
			var p struct {
				Message *string `json:"message,omitempty"`
			}
			r.UnmarshalParams(&p)

			// Check if the message property was changed
			if p.Message != nil && *p.Message != myModel.Message {
				// Update the model
				myModel.Message = *p.Message
				// Send a change event with updated fields
				r.ChangeEvent(map[string]interface{}{"message": p.Message})
			}

			// Send success response
			w.OK(nil)
		}),
	)

	// Start service in separate goroutine
	stop := make(chan bool)
	go func() {
		defer close(stop)
		if err := s.ListenAndServe("nats://localhost:4222"); err != nil {
			fmt.Printf("%s\n", err.Error())
		}
	}()

	// Run a simple webserver to serve the client.
	// This is only for the purpose of making the example easier to run.
	go func() { log.Fatal(http.ListenAndServe(":8081", http.FileServer(http.Dir("./")))) }()
	fmt.Println("Client at: http://localhost:8081/")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
		// Graceful stop
		s.Shutdown()
	case <-stop:
	}
}
