<p align="center"><a href="https://resgate.io" target="_blank" rel="noopener noreferrer"><img width="100" src="https://resgate.io/img/resgate-logo.png" alt="Resgate logo"></a></p>
<h2 align="center"><b>Testing for Go RES Service</b><br/>Synchronize Your Clients</h2>
<p align="center">
<a href="../../LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://pkg.go.dev/github.com/jirenius/go-res/restest"><img src="https://img.shields.io/static/v1?label=reference&message=go.dev&color=5673ae" alt="Reference"></a>
</p>

---

Package *restest* provides utilities for testing res services.

## Basic usage

```go
func TestService(t *testing.T) {
    // Create service to test
    s := res.NewService("foo")
    s.Handle("bar.$id",
        res.Access(res.AccessGranted),
        res.GetModel(func(r res.ModelRequest) {
            r.Model(struct {
                Message string `json:"msg"`
            }{r.PathParam("id")})
        }),
    )

    // Create test session
    c := restest.NewSession(t, s)
    defer c.Close()

    // Test sending get request and validate response
    c.Get("foo.bar.42").
        Response().
        AssertModel(map[string]string{"msg": "42"})
}
```
