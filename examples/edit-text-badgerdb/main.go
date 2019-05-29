/*
This is the edit-text example with persistance to BadgerDB.
 * It exposes a single resource: "example.shared".
 * It allows setting the resource's Message property through the "set" method.
 * It persist all changes to BadgerDB.
 * It serves a web client at http://localhost:8082
*/
package main

import (
	"log"
	"net/http"

	"github.com/dgraph-io/badger"
	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/middleware"
)

func main() {
	// Create badger DB
	opts := badger.DefaultOptions
	opts.Dir = "./db"
	opts.ValueDir = "./db"
	opts.Truncate = true
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := res.NewService("example")
	s.Handle("shared",
		// Define resource type.
		res.Model,
		// Allow everone to access this resource
		res.Access(res.AccessGranted),
		// BadgerDB middleware adds a GetResource and ApplyChange handler
		middleware.BadgerDB{DB: db, Default: map[string]interface{}{"message": "Resgate loves BadgerDB"}},

		// Handle setting of the message
		res.Set(func(r res.CallRequest) {
			// Get current resource value from BadgerDB
			m := r.RequireValue().(map[string]interface{})

			var p struct {
				Message *string `json:"message,omitempty"`
			}
			r.ParseParams(&p)

			// Check if the message property was changed
			if p.Message != nil && *p.Message != m["message"] {
				// Send a change event with updated fields
				// BadgerDB middleware will use the event to updated the stored model
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

	s.ListenAndServe("nats://localhost:4222")
}
