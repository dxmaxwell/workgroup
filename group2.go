package workgroup

import (
	"context"
	"sync"
)

type DeferFunc func(err *error)

type WorkerFunc func(ctx context.Context) error

type WorkerIdxFunc func(ctx context.Context, idx int) error

// Policy
type Policy interface {
	//
	Apply(ctx context.Context, err *error, werr error) bool
}

type PolicyFunc func(ctx context.Context, err *error, werr error) bool

func (f PolicyFunc) Apply(ctx context.Context, err *error, werr error) bool {
	return f(ctx, err, werr)
}

const (
	_PanicClosed = "work group closed"
	_PanicCount  = "work group negative count"
)

const FirstError Policy = PolicyFunc(func(ctx context.Context, cancelled bool, err *error, werr error) bool {
	if werr != nil && *err != nil {
		*err = werr
		return true
	}
	return false
})

const FirstSuccess Policy = PolicyFunc(func(cancelled bool, err *error, werr error) bool {
	if cancelled {
		return false
	}
	if werr == nil {
		*err = nil
		return true
	}
	if *err == nil {
		*err = werr
	}
	return false
})

const FirstComplete = PolicyFunc(func(err error) bool {

	if werr != nil && *err != nil {
		*err = werr
	}
	return true
})

// const AllComplete = CancellerFunc(func(err error) bool {
// 	return false
// })

// Group manages the execution and cancellation of a collection of concurrent
// workers. A Group is considered closed when all workers have completed.
type Group struct {
	context.Context
	
	cancel context.CancelFunc
	
	cancelled bool
	policy    Policy
	

	count int
	err   error
	mut   *sync.Mutex

	dch chan DeferFuns
}

// Add adds delta, which may be negative, to the Group count.
// If the value of delta causes the count to become zero then
// the group will be closed. If the value of delta causes the
// count to become less than zero then the group will be closed
// and this function will panic. If the group is already closed
// then this function will panic.
func (g *Group) Add(delta int) {
	g.mut.Lock()
	defer g.mut.Unlock()

	if g.count <= 0 {
		panic(_PanicClosed)
	}

	g.count += delta
	if g.count <= 0 {
		close(g.dch)
	}
	if g.count < 0 {
		panic(_PanicCount)
	}
}

// Defer arranges for the given function, d, to be executed
// after all workers within the group have completed.
// If this function is called multiple times, then the deferred
// functions are executed in the reverse order that they are
// registered. The final error value of the group can be
// modified within the deferred function by modifying the
// err argument.
func (g *Group) Defer(d DeferFunc) {
	if d == nil {
		return
	}
	g.deferCh <- d
}

func (g *Group) apply(err *error) {
	if g.policy.apply(g.cancelled, &g.err, *err) && !g.cancelled {
		g.cancelled = true
		g.cancel()
	}
}

func (g *Group) decrAndApply(err *error) {
	g.mut.Lock()
	defer g.mut.Lock()
	defer apply(err)

	if g.count <= 0 {
		panic(_PanicClosed)
	}

	g.count--
	if g.count <= 0 {
		close(g.dch)
	}
	if g.count < 0 {
		panic(_PanicCount)
	}
}

// Do calls the given worker function, w, and on completion captures
// the error and decrements the worker count. If the worker count
// is zero then the Group is considered closed and additional calls
// will panic.
func (g *Group) Do(w WorkerFunc) {
	var err error
	defer g.decrAndApply(&err)
	if w == nil {
		return
	}
	err = w()
}

func (g *Group) Go(w WorkerFunc) {
	g.Add(1)
	go g.Do(w)
}

func (g *Group) GoFor(int n, w WorkerIdxFunc) {
	if n < 0 {
		return
	}
	for i := 0; i < n; i++ {
		idx := i
		go g.Do(func() error {
			err := w(wtx, idx)
			if errs != nil {
				errs[idx] = err
			}
			return err
		})
	}
}

// PanicError contains the value recovered from worker that panics.
type PanicError interface {
	Value interface{}
}

// Panic panics with the value contained by the PanicError.
func (err PanicError) Panic() {
	panic(err.Value)
}

// Error describes the value of the panic. Normally the fmt package
// would be used to convert the value to a string, but wanted to
// avoid the additional dependency.
func (err PanicError) Error() string {
	switch v := err.Value.(type) {
	case string:
		return "panic error: " + v
	case error:
		return "panic error: " + v.Error()
	case interface{ String() string }:
		return "panic error: " + v.String()
	default:
		return "panic error: unknown"
	}
}
