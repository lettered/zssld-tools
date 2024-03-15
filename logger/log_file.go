package logger

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// FileLogger log program stdout/stderr to file
type FileLogger struct {
	name            string
	maxSize         int64
	backups         int
	fileSize        int64
	file            *os.File
	logEventEmitter LogEventEmitter
	locker          sync.Locker
}

// NewFileLogger creates FileLogger object
func NewFileLogger(name string, maxSize int64, backups int, logEventEmitter LogEventEmitter, locker sync.Locker) *FileLogger {
	logger := &FileLogger{name: name,
		maxSize:         maxSize,
		backups:         backups,
		fileSize:        0,
		file:            nil,
		logEventEmitter: logEventEmitter,
		locker:          locker}
	logger.openFile(false)
	return logger
}

// SetPid sets pid of the program
func (l *FileLogger) SetPid(pid int) {
	// NOTHING TO DO
}

// open the file and truncate the file if trunc is true
func (l *FileLogger) openFile(trunc bool) error {
	if l.file != nil {
		l.file.Close()
	}
	var err error
	fileInfo, err := os.Stat(l.name)

	if trunc || err != nil {
		l.file, err = os.Create(l.name)
	} else {
		l.fileSize = fileInfo.Size()
		l.file, err = os.OpenFile(l.name, os.O_RDWR|os.O_APPEND, 0666)
	}
	if err != nil {
		fmt.Printf("Fail to open log file --%s-- with error %v\n", l.name, err)
	}
	return err
}

func (l *FileLogger) backupFiles() {
	for i := l.backups - 1; i > 0; i-- {
		src := fmt.Sprintf("%s.%d", l.name, i)
		dest := fmt.Sprintf("%s.%d", l.name, i+1)
		if _, err := os.Stat(src); err == nil {
			os.Rename(src, dest)
		}
	}
	dest := fmt.Sprintf("%s.1", l.name)
	os.Rename(l.name, dest)
}

// ClearCurLogFile clears contents (re-open with truncate) of current log file
func (l *FileLogger) ClearCurLogFile() error {
	l.locker.Lock()
	defer l.locker.Unlock()

	return l.openFile(true)
}

// ClearAllLogFile clears contents of all log files (re-open with truncate)
func (l *FileLogger) ClearAllLogFile() error {
	l.locker.Lock()
	defer l.locker.Unlock()

	for i := l.backups; i > 0; i-- {
		logFile := fmt.Sprintf("%s.%d", l.name, i)
		_, err := os.Stat(logFile)
		if err == nil {
			err = os.Remove(logFile)
			if err != nil {
				return err //faults.NewFault(faults.Failed, err.Error())
			}
		}
	}
	err := l.openFile(true)
	if err != nil {
		return err //faults.NewFault(faults.Failed, err.Error())
	}
	return nil
}

// ReadLog reads log from current logfile
func (l *FileLogger) ReadLog(offset int64, length int64) (string, error) {
	if offset < 0 && length != 0 {
		return "", errors.New("BAD_ARGUMENTS") //faults.NewFault(faults.BadArguments, "BAD_ARGUMENTS")
	}
	if offset >= 0 && length < 0 {
		return "", errors.New("BAD_ARGUMENTS") //faults.NewFault(faults.BadArguments, "BAD_ARGUMENTS")
	}

	l.locker.Lock()
	defer l.locker.Unlock()
	f, err := os.Open(l.name)

	if err != nil {
		return "", errors.New("FAILED") //faults.NewFault(faults.Failed, "FAILED")
	}
	defer f.Close()

	// check the length of file
	statInfo, err := f.Stat()
	if err != nil {
		return "", errors.New("FAILED") //faults.NewFault(faults.Failed, "FAILED")
	}

	fileLen := statInfo.Size()

	if offset < 0 { // offset < 0 && length == 0
		offset = fileLen + offset
		if offset < 0 {
			offset = 0
		}
		length = fileLen - offset
	} else if length == 0 { // offset >= 0 && length == 0
		if offset > fileLen {
			return "", nil
		}
		length = fileLen - offset
	} else { // offset >= 0 && length > 0

		// if the offset exceeds the length of file
		if offset >= fileLen {
			return "", nil
		}

		// compute actual bytes should be read

		if offset+length > fileLen {
			length = fileLen - offset
		}
	}

	b := make([]byte, length)
	n, err := f.ReadAt(b, offset)
	if err != nil {
		return "", errors.New("FAILED") //faults.NewFault(faults.Failed, "FAILED")
	}
	return string(b[:n]), nil
}

// ReadTailLog tails current log file
func (l *FileLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	if offset < 0 {
		return "", offset, false, fmt.Errorf("offset should not be less than 0")
	}
	if length < 0 {
		return "", offset, false, fmt.Errorf("length should be not be less than 0")
	}
	l.locker.Lock()
	defer l.locker.Unlock()

	// open the file
	f, err := os.Open(l.name)
	if err != nil {
		return "", 0, false, err
	}

	defer f.Close()

	// get the length of file
	statInfo, err := f.Stat()
	if err != nil {
		return "", 0, false, err
	}

	fileLen := statInfo.Size()

	// check if offset exceeds the length of file
	if offset >= fileLen {
		return "", fileLen, true, nil
	}

	// get the length
	if offset+length > fileLen {
		length = fileLen - offset
	}

	b := make([]byte, length)
	n, err := f.ReadAt(b, offset)
	if err != nil {
		return "", offset, false, err
	}
	return string(b[:n]), offset + int64(n), false, nil

}

// Write overrides function in io.Writer. Write log message to the file
func (l *FileLogger) Write(p []byte) (int, error) {
	l.locker.Lock()
	defer l.locker.Unlock()

	n, err := l.file.Write(p)

	if err != nil {
		return n, err
	}
	l.logEventEmitter.emitLogEvent(string(p))
	l.fileSize += int64(n)
	if l.fileSize >= l.maxSize {
		fileInfo, errStat := os.Stat(l.name)
		if errStat == nil {
			l.fileSize = fileInfo.Size()
		} else {
			return n, errStat
		}
	}
	if l.fileSize >= l.maxSize {
		l.Close()
		l.backupFiles()
		l.openFile(true)
	}
	return n, err
}

// Close file logger
func (l *FileLogger) Close() error {
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}
