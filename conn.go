package res

import nats "github.com/nats-io/go-nats"

// Conn is an interface that represents a connection to a NATS server.
// It is implemented by nats.Conn.
type Conn interface {
	// Publish publishes the data argument to the given subject
	Publish(subject string, payload []byte) error

	// ChanSubscribe subscribes to messages matching the subject pattern.
	ChanSubscribe(subject string, ch chan *nats.Msg) (*nats.Subscription, error)

	// Close will close the connection to the server.
	Close()
}
