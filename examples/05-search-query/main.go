/*
A customer management system, where you can search and filter customers by name
and country. The search results are live and updated as customers are edited,
added, or deleted by multiple users simultaneously.
*/
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dgraph-io/badger"

	"github.com/jirenius/go-res"
)

func main() {
	// Create badger DB
	db, err := badger.Open(badger.DefaultOptions("./db").WithTruncate(true))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create badgerDB store for customers
	customerStore := NewCustomerStore(db)
	// Seed database with initial customers, if not done before
	customerStore.Init()

	// Create a new RES Service
	s := res.NewService("search")

	// Add handlers
	s.Handle("countries",
		CountriesHandler,
		res.Access(res.AccessGranted))
	s.Handle("customer.$id",
		&CustomerHandler{CustomerStore: customerStore},
		res.Access(res.AccessGranted))
	s.Handle("customers",
		&CustomerQueryHandler{CustomerStore: customerStore},
		res.Access(res.AccessGranted))

	// Run a simple webserver to serve the client.
	// This is only for the purpose of making the example easier to run.
	go func() { log.Fatal(http.ListenAndServe(":8085", http.FileServer(http.Dir("wwwroot/")))) }()
	fmt.Println("Client at: http://localhost:8085/")

	s.ListenAndServe("nats://localhost:4222")
}
