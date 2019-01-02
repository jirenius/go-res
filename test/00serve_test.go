package test

import (
	"testing"
)

// Test that the service can be served without error
func TestStart(t *testing.T) {
	runTest(t, nil, nil)
}
