package logger

// Logger is used to write log entries.
// The interface is meant to be simple to wrap for other logger implementations.
type Logger interface {
	// Infof logs service state, such as connects and reconnects to NATS.
	Infof(format string, v ...interface{})

	// Errorf logs errors in the service, or incoming messages not complying
	// with the RES protocol.
	Errorf(format string, v ...interface{})

	// Tracef all network traffic going to and from the service.
	Tracef(format string, v ...interface{})
}
