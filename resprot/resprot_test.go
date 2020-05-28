package resprot_test

import (
	"testing"
	"time"

	"github.com/jirenius/go-res/resprot"
	"github.com/nats-io/nats.go"
)

func TestUsage_SendRequest(t *testing.T) {
	conn, _ := nats.Connect("nats://127.0.0.1:4222")
	response := resprot.SendRequest(conn, "call.example.ping", nil, time.Second)

	// ------------------

	if !response.HasError() {
		t.Errorf("expected error, but found none")
	}
}

func TestUsage_SendModelRequest(t *testing.T) {
	conn, _ := nats.Connect("invalid")

	// ---

	response := resprot.SendRequest(conn, "get.example.model", nil, time.Second)

	var model struct {
		Message string `json:"message"`
	}
	_, err := response.ParseModel(&model)

	// ------------------

	if err == nil {
		t.Errorf("expected error, but found none")
	}
}

func TestUsage_SendCallRequest(t *testing.T) {
	conn, _ := nats.Connect("invalid")

	// ---

	response := resprot.SendRequest(conn, "call.math.add", resprot.Request{Params: struct {
		A float64 `json:"a"`
		B float64 `json:"a"`
	}{5, 6}}, time.Second)

	var result struct {
		Sum float64 `json:"sum"`
	}
	err := response.ParseResult(&result)

	// ---

	if err == nil {
		t.Errorf("expected error, but found none")
	}
}
