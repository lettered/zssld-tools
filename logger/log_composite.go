package logger

import "sync"

// CompositeLogger dispatch the log message to other loggers
type CompositeLogger struct {
	lock    sync.Mutex
	loggers []Logger
}

// NewCompositeLogger creates new CompositeLogger object (pool of loggers)
func NewCompositeLogger(loggers []Logger) *CompositeLogger {
	return &CompositeLogger{loggers: loggers}
}

// AddLogger adds logger to CompositeLogger pool
func (cl *CompositeLogger) AddLogger(logger Logger) {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	cl.loggers = append(cl.loggers, logger)
}

// RemoveLogger removes logger from CompositeLogger pool
func (cl *CompositeLogger) RemoveLogger(logger Logger) {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	for i, t := range cl.loggers {
		if t == logger {
			cl.loggers = append(cl.loggers[:i], cl.loggers[i+1:]...)
			break
		}
	}
}

// Write dispatches log data to the loggers in CompositeLogger pool
func (cl *CompositeLogger) Write(p []byte) (n int, err error) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	for i, logger := range cl.loggers {
		if i == 0 {
			n, err = logger.Write(p)
		} else {
			logger.Write(p)
		}
	}
	return
}

// Close all loggers in CompositeLogger pool
func (cl *CompositeLogger) Close() (err error) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	for i, logger := range cl.loggers {
		if i == 0 {
			err = logger.Close()
		} else {
			logger.Close()
		}
	}
	return
}

// SetPid sets pid to all loggers in CompositeLogger pool
func (cl *CompositeLogger) SetPid(pid int) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	for _, logger := range cl.loggers {
		logger.SetPid(pid)
	}
}

// ReadLog read log data from first logger in CompositeLogger pool
func (cl *CompositeLogger) ReadLog(offset int64, length int64) (string, error) {
	return cl.loggers[0].ReadLog(offset, length)
}

// ReadTailLog tail the log data from first logger in CompositeLogger pool
func (cl *CompositeLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return cl.loggers[0].ReadTailLog(offset, length)
}

// ClearCurLogFile clear the first logger file in CompositeLogger pool
func (cl *CompositeLogger) ClearCurLogFile() error {
	return cl.loggers[0].ClearCurLogFile()
}

// ClearAllLogFile clear all the files of first logger in CompositeLogger pool
func (cl *CompositeLogger) ClearAllLogFile() error {
	return cl.loggers[0].ClearAllLogFile()
}
