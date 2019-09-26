// Public Domain (-) 2010-2014 The Golly Authors.
// See the Golly UNLICENSE file for details.

package log

import (
	"sync"
)

type Handler interface {
	Async() bool
	Close() error
	Flush() error
	Log(*Entry) error
}

// AsyncHandler wraps a handler with a channel and runs it in a separate
// goroutine so that logging doesn't block the current thread.
func AsyncHandler(bufsize int, h Handler) Handler {
	a := &async{
		flush:     make(chan struct{}),
		flushWait: make(chan struct{}),
		handler:   h,
		queue:     make(chan *Entry, bufsize),
		stop:      make(chan struct{}),
	}
	go a.run()
	return a
}

type async struct {
	closed    bool
	handler   Handler
	flush     chan struct{}
	flushWait chan struct{}
	mu        sync.Mutex
	queue     chan *Entry
	stop      chan struct{}
}

func (a async) Async() bool {
	return true
}

func (a async) Close() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.closed = true
	a.mu.Unlock()
	a.stop <- struct{}{}
	<-a.stop
	close(a.stop)
	return a.handler.Close()
}

func (a async) Flush() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.mu.Unlock()
	a.flush <- struct{}{}
	<-a.flushWait
	return nil
}

func (a async) Log(e *Entry) error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.queue <- e
	a.mu.Unlock()
	return nil
}

func (a async) run() {
	var e *Entry
	flush := false
	stop := false
	for {
		select {
		case e = <-a.queue:
			a.handler.Log(e)
		case <-a.stop:
			stop = true
		case <-a.flush:
			flush = true
		}
		if flush {
			a.mu.Lock()
			if len(a.queue) == 0 {
				a.mu.Unlock()
				a.handler.Flush()
				flush = false
				a.flushWait <- struct{}{}
			} else {
				a.mu.Unlock()
			}
		} else if stop && len(a.queue) == 0 { // TODO(tav): check if len is thread safe
			close(a.flush)
			close(a.flushWait)
			close(a.queue)
			a.stop <- struct{}{}
			break
		}
	}
}

func FailoverHandler(handlers ...Handler) Handler {
	return failover{handlers}
}

type failover struct {
	handlers []Handler
}

func (f failover) Async() bool {
	for _, h := range f.handlers {
		if h.Async() {
			return true
		}
	}
	return false
}

func (f failover) Close() error {
	var firstError error
	for _, h := range f.handlers {
		err := h.Close()
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func (f failover) Flush() error {
	var firstError error
	for _, h := range f.handlers {
		err := h.Flush()
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func (f failover) Log(e *Entry) error {
	var err error
	for _, h := range f.handlers {
		err = h.Log(e)
		if err == nil {
			return nil
		}
	}
	return err
}

func MultiHandler(handlers ...Handler) Handler {
	return list{handlers}
}

type list struct {
	handlers []Handler
}

func (l list) Async() bool {
	for _, h := range l.handlers {
		if h.Async() {
			return true
		}
	}
	return false
}

func (l list) Close() error {
	var firstError error
	for _, h := range l.handlers {
		err := h.Close()
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func (l list) Flush() error {
	var firstError error
	for _, h := range l.handlers {
		err := h.Flush()
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func (l list) Log(e *Entry) error {
	var prevError error
	for _, h := range l.handlers {
		err := h.Log(e)
		if prevError != nil {
			prevError = err
		}
	}
	return prevError
}
