package workgroup

import (
	"context"
	"sync"
)



type WorkGroup4 interface {
	Add(delta int)
	Do(worker func() error) 
	Go(worker func() error)
}

type Context4 interface {
	context.Context
	WorkGroup
}

func _CallSafely(f func() error) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = &_WGPanicError{v}
		}
	}()
	return f()
}

type _WorkGroup4 struct {
	context.Context
	_CMode _WGCancelMode
	_Cancel context.CancelFunc

	_Cond sync.Cond
	_Count int
	_Defers []func(*error)
}

func (wg *_WorkGroup4) Add(delta int) {
	wg._Cond.L.Lock()
	defer wg._Cond.L.Unlock()

	if wg.Count < 0 {
		panic(_WGClosedPanic) // TODO: "misuse"
	}
	if wg.Count + delta < 0 {
		panic(_WGClosedPanic)
	}
	wg_Count += delta
}

func (wg *_WorkGroup4) Do(w func() error) {
	err := _CallSafely(w)

	wg._Cond.L.Lock()
	defer wg._Cond.L.Unlock()

	if wg._Error == nil {
		wg._Error = err
		wg._Cancel()
	} else if (wg._CMode == _FirstCancelMode) {
		wg._Cancel()
	}

	wg._Count--
	if wg._Count == 0 {
		wg._Cond.Signal()
	}
}


func (wg *_WorkGroup4) Go(w func() error) {
	wg.Add(1)
	wg.Do(w)
}

func (wg *_WorkGroup4) All(init func(wtx *_WorkGroup4)) {
	
	if (init == nil) {
		return nil
	}

	ctx, cancel := context.WithCancel(wg)

	wtx := &_WorkGroup4{
		Context: ctx,
		_Cond: sync.NewCond(&sync.Mutex{}),
		_CMode: _AllCancelMode,
		_Cancel: cancel,
	}

	wtx.Go(func() error {
		init(wg)
		return nil
	})
}


func GoForErrs(n int, d func(*error, []error), w func(int) error) func(*_WorkGroup4) {
	return func(wtx *_WorkGroup4) {
		wtx.Add(n)

		errs := make([]error, n)

		for i := 0; i < n; i++ {
			idx := i
			go wtx.Do(func() error {
				err[idx] = w(idx)
				return err[idx]
			})
		}
		wtx.Defer(func(err *error) {
			d(err, errs)
		})
	}
}


func All(parent context.Context, init func(wtx Context4)) (err error) {

	if (init == nil) {
		return nil
	}

	if (parent == nil) {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	wg := &_WorkGroup4{
		Context: ctx,
		_Cond: sync.NewCond(&sync.Mutex{}),
		_CMode: _AllCancelMode,
		_Cancel: cancel,
	}

	wg.Go(func() error {
		init(wg)
		return nil
	})

	wg._Cond.L.Lock()
	for wg._Count > 0 {
		wg._Cond.Wait()
	}

	wg._Count = -1
	wg._Cancel()

	err = wg._Error

	for _, d := range wg._Defers {
		defer func(f func(*error)) {
			f(&err)
		}(d)
	}

	wg._Cond.L.Unlock()

	if err, ok := err.(PanicError); ok {
		panic(err.Recover())
	}

	return
}