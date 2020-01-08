/*
This is an example of a simple text field that can be edited by multiple clients.
* It exposes a single resource: "text.shared".
* It allows setting the resource's Message property through the "set" method.
* It resets the model on server restart.
* It serves a web client at http://localhost:8082
*/
package main

import (
	"log"
	"net/http"

	"github.com/jirenius/go-res"
)

// Model is the structure for our model resource
type Model struct {
	Message string `json:"message"`
}

// The model we will serve
var shared = &Model{Message: "Hello, Go World!"}

func main() {
	// Create a new RES Service
	s := res.NewService("text")

	// Add handlers for "text.shared" resource
	s.Handle("shared",
		// Allow everone to access this resource
		res.Access(res.AccessGranted),

		// Respond to get requests with the model
		res.GetModel(func(r res.ModelRequest) {
			r.Model(shared)
		}),

		// Handle setting of the message
		res.Set(func(r res.CallRequest) {
			var p struct {
				Message *string `json:"message,omitempty"`
			}
			r.ParseParams(&p)

			// Check if the message property was changed
			if p.Message != nil && *p.Message != shared.Message {
				// Update the model
				shared.Message = *p.Message
				// Send a change event with updated fields
				r.ChangeEvent(map[string]interface{}{"message": p.Message})
			}

			// Send success response
			r.OK(nil)
		}),
	)

	// Run a simple webserver to serve the client.
	// This is only for the purpose of making the example easier to run.
	go func() { log.Fatal(http.ListenAndServe(":8082", http.FileServer(http.Dir("./")))) }()
	log.Println("Client at: http://localhost:8082/")

	// Start the service
	s.ListenAndServe("nats://localhost:4222")
}
