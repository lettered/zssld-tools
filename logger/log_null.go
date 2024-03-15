package logger

import (
	"errors"
	"fmt"
)

// NullLogger discard the program stdout/stderr log
type NullLogger struct {
	logEventEmitter LogEventEmitter
}

// NewNullLogger creates NullLogger object
func NewNullLogger(logEventEmitter LogEventEmitter) *NullLogger {
	return &NullLogger{logEventEmitter: logEventEmitter}
}

// SetPid sets pid of program
func (l *NullLogger) SetPid(pid int) {
	// NOTHING TO DO
}

// Write log to NullLogger
func (l *NullLogger) Write(p []byte) (int, error) {
	l.logEventEmitter.emitLogEvent(string(p))
	return len(p), nil
}

// Close the NullLogger
func (l *NullLogger) Close() error {
	return nil
}

// ReadLog returns error for NullLogger
func (l *NullLogger) ReadLog(offset int64, length int64) (string, error) {
	return "", errors.New("NO_FILE") //faults.NewFault(faults.NoFile, "NO_FILE")
}

// ReadTailLog returns error for NullLogger
func (l *NullLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return "", 0, false, errors.New("NO_FILE") //faults.NewFault(faults.NoFile, "NO_FILE")
}

// ClearCurLogFile returns error for NullLogger
func (l *NullLogger) ClearCurLogFile() error {
	return fmt.Errorf("No log")
}

// ClearAllLogFile returns error for NullLogger
func (l *NullLogger) ClearAllLogFile() error {
	return errors.New("NO_FILE") //faults.NewFault(faults.NoFile, "NO_FILE")
}
