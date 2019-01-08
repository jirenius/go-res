# RES service package

[![License][License-Image]][License-Url]
[![ReportCard][ReportCard-Image]][ReportCard-Url]
[![Build Status][Build-Status-Image]][Build-Status-Url]
[![Coverage Status][Coverage-Status-Image]][Coverage-Status-Url]
[![GoDoc][GoDoc-Image]][GoDoc-Url]

A [Go](http://golang.org) package implementing the RES-Service protocol for [Resgate - Real-time API Gateway](https://github.com/jirenius/resgate).  
When you want to create stateless REST API services but need to have all your resources updated in real time on your reactive web clients.

All resources and methods served by RES services are made accessible through [Resgate](https://github.com/jirenius/resgate) through both:
* Ordinary HTTP REST requests
* WebSocket using [ResClient](https://www.npmjs.com/package/resclient)

With ResClient, all resources will be updated in real time, without having to write a single line of client code to handle specific events. It just works.

## Installation

```bash
go get github.com/jirenius/go-res
```

## Examples

* [Hello World](examples/hello-world/) - Single model updated in real time
* [Book Collection](examples/book-collection/) - List of books, added, edited, and updated in real time

### As easy as

```go
package main

import res "github.com/jirenius/go-res"

func main() {
    s := res.NewService("myservice")
    s.Handle("mymodel",
        res.Access(res.AccessGranted),
        res.GetModel(func(r res.ModelRequest) {
            r.Model(map[string]string{"greeting": "welcome"})
        }),
    )
    s.ListenAndServe("nats://localhost:4222")
}
```

### Usage

While a RES service communicates over a message broker (NATS Server), instead of listening to HTTP request, the pattern of requests and responses are similar.

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

#### Start service

```go
s.ListenAndServe("nats://localhost:4222")
```

## Credits

Inspiration on the go-res API has been taken from [github.com/go-chi/chi](https://github.com/go-chi/chi), a great package when writing ordinary HTTP services, and will continue to do so when it is time to implement Middleware, sub-handlers, and mounting.

## Contributing

The go-res package is still under development, and commits may still contain breaking changes. It should only be used for educational purpose. Any feedback on the package API or its implementation is highly appreciated!

If you find any issues, feel free to [report them](https://github.com/jirenius/go-res/issues/new) as an Issue.

[GoDoc-Url]: http://godoc.org/github.com/jirenius/go-res
[GoDoc-Image]: https://godoc.org/github.com/jirenius/go-res?status.svg
[License-Url]: http://opensource.org/licenses/MIT
[License-Image]: https://img.shields.io/badge/license-MIT-blue.svg
[ReportCard-Url]: http://goreportcard.com/report/jirenius/go-res
[ReportCard-Image]: http://goreportcard.com/badge/github.com/jirenius/go-res
[Build-Status-Url]: https://travis-ci.com/jirenius/go-res
[Build-Status-Image]: https://travis-ci.com/jirenius/go-res.svg?branch=master
[Coverage-Status-Url]: https://coveralls.io/github/jirenius/go-res?branch=master
[Coverage-Status-Image]: https://coveralls.io/repos/github/jirenius/go-res/badge.svg?branch=master
