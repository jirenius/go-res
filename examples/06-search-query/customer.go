package main

import (
	"net/mail"
	"strings"
)

// Customer represents a customer model.
type Customer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Country string `json:"country"`
}

// RID returns the customer's Resource ID.
func (c *Customer) RID() string {
	return "search.customer." + c.ID
}

// TrimAndValidate trims the customer properties from whitespace and validates the values.
// If an error is encountered, a non-empty error message is returned,
// otherwise an empty string.
func (c *Customer) TrimAndValidate() string {
	return customerTrimAndValidate(&c.Name, &c.Email, &c.Country)
}

// customerTrimAndValidate trims any non-nil string pointer and validates the values.
// If an error is encountered, a non-empty error message is returned,
// otherwise an empty string.
func customerTrimAndValidate(name, email, country *string) string {
	if name != nil {
		// Trim and validate name
		*name = strings.TrimSpace(*name)
		if *name == "" {
			return "Name must not be empty."
		}
	}
	if email != nil {
		// Trim and validate email
		*email = strings.TrimSpace(*email)
		if *email != "" {
			if _, err := mail.ParseAddress(*email); err != nil {
				return "Invalid email address."
			}
		}
	}
	if country != nil {
		// Trim and validate country
		*country = strings.TrimSpace(*country)
		if *country != "" && !countries.Contains(*country) {
			return "Country must be empty or one of the following: " + strings.Join(countries, ", ") + "."
		}
	}
	return ""
}
