/*
Package restest provides utilities for testing res services:

Usage

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
*/
package restest
