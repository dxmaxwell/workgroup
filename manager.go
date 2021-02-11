package workgroup

import (
	"errors"
	"sync"
)

// DefaultManager is a function that provides the default manager.
var DefaultManager = CancelNeverFirstError

// Canceller cancels the work context.
type Canceller interface {
	Cancel()
}

// CancellerFunc is a function type that implements the Canceller interface.
type CancellerFunc func()

// Cancel calls the underlying function to cancel.
func (c CancellerFunc) Cancel() {
	c()
}

// Manager provides an interface for management of a work group.
type Manager interface {
	Manage(ctx Ctx, c Canceller, idx int, err *error) int
	Result() error
}

type firstError struct {
	mutex     sync.Mutex
	ncomplete int
	nerror    int
	result    error
}

// CancelOnFirstError initilizes a manager that
// cancels the work group context when a worker
// completes with an error.
func CancelOnFirstError() Manager {
	return &firstError{}
}

func (s *firstError) Result() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.result
}

func (s *firstError) Manage(ctx Ctx, c Canceller, idx int, err *error) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ncomplete++
	if *err != nil {
		s.nerror++
		if s.nerror == 1 {
			s.result = *err
			c.Cancel()
		}
	}

	return s.ncomplete
}

type firstSuccess struct {
	mutex    sync.Mutex
	nsuccess int
	nerror   int
	result   error
}

// CancelOnFirstSuccess initializes a manager that
// that cancels the work group context when a worker
// completes without error. If all workers complete
// with an error, then the first error is returned.
func CancelOnFirstSuccess() Manager {
	return &firstSuccess{}
}

func (s *firstSuccess) Result() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.result
}

func (s *firstSuccess) Manage(ctx Ctx, c Canceller, idx int, err *error) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if *err != nil {
		s.nerror++
		if s.nerror == 1 && s.nsuccess == 0 {
			s.result = *err
		}
	} else {
		s.nsuccess++
		if s.nsuccess == 1 {
			s.result = nil
			c.Cancel()
		}
	}

	return s.nsuccess + s.nerror
}

type firstDone struct {
	mutex     sync.Mutex
	ncomplete int
	result    error
}

// CancelOnFirstComplete initializes a new manager that
// cancels the work group context when a worker completes
// with or with an error.
func CancelOnFirstComplete() Manager {
	return &firstDone{}
}

func (s *firstDone) Result() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.result
}

func (s *firstDone) Manage(ctx Ctx, c Canceller, idx int, err *error) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ncomplete++
	if s.ncomplete == 1 {
		s.result = *err
		c.Cancel()
	}

	return s.ncomplete
}

type neverFirstError struct {
	mutex     sync.Mutex
	ncomplete int
	result    error
}

// CancelNeverFirstError initializes a new manager that never
// cancels the work group context, but will return the error
// from the first worker that completes with an error.
func CancelNeverFirstError() Manager {
	return &neverFirstError{}
}

func (m *neverFirstError) Result() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.result
}

func (m *neverFirstError) Manage(ctx Ctx, c Canceller, idx int, err *error) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.ncomplete++
	if *err != nil {
		if m.result == nil {
			m.result = *err
		}
	}

	return m.ncomplete
}

type PanicError struct {
	Value interface{}
}

func (e *PanicError) Error() string {
	switch v := e.Value.(type) {
	case string:
		return "panic: " + v
	case interface{ String() string }:
		return "panic: " + v.String()
	default:
		return "panic: unknown"
	}
}

type recoverWrapper struct {
	p bool
	m Manager
}

// Recover wraps a Manager, m, and if a worker
// panics during execution this wrapper will
// recover and create an instance of PanicError
// which will passed to the wrapped manager.
func Recover(m Manager) Manager {
	return &recoverWrapper{m: m, p: false}
}

// Repanic wraps a Manager, m, and if a worker
// panics during execution this wrapper will
// recover and create an instance of PanicError
// which will be passed to the wrapped manager.
// If the result of the wrapped manager is
// an instance of PanicError, then this wapper
// will panic when accessing the result.
func Repanic(m Manager) Manager {
	return &recoverWrapper{m: m, p: true}
}

func (w *recoverWrapper) Result() error {
	err := w.m.Result()
	if w.p {
		var perr *PanicError
		if errors.As(err, &perr) {
			panic(perr.Value)
		}
	}
	return err
}

func (w *recoverWrapper) Manage(ctx Ctx, c Canceller, idx int, err *error) int {
	if v := recover(); v != nil {
		*err = &PanicError{Value: v}
	}
	return w.m.Manage(ctx, c, idx, err)
}
