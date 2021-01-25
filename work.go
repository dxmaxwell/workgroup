package workgroup

import (
	"context"
	"sync"
)

type Ctx = context.Context

type Worker func(Ctx) error

func Work(ctx Ctx, e Executer, m Manager, ws ...Worker) error {
	if ctx == nil {
		ctx = context.Background()
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

	wg.Add(len(ws))
	for _, w := range ws {
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

type WorkerIdx func(Ctx, int) error

func WorkFor(ctx Ctx, n int, e Executer, m Manager, w WorkerIdx) error {
	if ctx == nil {
		ctx = context.Background()
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

			err = w(ctx, index)
		})
	}

	wg.Wait()

	if r, ok := m.Result().(interface{ Panic() }); ok {
		r.Panic()
	}
	return m.Result()
}

func For(n int, e Executer, m Manager, w WorkerIdx) Worker {
	return func(ctx Ctx) error {
		return WorkFor(ctx, n, e, m, w)
	}
}

func WorkChan(ctx Ctx, e Executer, m Manager, ws <-chan Worker) error {
	if ctx == nil {
		ctx = context.Background()
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

	for w := range ws {
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

func Chan(e Executer, m Manager, ws <-chan Worker) Worker {
	return func(ctx Ctx) error {
		return WorkChan(ctx, e, m, ws)
	}
}
