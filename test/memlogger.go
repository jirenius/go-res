package test

import (
	"bytes"
	"fmt"
	"log"
	"sync"
)

// MemLogger writes log messages to os.Stderr
type MemLogger struct {
	log   *log.Logger
	b     *bytes.Buffer
	debug bool
	trace bool
	mu    sync.Mutex
}

// newMemLogger returns a new logger that writes to a bytes buffer
func newMemLogger(debug bool, trace bool) *MemLogger {
	logFlags := log.LstdFlags
	if debug {
		logFlags = log.Ltime
	}

	b := &bytes.Buffer{}

	return &MemLogger{
		log:   log.New(b, "", logFlags),
		b:     b,
		debug: debug,
		trace: trace,
	}
}

// Logf writes a log entry
func (l *MemLogger) Logf(prefix string, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.log.Print(prefix, fmt.Sprintf(format, v...))
}

// Debugf writes a debug entry
func (l *MemLogger) Debugf(prefix string, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.debug {
		l.log.Print(prefix, fmt.Sprintf(format, v...))
	}
}

// Tracef writes a trace entry
func (l *MemLogger) Tracef(prefix string, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.trace {
		l.log.Print(prefix, fmt.Sprintf(format, v...))
	}
}

// String returns the log
func (l *MemLogger) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}
