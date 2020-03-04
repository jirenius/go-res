<p align="center"><a href="https://resgate.io" target="_blank" rel="noopener noreferrer"><img width="100" src="https://resgate.io/img/resgate-logo.png" alt="Resgate logo"></a></p>
<h2 align="center"><b>Storage utilities for Go RES Service</b><br/>Synchronize Your Clients</h2>
<p align="center">
<a href="../../LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://pkg.go.dev/github.com/jirenius/go-res/store"><img src="https://img.shields.io/static/v1?label=reference&message=go.dev&color=5673ae" alt="Reference"></a>
</p>

---

Package *store* provides handlers and interfaces for working with database storage.

For more details and comments on the interfaces, see the [go.dev reference](https://pkg.go.dev/github.com/jirenius/go-res/store).

## Store interface

A *store* contains resources of a single type. It can be seen as a row in an Excel sheet or SQL table, or a document in a MongoDB collection.

Any database can be used with a wrapper that implements the following interface:

```go
// Store is a CRUD interface for storing resources of a specific type.
type Store interface {
    Read(id string) ReadTxn
    Write(id string) WriteTxn
    OnChange(func(id string, before, after interface{}))
}

// ReadTxn represents a read transaction.
type ReadTxn interface {
    ID() string
    Close() error
    Exists() bool
    Value() (interface{}, error)
}

// WriteTxn represents a write transaction.
type WriteTxn interface {
    ReadTxn
    Create(interface{}) error
    Update(interface{}) error
    Delete() error
}
```

## QueryStore interface

A *query store* provides the methods for making queries to an underlying database, and listen for changes that might affect the results.

```go
// QueryStore is an interface for quering the resource in a store.
type QueryStore interface {
    Query(query url.Values) (interface{}, error)
    OnQueryChange(func(QueryChange))
}

// QueryChange represents a change to a resource that may affects queries.
type QueryChange interface {
    ID() string
    Before() interface{}
    After() interface{}
    Events(q url.Values) (events []ResultEvent, reset bool, err error)
}

// ResultEvent represents an event on a query result.
type ResultEvent struct {
    Name string
    Idx int
    Value interface{}
    Changed map[string]interface{}
}
```

## Implementations

Use these examples as inspiration for your database implementation.

| Name | Description | Documentation
| --- | --- | ---
| [mockstore](mockstore/) | Mock store implementation for testing | [![Reference][godev]](https://pkg.go.dev/github.com/jirenius/go-res/store/mockstore)
| [badgerstore](badgerstore/) | BadgerDB store implementation | [![Reference][godev]](https://pkg.go.dev/github.com/jirenius/go-res/store/badgerstore)

[godev]: https://img.shields.io/static/v1?label=reference&message=go.dev&color=5673ae "Reference"