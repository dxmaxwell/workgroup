package workgroup

import (
	"context"
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
	// Manage controls the cancellation and overall error
	// of the work group. The index of completed worker
	// is provided along with its error. This function
	// is expected to return the number of completed workers.
	Manage(ctx context.Context, c Canceller, idx int, err *error) int

	// Error returns the final error of this work group.
	Error() error
}

type firstError struct {
	mutex     sync.Mutex
	ncomplete int
	nerror    int
	err       error
}

// CancelOnFirstError initilizes a manager that
// cancels the work group context when a worker
// completes with an error.
func CancelOnFirstError() Manager {
	return &firstError{}
}

func (m *firstError) Error() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.err
}

func (m *firstError) Manage(ctx context.Context, c Canceller, idx int, err *error) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.ncomplete++
	if *err != nil {
		m.nerror++
		if m.nerror == 1 {
			m.err = *err
			c.Cancel()
		}
	}

	return m.ncomplete
}

type firstSuccess struct {
	mutex    sync.Mutex
	nsuccess int
	nerror   int
	err      error
}

// CancelOnFirstSuccess initializes a manager that
// that cancels the work group context when a worker
// completes without error. If all workers complete
// with an error, then the first error is returned.
func CancelOnFirstSuccess() Manager {
	return &firstSuccess{}
}

func (m *firstSuccess) Error() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.err
}

func (m *firstSuccess) Manage(ctx context.Context, c Canceller, idx int, err *error) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if *err != nil {
		m.nerror++
		if m.nerror == 1 && m.nsuccess == 0 {
			m.err = *err
		}
	} else {
		m.nsuccess++
		if m.nsuccess == 1 {
			m.err = nil
			c.Cancel()
		}
	}

	return m.nsuccess + m.nerror
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

func (m *firstDone) Error() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.result
}

func (m *firstDone) Manage(ctx context.Context, c Canceller, idx int, err *error) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.ncomplete++
	if m.ncomplete == 1 {
		m.result = *err
		c.Cancel()
	}

	return m.ncomplete
}

type neverFirstError struct {
	mutex     sync.Mutex
	ncomplete int
	err       error
}

// CancelNeverFirstError initializes a new manager that never
// cancels the work group context, but will return the error
// from the first worker that completes with an error.
func CancelNeverFirstError() Manager {
	return &neverFirstError{}
}

func (m *neverFirstError) Error() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.err
}

func (m *neverFirstError) Manage(ctx context.Context, c Canceller, idx int, err *error) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.ncomplete++
	if *err != nil {
		if m.err == nil {
			m.err = *err
		}
	}

	return m.ncomplete
}

// PanicError is an error that represents a recovered panic
// and contains the value returned from a call to recover.
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

func (w *recoverWrapper) Error() error {
	err := w.m.Error()
	if w.p {
		var perr *PanicError
		if errors.As(err, &perr) {
			panic(perr.Value)
		}
	}
	return err
}

func (w *recoverWrapper) Manage(ctx context.Context, c Canceller, idx int, err *error) int {
	if v := recover(); v != nil {
		*err = &PanicError{Value: v}
	}
	return w.m.Manage(ctx, c, idx, err)
}
