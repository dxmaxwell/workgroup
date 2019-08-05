package workgroup

import (
	"sync"
	"context"
)





type WorkContext interface {
	context.Context

	//Go(w func() error, d func(err error) )
	GoFor(n int, w func(idx int) error, d func(err error))
	GoForErrs(n int, w func(idx int) error, d func(err error, errs []error))
}


type _Workgroup3 struct {
	context.Context

	_CMode _WGCancelMode
	_Cancel context.CancelFunc

	_WCond   *sync.Cond
	_Workers []func()
	_WDefers []func(*error)
	_WCount  int

	_ELock *sync.Mutex
	_Error error
}

func (wg *_Workgroup3) Go(w func(int) error, d func(error)) {
	wg._GoForErrs(1, nil, w, func(err error, errs []error) { d(err) })
}

func (wg *_Workgroup3) GoFor(n int, w func(int) error, d func(error)) {
	if n < 0 {
		n = 0
	}
	wg._GoForErrs(n, nil, w, func(err error, errs []error) { d(err) })
}

func (wg *_Workgroup3) GoForErrs(n int, w func(int) error, d func(error, []error)) {
	if n < 0 {
		n = 0
	}
	wg._GoForErrs(n, make([]error, n), w, d)
}

func (wg *_Workgroup3) _GoForErrs(n int, errs []error, w func(int) error, d func(error, []error)) {	
	wg._WCond.L.Lock()
	defer wg._WCond.L.Unlock()

	if wg._WCount < 0 {
		panic(_WGClosedPanic)
	}

	count := n

	for i := 0; i < n; i++ {
		go func(idx int) {
			var err error
			defer wg._Recover(&count, idx, &err, errs, d)
			err = w(idx)
		}(i)
	}

	wg._WDefers = append(wg._WDefers, func(err *error) { d(*err, errs) })
}


func (wg *_Workgroup3) _Recover(count *int, idx int, err *error, errs []error, d func(error, []error)) {
	wg._ELock.Lock()

	if v := recover(); v != nil {
		if errs != nil {
		  errs[idx] = &_WGPanicError{v}
		}
		if _, ok := wg._Error.(PanicError); !ok {
			if errs != nil {
				wg._Error = errs[idx]
			} else {
				wg._Error = &_WGPanicError{v}
			}
		}
		wg._Cancel()
	} else if err != nil {
		if errs != nil {
			errs[idx] = *err
		}
		if wg._Error == nil {
			// fmt.Println("Store Error Result")
			wg._Error = *err
		}
		wg._Cancel()
	} else if wg._CMode == _FirstCancelMode {
		wg._Cancel()
	}

	*count--
	if *count == 0 {
		// Unlock is done by DeferRecover!
		defer wg._DeferRecover()
		d(wg._Error, errs)
	} else {
		wg._ELock.Unlock()
	}
}


func (wg *_Workgroup3) _DeferRecover() {
	if v := recover(); v != nil {
		if _, ok := wg._Error.(PanicError); !ok {
			wg._Error = &_WGPanicError{v}
			wg._Cancel()
		}
	}
	wg._ELock.Unlock()

	wg._WCond.L.Lock()
	defer wg._WCond.L.Unlock()

	wg._WCount--
	if wg._WCount == 0 {
		wg._WCond.Signal()
	}
}


func All2(ptx context.Context, init func(WorkContext)) error {
	if ptx == nil {
		ptx = context.Background()
	}

	ctx, cancel := context.WithCancel(ptx)

	wg := &_Workgroup3{
		Context: ctx,
		_CMode: _AllCancelMode,
		_Cancel: cancel,
		_WCond: sync.NewCond(&sync.Mutex{}),
		_Workers: []func(){},
		_WDefers: []func(*error){}, 
		_ELock: &sync.Mutex{},
	}

	init(wg)

	wg._WCond.L.Lock()

	for _, worker := range wg._Workers {
		worker()
	}
	wg._Workers = nil

	for wg._WCount > 0 {
		wg._WCond.Wait()
	}
	wg._WCount = -1

	defer wg._ELock.Unlock()
	for _, d := range wg._WDefers {
		defer func(f func(*error)) {
			f(&wg._Error)
		}(d)
	}

	wg._WCond.L.Unlock()
	wg._ELock.Lock()

	if err, ok := wg._Error.(PanicError); ok {
		panic(err.Recover())
	}

	return wg._Error
}


