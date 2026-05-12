package res

import (
	"testing"

	nats "github.com/nats-io/nats.go"
)

func TestDebugPendingMessages_ReturnsLimitedTruncatedCopies(t *testing.T) {
	s := NewService("test")
	s.pending = []*nats.Msg{
		{Subject: "get.test.model", Reply: "reply.1", Data: []byte(`{"foo":"bar"}`)},
		{Subject: "call.test.model.set", Reply: "reply.2", Data: []byte(`{"baz":42}`)},
	}

	msgs := s.DebugPendingMessages(1, 5)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Subject != "get.test.model" {
		t.Fatalf("expected subject get.test.model, got %s", msgs[0].Subject)
	}
	if msgs[0].Reply != "reply.1" {
		t.Fatalf("expected reply reply.1, got %s", msgs[0].Reply)
	}
	if msgs[0].Bytes != len(`{"foo":"bar"}`) {
		t.Fatalf("expected byte count %d, got %d", len(`{"foo":"bar"}`), msgs[0].Bytes)
	}
	if string(msgs[0].Data) != `{"foo` {
		t.Fatalf("expected truncated data, got %q", msgs[0].Data)
	}
	if !msgs[0].DataTruncated {
		t.Fatal("expected DataTruncated to be true")
	}

	s.pending[0].Data[0] = '['
	if string(msgs[0].Data) != `{"foo` {
		t.Fatalf("expected copied data to remain unchanged, got %q", msgs[0].Data)
	}
}

func TestDebugPendingMessages_PayloadOptions(t *testing.T) {
	s := NewService("test")
	s.pending = []*nats.Msg{
		{Subject: "get.test.model", Data: []byte(`{"foo":"bar"}`)},
	}

	withoutPayload := s.DebugPendingMessages(0, 0)
	if len(withoutPayload) != 1 {
		t.Fatalf("expected 1 message, got %d", len(withoutPayload))
	}
	if withoutPayload[0].Data != nil {
		t.Fatalf("expected omitted data, got %q", withoutPayload[0].Data)
	}
	if withoutPayload[0].Bytes != len(`{"foo":"bar"}`) {
		t.Fatalf("expected byte count %d, got %d", len(`{"foo":"bar"}`), withoutPayload[0].Bytes)
	}

	fullPayload := s.DebugPendingMessages(0, -1)
	if string(fullPayload[0].Data) != `{"foo":"bar"}` {
		t.Fatalf("expected full data, got %q", fullPayload[0].Data)
	}
	if fullPayload[0].DataTruncated {
		t.Fatal("expected DataTruncated to be false")
	}
}
