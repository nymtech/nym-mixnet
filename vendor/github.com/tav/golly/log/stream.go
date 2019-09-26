// Public Domain (-) 2010-2014 The Golly Authors.
// See the Golly UNLICENSE file for details.

package log

import (
	"bufio"
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

var (
	errAlreadyClosed    = errors.New("log: stream already closed")
	errMissingFormatter = errors.New("log: stream options need to have the Formatter field set")
	errMissingStream    = errors.New("log: stream options need to have either the Filename or the Stream fields set")
)

type Options struct {
	BufferSize    int
	Filename      string
	Filter        func(*Entry) bool
	Formatter     Formatter
	FlushInterval time.Duration
	LogType       LogType
	Rotate        *RotateOptions
	Stream        io.WriteCloser
}

type RotateOptions struct {
	Filename   string
	LocalTime  bool
	MaxAge     time.Duration
	MaxBackups int
	MaxSize    int
	Signal     os.Signal
}

func StreamHandler(o *Options) (Handler, error) {
	s := &stream{}
	var (
		err error
	)
	if o.Filename != "" {
		// Rotate existing..
		s.file, err = os.OpenFile(o.Filename, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return nil, err
		}
		s.wc = s.file
	} else if o.Stream != nil {
		s.wc = o.Stream
	} else {
		return nil, errMissingStream
	}
	if o.Formatter == nil {
		return nil, errMissingFormatter
	}
	s.f = o.Formatter
	if o.BufferSize > 0 {
		s.buf = bufio.NewWriterSize(s.wc, o.BufferSize)
		if o.FlushInterval > 0 {
			go s.flush(o.FlushInterval)
		} else {
			go s.flush(time.Second)
		}
	}
	if o.LogType&ErrorLog != 0 {
		s.logErr = true
	}
	if o.LogType&InfoLog != 0 {
		s.logInfo = true
	}
	// process.RegisterSignalHandler(syscall.SIGHUP, file.Rotate)
	if s.file != nil && o.Rotate != nil {
		go s.rotate(o)
	}
	return s, nil
}

type stream struct {
	buf     *bufio.Writer
	closed  bool
	f       Formatter
	file    *os.File
	filter  func(*Entry) bool
	logInfo bool
	logErr  bool
	maxSize int64
	mu      sync.Mutex
	wc      io.WriteCloser
	written int64
}

func (s *stream) Async() bool {
	return false
}

func (s *stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		return s.wc.Close()
	}
	return errAlreadyClosed
}

func (s *stream) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errAlreadyClosed
	}
	if s.buf != nil {
		err := s.buf.Flush()
		if err != nil {
			return err
		}
	}
	if s.file != nil {
		err := s.file.Sync()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *stream) Log(e *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errAlreadyClosed
	}
	if e.Error {
		if !s.logErr {
			return nil
		}
	} else if !s.logInfo {
		return nil
	}
	if s.filter != nil && !s.filter(e) {
		return nil
	}
	return s.f.Write(e, s.wc)
}

func (s *stream) flush(duration time.Duration) {
	time.Sleep(duration)
	for s.Flush() != nil {
		time.Sleep(duration)
	}
}

func (s *stream) rotate(o *Options) {
}
