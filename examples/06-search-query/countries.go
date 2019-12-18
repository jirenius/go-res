package main

import "github.com/jirenius/go-res"

// Countries is a list of country names.
type Countries []string

var countries = Countries{
	"France",
	"Germany",
	"Sweden",
	"United Kingdom",
}

// Contains returns true if countries contains the country s.
func (cs Countries) Contains(s string) bool {
	for _, c := range cs {
		if c == s {
			return true
		}
	}
	return false
}

type countriesHandler struct{}

func (h countriesHandler) Get(r res.GetRequest) {
	r.Collection(countries)
}

func (h countriesHandler) Access(r res.AccessRequest) {
	r.AccessGranted()
}

// SetOption sets the handler methods to the res.Handler object
func (h countriesHandler) SetOption(hs *res.Handler) {
	hs.Get = h.Get
	hs.Access = h.Access
}
