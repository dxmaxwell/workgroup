// Package workgroup is a tool to manage goroutine life-cycle and error handling.
//
package workgroup

import (
	"context"
	"sync"
)

type WorkGroup interface {

	// Add adds delta to the workgroup count, if the value of
	// delta causes the count to become less than zero, then
	// this function will panic. If the workgroup is closed
	// then this function will panic.
	Add(delta int)

	// Do calls the given worker function, w, and on completion captures
	// the error and decrements the workgroup count. When the workgroup
	// count is zero then the workgroup is considered closed.
	Do(func() error)

	// Go arranges for the worker function, w, to be executed on a goroutine.
	Go(func() error)

	// Defer arranges for the given function, d, to be executed
	// after all goroutines within the workgroup have completed.
	// If this function is called multiple times, then the deferred
	// functions are executed in the reverse order that they are
	// registered. The final error value of the workgroup can be
	// modified within the deferred function by modifying the
	// err argument.
	Defer(func(err *error))
}

// Context extends the standard context.Context to
// provide methods for managing goroutine exection.
type Context interface {
	context.Context
	WorkGroup
	All(func(Context))
	First(func(Context))
}

// PanicError captures the panic value of the goroutine.
// If a worker function returns an instance of PanicError,
// it will cause the containing workgroup to panic.
type PanicError interface {
	error
	Recover() interface{}
}

type _PanicError struct {
	Value interface{}
}

// Error provides a string version of the panic value.
func (err _PanicError) Error() string {
	switch v := err.Value.(type) {
	case string:
		return "panic error: " + v
	case interface{ String() string }:
		return "panic error: " + v.String()
	default:
		return "panic error: unknown"
	}
}

func (err _PanicError) Recover() interface{} {
	return err.Value
}

type _CancelMode int

const (
	_CancelModeAll   = _CancelMode(iota)
	_CancelModeFirst = _CancelMode(iota)
)

const (
	_PanicClosed  = "workgroup closed"
	_PanicCounter = "workgroup negative count"
)

func _MustReturn(f func() error) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = _PanicError{v}
		}
	}()
	return f()
}

type _WorkGroup struct {
	context.Context
	_CMode  _CancelMode
	_Cancel context.CancelFunc

	_Cond   *sync.Cond
	_Count  int
	_Error  error
	_Defers []func(*error)
}

func _New(parent context.Context, cmode _CancelMode) *_WorkGroup {
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	wg := &_WorkGroup{
		Context: ctx,
		_Cond:   sync.NewCond(&sync.Mutex{}),
		_CMode:  cmode,
		_Cancel: cancel,
	}

	return wg
}

func (wg *_WorkGroup) Add(delta int) {
	wg._Cond.L.Lock()
	defer wg._Cond.L.Unlock()

	if wg._Count < 0 {
		panic(_PanicClosed) // TODO: "misuse"
	}
	if wg._Count+delta < 0 {
		panic(_PanicCounter)
	}
	wg._Count += delta
}

func (wg *_WorkGroup) Defer(d func(err *error)) {
	if d == nil {
		return
	}
	
	wg._Cond.L.Lock()
	defer wg._Cond.L.Unlock()

	if wg._Count < 0 {
		panic(_PanicClosed)
	}

	wg._Defers = append(wg._Defers, d)
}

func (wg *_WorkGroup) Do(w func() error) {	
	var err error
	if w != nil {
		err = _MustReturn(w)
	}

	wg._Cond.L.Lock()
	defer wg._Cond.L.Unlock()

	if wg._Error == nil {
		wg._Error = err
		wg._Cancel()
	} else if wg._CMode == _CancelModeFirst {
		wg._Cancel()
	}

	wg._Count--
	if wg._Count == 0 {
		wg._Cond.Signal()
	}
}

func (wg *_WorkGroup) Go(w func() error) {
	wg.Add(1)
	wg.Do(w)
}

func (wg *_WorkGroup) All(init func(Context)) {
	wg.Go(func() error {
		return All(wg, init)
	})
}

func (wg *_WorkGroup) First(init func(Context)) {
	wg.Go(func() error {
		return First(wg, init)
	})
}

// All creates a new workgroup based on the provided parent Context,
// (which can be nil) and executes configuration function, f. The
// workgroup Context is cancelled when the parent Context is canceled,
// a worker function returns an error, a worker function panics,
// the configuration function panics, or all worker functions have completed.
func All(parent context.Context, init func(wtx Context)) error {
	if init == nil {
		return nil
	}

	wtx := _New(parent, _CancelModeAll)

	_Init(wtx, init)

	return _Wait(wtx)
}

// First creates a new workgroup based on the provided parent Context,
// (which can be nil) and executes configuration function, f. The
// workgroup Context is cancelled when the parent Context is canceled,
// a worker function returns an error, a worker function panics,
// the configuration function panics, or the first worker function completes.
func First(parent context.Context, w func(wtx Context)) error {
	if w == nil {
		return nil
	}

	wtx := _New(parent, _CancelModeFirst)

	_Init(wtx, w)

	return _Wait(wtx)
}

func _Init(wtx *_WorkGroup, init func(Context)) {
	wtx.Add(1)
	// hijack the current goroutine!
	wtx.Do(func() error {
		init(wtx)
		return nil
	})
}

func _Wait(wtx *_WorkGroup) (err error) {
	wtx._Cond.L.Lock()
	for wtx._Count > 0 {
		wtx._Cond.Wait()
	}

	err = wtx._Error
	wtx._Count = -1
	wtx._Cancel()

	for _, d := range wtx._Defers {
		defer func(f func(*error)) {
			f(&err)
		}(d)
	}

	wtx._Cond.L.Unlock()

	if err, ok := err.(PanicError); ok {
		panic(err.Recover())
	}

	return err
}
