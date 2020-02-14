package main

import (
	"sort"

	"github.com/jirenius/go-res"
)

// Countries is a sorted list of country names.
var Countries = sort.StringSlice{
	"France",
	"Germany",
	"Sweden",
	"United Kingdom",
}

// CountriesContains searches for a country and returns true if it is found in
// Countries.
func CountriesContains(country string) bool {
	i := sort.SearchStrings(Countries, country)
	return i != len(Countries) && Countries[i] == country
}

// CountriesHandler is a handler that serves the static Countries collection.
var CountriesHandler = res.OptionFunc(func(rh *res.Handler) {
	rh.Get = func(r res.GetRequest) {
		r.Collection(Countries)
	}
})
