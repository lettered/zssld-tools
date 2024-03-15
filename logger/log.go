package logger

import (
	"io"
	"strings"
	"sync"
)

// Logger the log interface to log program stdout/stderr logs to file
type Logger interface {
	io.WriteCloser
	SetPid(pid int)
	ReadLog(offset int64, length int64) (string, error)
	ReadTailLog(offset int64, length int64) (string, int64, bool, error)
	ClearCurLogFile() error
	ClearAllLogFile() error
}

// LogEventEmitter the interface to emit log events
type LogEventEmitter interface {
	emitLogEvent(data string)
}

// SysLogger log program stdout/stderr to syslog
type SysLogger struct {
	NullLogger
	logWriter       io.WriteCloser
	logEventEmitter LogEventEmitter
}

// NullLocker no lock
type NullLocker struct {
}

// NewNullLocker creates new NullLocker object
func NewNullLocker() *NullLocker {
	return &NullLocker{}
}

// Lock is a stub function for NullLocker
func (l *NullLocker) Lock() {
}

// Unlock is a stub function for NullLocker
func (l *NullLocker) Unlock() {
}

// NullLogEventEmitter will not emit log to any listener
type NullLogEventEmitter struct {
}

// NewNullLogEventEmitter creates new NullLogEventEmitter object
func NewNullLogEventEmitter() *NullLogEventEmitter {
	return &NullLogEventEmitter{}
}

// emitLogEvent emit the log
func (ne *NullLogEventEmitter) emitLogEvent(data string) {
}

// NewLogger creates logger for a program with parameters
func NewLogger(programName string, logFile string, locker sync.Locker, maxBytes int64, backups int, props map[string]string, logEventEmitter LogEventEmitter) Logger {
	files := splitLogFile(logFile)
	loggers := make([]Logger, 0)
	for i, f := range files {
		var lr Logger
		if i == 0 {
			lr = createLogger(programName, f, locker, maxBytes, backups, props, logEventEmitter)
		} else {
			lr = createLogger(programName, f, NewNullLocker(), maxBytes, backups, props, NewNullLogEventEmitter())
		}
		loggers = append(loggers, lr)
	}
	return NewCompositeLogger(loggers)
}

func splitLogFile(logFile string) []string {
	files := strings.Split(logFile, ",")
	for i, f := range files {
		files[i] = strings.TrimSpace(f)
	}
	return files
}

func createLogger(programName string, logFile string, locker sync.Locker, maxBytes int64, backups int, props map[string]string, logEventEmitter LogEventEmitter) Logger {
	if logFile == "/dev/stdout" {
		return NewStdoutLogger(logEventEmitter)
	}
	if logFile == "/dev/stderr" {
		return NewStderrLogger(logEventEmitter)
	}
	if logFile == "/dev/null" {
		return NewNullLogger(logEventEmitter)
	}
	if logFile == "syslog" {
		return NewSysLogger(programName, props, logEventEmitter)
	}

	if len(logFile) > 0 {
		return NewFileLogger(logFile, maxBytes, backups, logEventEmitter, locker)
	}
	return NewNullLogger(logEventEmitter)
}
