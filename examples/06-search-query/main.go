package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dgraph-io/badger"

	"github.com/jirenius/go-res"
)

func main() {
	// // Create badger DB
	db, err := badger.Open(badger.DefaultOptions("./db").WithTruncate(true))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a new RES Service
	s := res.NewService("search")

	// Add handlers
	s.Handle("countries", countriesHandler{})
	s.Handle("customer.$id", customerHandler{DB: db})
	s.Handle("customers", customerQueryHandler{DB: db})

	// Run a simple webserver to serve the client.
	// This is only for the purpose of making the example easier to run.
	go func() { log.Fatal(http.ListenAndServe(":8086", http.FileServer(http.Dir("wwwroot/")))) }()
	fmt.Println("Client at: http://localhost:8086/")

	s.ListenAndServe("nats://localhost:4222")
}
