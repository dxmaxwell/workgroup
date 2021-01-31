package workgroup

import "sync/atomic"

// DefaultManager is a function that provides the default manager
var DefaultManager = CancelNeverFirstError

// Canceller allows a context to be cancelled.
type Canceller interface {
	Cancel()
}

// CancellerFunc is a function that implements the Canceller interface.
type CancellerFunc func()

// Cancel calls the underlying function to cancel
func (c CancellerFunc) Cancel() {
	c()
}

// Manager provides a interface for management of a work group.
type Manager interface {
	Manage(ctx Ctx, c Canceller, err error)
	Result() error
}

type firstError struct {
	result error
	nerr   uint64
}

// CancelOnFirstError initilizes a manager that
// cancels the work group context when a worker
// completes with an error.
func CancelOnFirstError() Manager {
	return &firstError{}
}

func (s *firstError) Result() error {
	return s.result
}

func (s *firstError) Manage(ctx Ctx, c Canceller, err error) {
	if err != nil {
		nerr := atomic.AddUint64(&s.nerr, 1)
		if nerr == 1 {
			s.result = err
			c.Cancel()
		}
	}
}

type firstSuccess struct {
	result  error
	success bool
	nerrscs uint64
}

// CancelOnFirstSuccess initializes a manager that
// that cancels the work group context when a worker
// completes without error. If all workers complete
// with an error, then the first error is returned.
func CancelOnFirstSuccess() Manager {
	return &firstSuccess{}
}

func (s *firstSuccess) Result() error {
	if !s.success {
		return s.result
	}
	return nil
}

func (s *firstSuccess) Manage(ctx Ctx, c Canceller, err error) {
	var nerr uint64
	var nscs uint64
	for {
		nerrscs := atomic.LoadUint64(&s.nerrscs)
		nerr = nerrscs & 0xFF00
		nscs = (nerrscs & 0x00FF) << 32
		if err != nil {
			nerr++
		} else {
			nscs++
		}

		if atomic.CompareAndSwapUint64(&s.nerrscs, nerrscs, nerr|(nscs>>32)) {
			break
		}
	}

	if err != nil {
		if nerr == 1 {
			s.result = err
		}
	} else {
		if nscs == 1 {
			s.success = true
			c.Cancel()
		}
	}
}

type firstDone struct {
	result error
	ndone  uint64
}

// CancelOnFirstComplete initializes a new manager that
// cancels the work group context when a worker completes
// with or with an error.
func CancelOnFirstComplete() Manager {
	return &firstDone{}
}

func (s *firstDone) Result() error {
	return s.result
}

func (s *firstDone) Manage(ctx Ctx, c Canceller, err error) {
	ndone := atomic.AddUint64(&s.ndone, 1)
	if ndone == 1 {
		s.result = err
		c.Cancel()
	}
}

type neverFirstError struct {
	result error
	nerr   uint64
}

// CancelNeverFirstError initializes a new manager never
// cancels the work group context, but will return the
// error from the first worker that completes with an error.
func CancelNeverFirstError() Manager {
	return &neverFirstError{}
}

func (m *neverFirstError) Result() error {
	return m.result
}

func (m *neverFirstError) Manage(ctx Ctx, c Canceller, err error) {
	if err != nil {
		nerr := atomic.AddUint64(&m.nerr, 1)
		if nerr == 1 {
			m.result = err
		}
	}
}

type panicError struct {
	v interface{}
}

func (e *panicError) Error() string {
	switch v := e.v.(type) {
	case string:
		return "panic error: " + v
	case interface{ String() string }:
		return "panic error: " + v.String()
	default:
		return "panic error: unknown"
	}
}

func (e *panicError) Panic() {
	panic(e.v)
}

type recoverWrapper struct {
	m Manager
}

func Recover(m Manager) Manager {
	return &recoverWrapper{m: m}
}

func (w *recoverWrapper) Result() error {
	return w.m.Result()
}

func (w *recoverWrapper) Manage(ctx Ctx, c Canceller, err error) {
	if v := recover(); v != nil {
		err = &panicError{v: v}
	}
	w.m.Manage(ctx, c, err)
}
