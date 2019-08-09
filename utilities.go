package workgroup

import (
	"runtime"
)


type _ContextKey string

const ContextKeyPoolSizeDefault = _ContextKey("POOL_SIZE_DEFAULT")



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


func _GoPoolErrs(wtx Context, size int, errs []error, d func(*error), w func(Context) error) {
	if size < 1 {
		if v, ok := wtx.Value(ContextKeyPoolSizeDefault).(int); ok {
			size = v
		}
		if size < 1 {
			size = runtime.NumCPU()
		}
	}

	_GoForErrs(wtx, size, errs, d, func(c Context, _ int) error { return w(c) })
}


func _GoForPoolErrs(wtx Context, n, size int, errs []error, d func(*error), w func(Context, int) error) {
	if size < 1 {
		if v, ok := wtx.Value(ContextKeyPoolSizeDefault).(int); ok {
			size = v
		}
		if size < 1 {
			size = runtime.NumCPU()
		}
	}

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
