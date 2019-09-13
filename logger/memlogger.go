package logger

import (
	"bytes"
	"fmt"
	"log"
	"sync"
)

// MemLogger writes log messages to a bytes buffer.
type MemLogger struct {
	log      *log.Logger
	b        *bytes.Buffer
	logInfo  bool
	logErr   bool
	logTrace bool
	mu       sync.Mutex
}

// NewMemLogger returns a new logger that writes to a bytes buffer
func NewMemLogger() *MemLogger {
	b := &bytes.Buffer{}

	return &MemLogger{
		log:     log.New(b, "", log.LstdFlags),
		b:       b,
		logErr:  true,
		logInfo: true,
	}
}

// SetFlags sets the output flags for the logger.
func (l *MemLogger) SetFlags(flag int) *MemLogger {
	l.log.SetFlags(flag)
	return l
}

// SetInfo sets whether info entries should be logged.
func (l *MemLogger) SetInfo(logInfo bool) *MemLogger {
	l.logInfo = logInfo
	return l
}

// SetErr sets whether error entries should be logged.
func (l *MemLogger) SetErr(logErr bool) *MemLogger {
	l.logErr = logErr
	return l
}

// SetTrace sets whether trace entries should be logged.
func (l *MemLogger) SetTrace(logTrace bool) *MemLogger {
	l.logTrace = logTrace
	return l
}

// Infof writes an info log entry.
func (l *MemLogger) Infof(format string, v ...interface{}) {
	if l.logInfo {
		l.mu.Lock()
		l.log.Print("[INF] ", fmt.Sprintf(format, v...))
		l.mu.Unlock()
	}
}

// Errorf writes an error log entry.
func (l *MemLogger) Errorf(format string, v ...interface{}) {
	if l.logErr {
		l.mu.Lock()
		l.log.Print("[ERR] ", fmt.Sprintf(format, v...))
		l.mu.Unlock()
	}
}

// Tracef writes a trace log entry.
func (l *MemLogger) Tracef(format string, v ...interface{}) {
	if l.logTrace {
		l.mu.Lock()
		l.log.Print("[TRA] ", fmt.Sprintf(format, v...))
		l.mu.Unlock()
	}
}

// String returns the log
func (l *MemLogger) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}
