package workgroup

import (
	"context"
	"sync"
)

type Ctx = context.Context

type Worker func(Ctx) error

type WorkerIdx func(Ctx, int) error

func Work(ctx Ctx, e Executer, m Manager, g ...Worker) error {
	if ctx == nil {
		ctx = context.TODO()
	}

	if e == nil {
		e = DefaultExecuter()
	}

	if m == nil {
		m = DefaultManager()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	wg.Add(len(g))
	for _, w := range g {
		worker := w
		e.Execute(func() {
			defer wg.Done()

			var err error
			defer func() {
				m.Manage(ctx, CancellerFunc(cancel), err)
			}()

			err = worker(ctx)
		})
	}

	wg.Wait()

	if r, ok := m.Result().(interface{ Panic() }); ok {
		r.Panic()
	}
	return m.Result()
}

func Group(e Executer, m Manager, g ...Worker) Worker {
	return func(ctx Ctx) error {
		return Work(ctx, e, m, g...)
	}
}

func WorkFor(ctx Ctx, n int, e Executer, m Manager, g WorkerIdx) error {
	if ctx == nil {
		ctx = context.TODO()
	}

	if e == nil {
		e = DefaultExecuter()
	}

	if m == nil {
		m = DefaultManager()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	wg.Add(n)
	for i := 0; i < n; i++ {
		index := i
		e.Execute(func() {
			defer wg.Done()

			var err error
			defer func() {
				m.Manage(ctx, CancellerFunc(cancel), err)
			}()

			err = g(ctx, index)
		})
	}

	wg.Wait()

	if r, ok := m.Result().(interface{ Panic() }); ok {
		r.Panic()
	}
	return m.Result()
}

func GroupFor(n int, e Executer, m Manager, g WorkerIdx) Worker {
	return func(ctx Ctx) error {
		return WorkFor(ctx, n, e, m, g)
	}
}

func WorkChan(ctx Ctx, e Executer, m Manager, g <-chan Worker) error {
	if ctx == nil {
		ctx = context.TODO()
	}

	if e == nil {
		e = DefaultExecuter()
	}

	if m == nil {
		m = DefaultManager()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	for w := range g {
		wg.Add(1)
		worker := w
		e.Execute(func() {
			defer wg.Done()

			var err error
			defer func() {
				m.Manage(ctx, CancellerFunc(cancel), err)
			}()

			err = worker(ctx)
		})
	}

	wg.Wait()

	if r, ok := m.Result().(interface{ Panic() }); ok {
		r.Panic()
	}
	return m.Result()
}

func GroupChan(e Executer, m Manager, g <-chan Worker) Worker {
	return func(ctx Ctx) error {
		return WorkChan(ctx, e, m, g)
	}
}
