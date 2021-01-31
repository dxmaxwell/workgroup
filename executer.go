package workgroup

import (
	"runtime"
)

// DefaultLimit is used for limited and pool executers
// default value is the number of CPUs provided by
// `runtime.NumCPU()`.
var DefaultLimit = runtime.NumCPU()

// DefaultExecuter is a function that provides the default Executer.
var DefaultExecuter = NewUnlimited

// Executer arranges for function, f, to executed.
type Executer interface {
	Execute(f func())
}

type unlimited struct{}

// NewUnlimited returns a executer that will execute functions
// on an unlimted number of goroutines in parallel.
func NewUnlimited() Executer {
	return &unlimited{}
}

func (u *unlimited) Execute(f func()) {
	go f()
}

type limited struct {
	ch chan struct{}
}

// NewLimited returns an executer that will execute functions
// on at most, n, goroutines simultaneously. If n <= 0 then
// the value provided by DefaultLimit will be used.
func NewLimited(n int) Executer {
	if n <= 0 {
		n = DefaultLimit
	}
	if n <= 0 {
		n = runtime.NumCPU()
	}
	return &limited{
		ch: make(chan struct{}, n),
	}
}

func (l *limited) add() {
	l.ch <- struct{}{}
}

func (l *limited) done() {
	<-l.ch
}

func (l *limited) Execute(f func()) {
	l.add()
	go func() {
		defer l.done()
		f()
	}()
}

type pool struct {
	ch chan func()
}

// NewPool initializes a new pool executer that will execute
// functions on fixed number of goroutines. If n <= 0 then
// the values in DefaultLimit is used. Note that the provided
// context must be cancelled to ensure that the pool releases
// all resources.
func NewPool(ctx Ctx, n int) Executer {
	if n <= 0 {
		n = DefaultLimit
	}
	if n <= 0 {
		n = runtime.NumCPU()
	}

	p := &pool{
		ch: make(chan func()),
	}

	if ctx != nil {
		go func() {
			<-ctx.Done()
			close(p.ch)
		}()
	}

	for i := 0; i < n; i++ {
		go func() {
			for f := range p.ch {
				f()
			}
		}()
	}
	return p
}

func (p *pool) Execute(f func()) {
	p.ch <- f
}
