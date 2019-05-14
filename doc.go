/*
Package res provides RES service implementations for realtime API's through Resgate:

https://github.com/resgateio/resgate

The implementation provides low level methods to listen to and handle incoming
requests, and to send events.

Concurrency

Requests are handled concurrently for multiple resources, but the package
guarantees that only one goroutine is executing handlers for any unique
resource at any one time. This allows handlers to modify models and collections
without additional synchronization such as mutexes.

Usage

Create a new service:

	s := res.NewService("myservice")

Add handlers for a model resource:

	mymodel := map[string]interface{}{"name": "foo", "value": 42}
	s.Handle("mymodel",
		res.Access(res.AccessGranted),
		res.GetModel(func(r res.ModelRequest) {
			r.Model(mymodel)
		}),
	)

Add handlers for a collection resource:

	mycollection := []string{"first", "second", "third"}
	s.Handle("mycollection",
		res.Access(res.AccessGranted),
		res.GetCollection(func(r res.CollectionRequest) {
			r.Collection(mycollection)
		}),
	)

Add handlers for parameterized resources:

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

Add handlers for method calls:

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

Send change event on model update:

	s.With("myservice.mymodel", func(r res.Resource) {
		mymodel["name"] = "bar"
		r.ChangeEvent(map[string]interface{}{"name": "bar"})
	})

Send add event on collection update:

	s.With("myservice.mycollection", func(r res.Resource) {
		mycollection = append(mycollection, "fourth")
		r.AddEvent("fourth", len(mycollection)-1)
	})

Add handlers for authentication:

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

Add handlers for access control:

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

Start service:

	s.ListenAndServe("nats://localhost:4222")
*/
package res
