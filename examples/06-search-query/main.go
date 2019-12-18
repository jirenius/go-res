package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dgraph-io/badger"
	"github.com/jirenius/go-res/logger"

	"github.com/jirenius/go-res"
)

func printDB(db *badger.DB) {
	db.Update(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			// Load item and unmarshal it
			item := it.Item()
			err := item.Value(func(value []byte) error {
				fmt.Printf("[%s] %s\n", string(item.Key()), string(value))
				return nil
			})
			if err != nil {
				panic(err)
			}
		}
		return nil
	})
}

func main() {
	// // Create badger DB
	db, err := badger.Open(badger.DefaultOptions("./db").WithTruncate(true))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	printDB(db)
	// Create a new RES Service
	s := res.NewService("search")

	s.SetLogger(logger.NewStdLogger().SetTrace(true))

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
