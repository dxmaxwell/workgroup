// Package workgroup is a tool to manage goroutine life-cycle and error handling.
//
package workgroup

import (
	"context"
	"runtime"
	"sync"
)

// Context extends the standard context.Context to
// provide methods for managing goroutine exection.
type Context interface {
	context.Context
	Go(func() error)
	GoFor(int, func(int) error)
	GoForPool(int, int, func(int) error)
	Defer(func(*error, []error))
}

// PanicError captures the panic value of the goroutine.
// If a worker function returns an instance of PanicError,
// it will cause the containing workgroup to panic.
type PanicError interface {
	error
	Recover() interface{}
}

type _WGPanicError struct {
	Value interface{}
}

// Error provides a string version of the panic value.
func (err *_WGPanicError) Error() string {
	switch v := err.Value.(type) {
	case string:
		return v
	case interface { String() string }:
		return v.String()
	default:
		return "unknown"
	}
}

func (err *_WGPanicError) Recover() interface{} {
	return err.Value
}

type _WGCancelMode int

const (
	_AllCancelMode   = _WGCancelMode(iota)
	_FirstCancelMode = _WGCancelMode(iota)
)

const _WGClosedPanic = "workgroup closed"

type _WorkGroup struct {
	context.Context

	_Mut sync.Mutex

	_CMode  _WGCancelMode
	_Cancel context.CancelFunc

	_Count   int
	_First   error
	_Results []error

	_Defers []func(*error, []error)

	_Closed chan struct{}
}

// Defer arranges for the given function, d, to be executed
// after all goroutines within the workgroup have completed.
// If this function is called multiple times, then the deferred
// functions are executed in the reverse order that they are
// registered. The final error value of the workgroup can be
// modified within the deferred function by modifying the 
// err argument.
func (wg *_WorkGroup) Defer(d func(*error, []error)) {
	wg._Mut.Lock()
	defer wg._Mut.Unlock()

	if wg._Count < 0 {
		panic(_WGClosedPanic)
	}

	wg._Defers = append(wg._Defers, d)
}

// Go arranges for the worker funtion, w, to be executed in a goroutine.
func (wg *_WorkGroup) Go(w func() error) {
	wg._Mut.Lock()
	defer wg._Mut.Unlock()

	if wg._Count < 0 {
		panic(_WGClosedPanic)
	}

	wg._Count++

	erridx := -1
	if wg._Results != nil {
		erridx = len(wg._Results)
		wg._Results = append(wg._Results, nil)
	}

	go func() {
		var err error
		defer wg._Recover(&err, erridx)
		err = w()
	}()
}

// GoFor arranges for the worker function, w, to be executed on n goroutines.
func (wg *_WorkGroup) GoFor(n int, w func(int) error) {
	if n <= 0 {
		return
	}

	wg._Mut.Lock()
	defer wg._Mut.Unlock()

	if wg._Count < 0 {
		panic(_WGClosedPanic)
	}

	for i := 0; i < n; i++ {
		wg._Count++

		erridx := -1
		if wg._Results != nil {
			erridx = len(wg._Results)
			wg._Results = append(wg._Results, nil)
		}

		go func(idx int) {
			var err error
			defer wg._Recover(&err, erridx)
			err = w(idx)
		}(i)
	}
}

// GoForPool arranges for worker function, w, to be executed n times using a pool of goroutines.
// If the pool size is <=0 then the default size is used.
func (wg *_WorkGroup) GoForPool(n, size int, f func(int) error) {
	if size <= 0 {
		// Look for value in context!
		size = runtime.NumCPU()
	}

	if size >= n {
		wg.GoFor(n, f)
		return
	}

	if n <= 0 {
		return
	}

	wg._Mut.Lock()
	defer wg._Mut.Unlock()

	if wg._Count < 0 {
		panic(_WGClosedPanic)
	}

	ch := make(chan int, size)

	go func() {
		for i := 0; i < n; i++ {
			ch <- i
		}
		close(ch)
	}()

	erroffset := -1
	if wg._Results != nil {
		erroffset = len(wg._Results)
		for i := 0; i < n; i++ {
			wg._Results = append(wg._Results, nil)
		}
	}

	wg._Count += n
	for i := 0; i < size; i++ {
		go func() {
			for j := range ch {
				func(idx int) {
					erridx := -1
					if erroffset >= 0 {
						erridx = idx + erroffset
					}
					var err error
					defer wg._Recover(&err, erridx)
					err = f(idx)
				}(j)
			}
		}()
	}
}

