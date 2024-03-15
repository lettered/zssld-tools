package logger

import (
	"io"
	"os"
)

// StdLogger stdout/stderr logger implementation
type StdLogger struct {
	NullLogger
	logEventEmitter LogEventEmitter
	writer          io.Writer
}

// NewStdoutLogger creates StdLogger object
func NewStdoutLogger(logEventEmitter LogEventEmitter) *StdLogger {
	return &StdLogger{
		logEventEmitter: logEventEmitter,
		writer:          os.Stdout,
	}
}

// Write output to stdout/stderr
func (l *StdLogger) Write(p []byte) (int, error) {
	n, err := l.writer.Write(p)
	if err != nil {
		l.logEventEmitter.emitLogEvent(string(p))
	}
	return n, err
}

// NewStderrLogger creates stderr logger
func NewStderrLogger(logEventEmitter LogEventEmitter) *StdLogger {
	return &StdLogger{
		logEventEmitter: logEventEmitter,
		writer:          os.Stderr,
	}
}
