package main

import (
	"net/mail"
	"strings"

	"github.com/jirenius/go-res"
)

// Customer represents a customer model.
type Customer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Country string `json:"country"`
}

// TrimAndValidate trims the customer properties from whitespace and validates
// the values. If an error is encountered, a res.CodeInvalidParams error is
// returned.
func (c *Customer) TrimAndValidate() error {
	// Trim and validate name
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return &res.Error{Code: res.CodeInvalidParams, Message: "Name must not be empty."}
	}
	// Trim and validate email
	c.Email = strings.TrimSpace(c.Email)
	if c.Email != "" {
		if _, err := mail.ParseAddress(c.Email); err != nil {
			return &res.Error{Code: res.CodeInvalidParams, Message: "Invalid email address."}
		}
	}
	// Trim and validate country
	c.Country = strings.TrimSpace(c.Country)
	if c.Country != "" && !CountriesContains(c.Country) {
		return &res.Error{Code: res.CodeInvalidParams, Message: "Country must be empty or one of the following: " + strings.Join(Countries, ", ") + "."}
	}

	return nil
}
