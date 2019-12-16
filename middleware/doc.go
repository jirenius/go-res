/*
Package middleware provides middleware for the res package:

https://github.com/jirenius/go-res

Middleware can be used for adding handler functions to a res.Handler,
to perform tasks such as:

* storing, loading and updating persisted data
* synchronize changes between multiple service instances
* add additional logging
* provide helpers for complex live queries

Currently, only the BadgerDB middleware is created, to demonstrate
database persistence.

Usage

Add middleware to a resource:

	s.Handle("user.$id",
		middlware.BadgerDB{DB: db},
	)

*/
package middleware
