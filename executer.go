package workgroup

import (
	"runtime"
)

var DefaultLimit = runtime.NumCPU()

var DefaultExecuter = NewUnlimited

// Executer interface
type Executer interface {
	Execute(f func())
}

type unlimited struct{}

func NewUnlimited() Executer {
	return &unlimited{}
}

func (u *unlimited) Execute(f func()) {
	go f()
}

type limited struct {
	ch chan struct{}
}

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
