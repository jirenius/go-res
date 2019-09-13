package logger

import (
	"fmt"
	"log"
	"os"
)

// StdLogger writes log entries to os.Stderr
type StdLogger struct {
	log      *log.Logger
	logInfo  bool
	logErr   bool
	logTrace bool
}

// NewStdLogger returns a new logger that writes to os.Stderr
// using the standard log package.
// By default, it will log info and error entries, but not trace entries.
func NewStdLogger() *StdLogger {
	return &StdLogger{
		log:     log.New(os.Stderr, "", log.LstdFlags),
		logErr:  true,
		logInfo: true,
	}
}

// SetFlags sets the output flags for the logger.
func (l *StdLogger) SetFlags(flag int) *StdLogger {
	l.log.SetFlags(flag)
	return l
}

// SetInfo sets whether info entries should be logged.
func (l *StdLogger) SetInfo(logInfo bool) *StdLogger {
	l.logInfo = logInfo
	return l
}

// SetErr sets whether error entries should be logged.
func (l *StdLogger) SetErr(logErr bool) *StdLogger {
	l.logErr = logErr
	return l
}

// SetTrace sets whether trace entries should be logged.
func (l *StdLogger) SetTrace(logTrace bool) *StdLogger {
	l.logTrace = logTrace
	return l
}

// Infof writes an info log entry.
func (l *StdLogger) Infof(format string, v ...interface{}) {
	if l.logInfo {
		l.log.Print("[INF] ", fmt.Sprintf(format, v...))
	}
}

// Errorf writes an error log entry.
func (l *StdLogger) Errorf(format string, v ...interface{}) {
	if l.logErr {
		l.log.Print("[ERR] ", fmt.Sprintf(format, v...))
	}
}

// Tracef writes a trace log entry.
func (l *StdLogger) Tracef(format string, v ...interface{}) {
	if l.logTrace {
		l.log.Print("[TRA] ", fmt.Sprintf(format, v...))
	}
}
