<p align="center"><a href="https://resgate.io" target="_blank" rel="noopener noreferrer"><img width="100" src="https://resgate.io/img/resgate-logo.png" alt="Resgate logo"></a></p>
<h2 align="center"><b>RES Service for Go</b><br/>Synchronize Your Clients</h2>
<p align="center">
<a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="http://goreportcard.com/report/jirenius/go-res"><img src="http://goreportcard.com/badge/github.com/jirenius/go-res" alt="Report Card"></a>
<a href="https://travis-ci.com/jirenius/go-res"><img src="https://travis-ci.com/jirenius/go-res.svg?branch=master" alt="Build Status"></a>
<a href="https://coveralls.io/github/jirenius/go-res?branch=master"><img src="https://coveralls.io/repos/github/jirenius/go-res/badge.svg?branch=master" alt="Coverage"></a>
<a href="https://pkg.go.dev/github.com/jirenius/go-res"><img src="https://img.shields.io/static/v1?label=reference&message=go.dev&color=5673ae" alt="Reference"></a>
</p>

---

[Go](http://golang.org) package used to create REST, real time, and RPC APIs, where all your reactive web clients are synchronized seamlessly through [Resgate](https://github.com/resgateio/resgate).

Visit [Resgate.io](https://resgate.io) for more information.

## Installation

```bash
go get github.com/jirenius/go-res
```

## As easy as

```go
package main

import res "github.com/jirenius/go-res"

func main() {
   s := res.NewService("example")
   s.Handle("model",
      res.Access(res.AccessGranted),
      res.GetModel(func(r res.ModelRequest) {
         r.Model(map[string]string{
            "message": "Hello, World!",
         })
      }),
   )
   s.ListenAndServe("nats://localhost:4222")
}
```

## Examples

| Example | Description
| --- | ---
| [Hello World](examples/01-hello-world/) | Smallest of services serving a static message.
| [Edit Text](examples/02-edit-text/) | Single text field that is updated in real time.
| [Edit Text Persisted](examples/03-edit-text-persisted/) | Edit Text example persisting changes using BadgerDB middleware.
| [Book Collection](examples/04-book-collection/) | List of book titles & authors that can be edited by many.
| [Book Collection Persisted](examples/05-book-collection-persisted/) | Book Collection example persisting changes using BadgerBD middleware.
| [Search Query](examples/06-search-query/) | Make live queries against a large customer database.

> **Note**
>
> Above examples are complete with both service and client.

## Middleware

The *middleware* subfolder contains packages that adds handler functions to a `res.Handler`, to perform tasks such as:

* store, load and update persisted data
* synchronize changes between multiple service instances
* perform live queries with indexed searches

| Name | Description | Documentation
| --- | --- | ---
| [resbadger](middleware/resbadger) | BadgerDB storage middleware. | <a href="https://pkg.go.dev/github.com/jirenius/go-res/middleware/resbadger"><img src="https://img.shields.io/static/v1?label=reference&message=go.dev&color=5673ae" alt="Reference"></a>

## Usage

#### Create a new service

```go
s := res.NewService("myservice")
```

#### Add handlers for a model resource

```go
mymodel := map[string]interface{}{"name": "foo", "value": 42}
s.Handle("mymodel",
   res.Access(res.AccessGranted),
   res.GetModel(func(r res.ModelRequest) {
      r.Model(mymodel)
   }),
)
```

#### Add handlers for a collection resource

```go
mycollection := []string{"first", "second", "third"}
s.Handle("mycollection",
   res.Access(res.AccessGranted),
   res.GetCollection(func(r res.CollectionRequest) {
      r.Collection(mycollection)
   }),
)
```

#### Add handlers for parameterized resources

```go
s.Handle("article.$id",
   res.Access(res.AccessGranted),
   res.GetModel(func(r res.ModelRequest) {
      article := getArticle(r.PathParam("id"))
      if article == nil {
         r.NotFound()
      } else {
         r.Model(article)
      }
   }),
)
```

#### Add handlers for method calls

```go
s.Handle("math",
   res.Access(res.AccessGranted),
   res.Call("double", func(r res.CallRequest) {
      var p struct {
         Value int `json:"value"`
      }
      r.ParseParams(&p)
      r.OK(p.Value * 2)
   }),
)
```

#### Send change event on model update
A change event will update the model on all subscribing clients.

```go
s.With("myservice.mymodel", func(r res.Resource) {
   mymodel["name"] = "bar"
   r.ChangeEvent(map[string]interface{}{"name": "bar"})
})
```

#### Send add event on collection update:
An add event will update the collection on all subscribing clients.

```go
s.With("myservice.mycollection", func(r res.Resource) {
   mycollection = append(mycollection, "fourth")
   r.AddEvent("fourth", len(mycollection)-1)
})
```

#### Add handlers for authentication

```go
s.Handle("myauth",
   res.Auth("login", func(r res.AuthRequest) {
      var p struct {
         Password string `json:"password"`
      }
      r.ParseParams(&p)
      if p.Password != "mysecret" {
         r.InvalidParams("Wrong password")
      } else {
         r.TokenEvent(map[string]string{"user": "admin"})
         r.OK(nil)
      }
   }),
)
```

#### Add handlers for access control

```go
s.Handle("mymodel",
   res.Access(func(r res.AccessRequest) {
      var t struct {
         User string `json:"user"`
      }
      r.ParseToken(&t)
      if t.User == "admin" {
         r.AccessGranted()
      } else {
         r.AccessDenied()
      }
   }),
   res.GetModel(func(r res.ModelRequest) {
      r.Model(mymodel)
   }),
)
```

#### Using routes

```go
s.Route("v2", func(m *res.Mux) {
   m.Handle("mymodel",
      /* ... */
   )
})
```

#### Start service

```go
s.ListenAndServe("nats://localhost:4222")
```

## Credits

Inspiration on the go-res API has been taken from [github.com/go-chi/chi](https://github.com/go-chi/chi), a great package when writing ordinary HTTP services, and will continue to do so when it is time to implement Middleware, sub-handlers, and mounting.

## Contributing

The go-res package is still under development, but the API is mostly settled. Any feedback on the package API or its implementation is highly appreciated!

Once the API is fully settled, the package will be moved to the [resgateio](https://github.com/resgateio/) GitHub organization.

If you find any issues, feel free to [report them](https://github.com/jirenius/go-res/issues/new) as an issue.
