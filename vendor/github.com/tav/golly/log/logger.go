// Public Domain (-) 2010-2014 The Golly Authors.
// See the Golly UNLICENSE file for details.

package log

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

var (
	entryPool = &sync.Pool{}
	root      = &Logger{}
	slicePool = &sync.Pool{}
)

type Logger struct {
	context    string
	handler    Handler
	parent     *Logger
	stacktrace bool
	stop       bool
}

// Create a new logger.
func (l *Logger) New(ctx string) *Logger {
	if l.context != "" {
		if ctx != "" {
			ctx = l.context + "." + ctx
		} else {
			ctx = l.context
		}
	}
	return &Logger{
		context: l.context,
		parent:  l,
	}
}

// Close the underlying handler for this logger.
//
// Please note that if you set a handler on the root logger, then it is your
// responsibility to manually close any underlying resources.
func (l *Logger) Close() {
	if l.handler != nil {
		l.handler.Close()
	}
}

func (l *Logger) Debug(args ...interface{}) {
	l.log(fmt.Sprint(args...), nil, false, true)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(fmt.Sprintf(format, args...), nil, false, true)
}

func (l *Logger) DebugData(message string, data interface{}) {
	l.log(message, data, false, true)
}

func (l *Logger) Error(args ...interface{}) {
	l.log(fmt.Sprint(args...), nil, true, false)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(fmt.Sprintf(format, args...), nil, true, false)
}

func (l *Logger) ErrorData(message string, data interface{}) {
	l.log(message, data, true, false)
}

// Flush the underlying handler for this logger.
func (l *Logger) Flush() {
	if l.handler != nil {
		l.handler.Flush()
	}
}

func (l *Logger) Info(args ...interface{}) {
	l.log(fmt.Sprint(args...), nil, false, false)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(fmt.Sprintf(format, args...), nil, false, false)
}

func (l *Logger) InfoData(message string, data interface{}) {
	l.log(message, data, false, false)
}

func (l *Logger) log(msg string, data interface{}, isError bool, debug bool) {
	var e *Entry
	entry := entryPool.Get()
	if entry == nil {
		e = &Entry{}
	} else {
		e = entry.(*Entry)
		e.File = ""
		e.Line = 0
		e.Stacktrace = ""
	}
	e.Context = l.context
	e.Data = data
	e.Error = isError
	e.Message = msg
	e.Timestamp = time.Now()
	l.logEntry(e, debug, 3)
}

func (l *Logger) logEntry(e *Entry, debug bool, depth int) {
	var buf []byte
	var debugSet bool
	var async bool
	for {
		if debug && !debugSet {
			_, e.File, e.Line, _ = runtime.Caller(depth)
			slice := slicePool.Get()
			if slice == nil {
				buf = make([]byte, 4096)
			} else {
				buf = slice.([]byte)
			}
			e.Stacktrace = string(buf[:runtime.Stack(buf, false)])
			debugSet = true
		}
		if l.handler != nil {
			l.handler.Log(e)
			if l.handler.Async() {
				async = true
			}
		}
		if l.parent == nil || l.stop {
			break
		}
		l = l.parent
	}
	if !async {
		entryPool.Put(e)
		if debugSet {
			slicePool.Put(buf)
		}
	}
}

func (l *Logger) LogEntry(e *Entry) {
	l.logEntry(e, false, 2)
}

// SetHandler and ToggleDebug are not intended to be threadsafe. Make sure
// to set them before using the logger from multiple goroutines.
func (l *Logger) SetHandler(h Handler) {
	l.handler = h
}

func (l *Logger) StopPropagation() {
	l.stop = true
}
