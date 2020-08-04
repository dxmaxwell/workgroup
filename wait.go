package workgroup

import (
	"context"
	"runtime"
)

type _ContextKey string

// CtxKeyPoolSizeDefault is the key used to override the pool size default.
const CtxKeyPoolSizeDefault = _ContextKey("POOL_SIZE_DEFAULT")

// GoFor arranges for worker function, w, to be executed n times concurrently.
// On completion the deferred function, d, is executed if it is provided.
func GoFor(n int, d func(*error), w func(Context, int) error) func(Context) {
	return func(wtx Context) {
		if n < 0 {
			n = 0
		}
		_GoForErrs(wtx, n, nil, d, w)
	}
}

// GoForErrs arranges for worker, w, to be executed on n number of goroutines.
// On completion the deferred function, d, is executed if it is provided.
// Errors are collected from each worker function and provided as an
// argument to the deferred function.
func GoForErrs(n int, d func(*error, []error), w func(Context, int) error) func(Context) {
	return func(wtx Context) {
		if n < 0 {
			n = 0
		}
		errs := make([]error, n)
		_GoForErrs(wtx, n, errs, func(err *error) { d(err, errs) }, w)
	}
}

func _GoForErrs(wtx Context, n int, errs []error, d func(*error), w func(Context, int) error) {
	wtx.Add(n)
	wtx.Defer(d)

	for i := 0; i < n; i++ {
		idx := i
		go wtx.Do(func() error {
			err := w(wtx, idx)
			if errs != nil {
				errs[idx] = err
			}
			return err
		})
	}
}

// GoPool arranges for worker function, w, to be executed on size number of goroutines.
// If size is zero of less then the default pool size is used. The default pool size
// either obtained from the Context, using the key CtxKeyPoolSizeDefault, otherwise
// the number of CPUs is used. On completion deferred function, d, is executed if
// it is provided.
func GoPool(size int, d func(*error), w func(Context) error) func(Context) {
	return func(wtx Context) {
		if size < 1 {
			size = _PoolSize(wtx, size)
		}
		_GoPoolErrs(wtx, size, nil, d, w)
	}
}

// GoPoolErrs arranges for worker function, w, to be executed on size number of goroutines.
// If size is zero of less then the default pool size is used. The default pool size
// either obtained from the Context, using the key CtxKeyPoolSizeDefault, otherwise
// the number of CPUs is used. On completion deferred function, d, is executed if
// it is provided. Errors are collected from each worker function and provided as an
// argument to the deferred function.
func GoPoolErrs(size int, d func(*error, []error), w func(Context) error) func(Context) {
	return func(wtx Context) {
		if size < 1 {
			size = _PoolSize(wtx, size)
		}
		errs := make([]error, size)
		_GoPoolErrs(wtx, size, errs, func(err *error) { d(err, errs) }, w)
	}
}

func _GoPoolErrs(wtx Context, size int, errs []error, d func(*error), w func(Context) error) {
	_GoForErrs(wtx, size, errs, d, func(c Context, _ int) error { return w(c) })
}

// GoForPool arranges for worker function, w, to be executed n times on size number of goroutines.
// If size is zero of less then the default pool size is used. The default pool size
// either obtained from the Context, using the key CtxKeyPoolSizeDefault, otherwise
// the number of CPUs is used. On completion deferred function, d, is executed if
// it is provided.
func GoForPool(n, size int, d func(*error), w func(Context, int) error) func(Context) {
	return func(wtx Context) {
		if n < 0 {
			n = 0
		}
		if size < 1 {
			size = _PoolSize(wtx, size)
		}
		_GoForPoolErrs(wtx, n, size, nil, d, w)
	}
}

// GoForPoolErrs arranges for worker function, w, to be executed n times on size number of goroutines.
// If size is zero of less then the default pool size is used. The default pool size
// either obtained from the Context, using the key CtxKeyPoolSizeDefault, otherwise
// the number of CPUs is used. On completion deferred function, d, is executed if
// it is provided. Errors are collected from each worker function and provided as an
// argument to the deferred function.
func GoForPoolErrs(n, size int, d func(*error, []error), w func(Context, int) error) func(Context) {
	return func(wtx Context) {
		if n < 0 {
			n = 0
		}
		if size < 1 {
			size = _PoolSize(wtx, size)
		}
		errs := make([]error, n)
		_GoForPoolErrs(wtx, n, size, errs, func(err *error) { d(err, errs) }, w)
	}
}

// Discover the default pool size from the Context or use the runtime NumCPU().
func _PoolSize(ctx context.Context, size int) int {
	if v, ok := ctx.Value(CtxKeyPoolSizeDefault).(int); ok {
		size = v
	}
	if size < 1 {
		size = runtime.NumCPU()
	}
	return size
}

func _GoForPoolErrs(wtx Context, n, size int, errs []error, d func(*error), w func(Context, int) error) {
	if n <= size {
		_GoForErrs(wtx, n, errs, d, w)
		return
	}

	wtx.Add(n)

	ch := make(chan int, size)

	_GoPoolErrs(wtx, size, nil, d, func(c Context) error {
		for j := range ch {
			idx := j
			wtx.Do(func() error {
				err := w(wtx, idx)
				if errs != nil {
					errs[idx] = err
				}
				return err
			})
		}
		return nil
	})

	for j := 0; j < n; j++ {
		ch <- j
	}
	close(ch)
}