func (wg *_WorkGroup) _Setup(f func(Context)) {
	defer func() {
		if v := recover(); v != nil {
			wg._Mut.Lock()
			defer wg._Mut.Unlock()
			if _, ok := wg._First.(PanicError); !ok {
				wg._First = &_WGPanicError{v}
				wg._Cancel()
			}
		}
	}()

	f(wg)
}

func (wg *_WorkGroup) _Recover(err *error, erridx int) {
	// fmt.Println("RECOVER")
	wg._Mut.Lock()
	defer wg._Mut.Unlock()

	if v := recover(); v != nil {
		if erridx >= 0 {
			wg._Results[erridx] = &_WGPanicError{v}
		}
		if _, ok := wg._First.(PanicError); !ok {
			if erridx >= 0 {
				wg._First = wg._Results[erridx]
			} else {
				wg._First = &_WGPanicError{v}
			}
		}
		wg._Cancel()
	} else if err != nil {
		if erridx >= 0 {
			wg._Results[erridx] = *err
		}
		if wg._First == nil {
			// fmt.Println("Store Error Result")
			wg._First = *err
		}
		wg._Cancel()
	} else if wg._CMode == _FirstCancelMode {
		wg._Cancel()
	}

	wg._Count--
	if wg._Count == 0 {
		wg._Count = -1
		close(wg._Closed)
		wg._Cancel()
	}
}

func (wg *_WorkGroup) _Wait() ([]error, error) {

	<-wg._Closed

	first := wg._First
	results := wg._Results

	for _, d := range wg._Defers {
		defer func(f func(*error, []error)) {
			f(&first, results)
		}(d)
	}

	if first, ok := wg._First.(PanicError); ok {
		panic(first.Recover())
	}

	return results, first
}

// All creates a new workgroup based on the provided parent Context,
// (which can be nil) and executes configuration function, f. The 
// workgroup Context is cancelled when the parent Context is canceled,
// a worker function returns an error, a worker function panics,
// the configuration function panics, or all worker functions have completed.
func All(parent context.Context, f func(Context)) error {
	if f == nil {
		return nil
	}

	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	wg := &_WorkGroup{
		Context: ctx,
		_Mut:   sync.Mutex{},
		_CMode:  _AllCancelMode,
		_Cancel: cancel,
		_Closed: make(chan struct{}),
	}

	wg._Setup(f)

	_, err := wg._Wait()

	return err
}

// First creates a new workgroup based on the provided parent Context,
// (which can be nil) and executes configuration function, f. The 
// workgroup Context is cancelled when the parent Context is canceled,
// a worker function returns an error, a worker function panics,
// the configuration function panics, or the first worker function completes.
func First(parent context.Context, w func(Context)) error {
	if w == nil {
		return nil
	}

	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	wg := &_WorkGroup{
		Context: ctx,
		_Mut:   sync.Mutex{},
		_CMode:  _FirstCancelMode,
		_Cancel: cancel,
		_Closed: make(chan struct{}),
	}

	wg._Setup(w)

	_, err := wg._Wait()

	return err
}

// AllWithErrors creates a new workgroup based on the provided parent Context,
// (which can be nil) and executes configuration function, f. The 
// workgroup Context is cancelled when the parent Context is canceled,
// a worker function returns an error, a worker function panics,
// the configuration function panics, or all worker functions have completed.
func AllWithErrors(parent context.Context, w func(Context)) ([]error, error) {
	results := []error{}
	
	if w == nil {
		return results, nil
	}

	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	wg := &_WorkGroup{
		Context:  ctx,
		_Mut:    sync.Mutex{},
		_CMode:   _AllCancelMode,
		_Cancel:  cancel,
		_Closed: make(chan struct{}),
		_Results: results,
	}

	wg._Setup(w)

	return wg._Wait()
}

// FirstWithErrors creates a new workgroup based on the provided parent Context,
// (which can be nil) and executes configuration function, f. The 
// workgroup Context is cancelled when the parent Context is canceled,
// a worker function returns an error, a worker function panics,
// the configuration function panics, or the first worker function completes.
func FirstWithErrors(parent context.Context, w func(Context)) ([]error, error) {
	results := []error{}
	
	if w == nil {
		return results, nil
	}

	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	wg := &_WorkGroup{
		Context:  ctx,
		_Mut:    sync.Mutex{},
		_CMode:   _FirstCancelMode,
		_Cancel:  cancel,
		_Closed: make(chan struct{}),
		_Results: results,
	}

	wg._Setup(w)

	return wg._Wait()
}
