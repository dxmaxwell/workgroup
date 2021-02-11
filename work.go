package workgroup

import (
	"context"
	"sync"
)

// Ctx is an alias for the standard `context.Context`
type Ctx = context.Context

// Worker is a function that performs work
type Worker func(Ctx) error

// IdxWorker is a function that performs work with for a given index
type IdxWorker func(Ctx, int) error

// Work arranges for a group of workers to be executed
// and then waits for these workers to complete.
// The executer, e, is responsible for executing these workers
// with various levels of concurrancy. The manager, m, determines
// when the context will be canceled and which error is returned.
// If executer, e, is not provided then DefaultExecuter
// is called for obtain the default. If manager, m, is not provied
// then DefaultManager is called be obtain the default manager.
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
	for i, w := range g {
		index := i
		worker := w
		e.Execute(ctx, func(ctx Ctx) {
			defer wg.Done()

			var err error
			defer m.Manage(ctx, CancellerFunc(cancel), index, &err)
			err = worker(ctx)
		})
	}

	wg.Wait()

	return m.Error()
}

// Group returns a worker that immediately calls the
// Work() function to execute the given group of workers.
func Group(e Executer, m Manager, g ...Worker) Worker {
	return func(ctx Ctx) error {
		return Work(ctx, e, m, g...)
	}
}

// WorkFor arranges for the worker, w, to be executed n times
// and waits for these workers to complete before returning.
// See documention for Work() for details.
func WorkFor(ctx Ctx, n int, e Executer, m Manager, w IdxWorker) error {
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
		e.Execute(ctx, func(ctx Ctx) {
			defer wg.Done()

			var err error
			defer m.Manage(ctx, CancellerFunc(cancel), index, &err)
			err = w(ctx, index)
		})
	}

	wg.Wait()

	return m.Error()
}

// GroupFor returns a worker that immediately calls the
// WorkFor() function to execute the worker n times.
func GroupFor(n int, e Executer, m Manager, w IdxWorker) Worker {
	return func(ctx Ctx) error {
		return WorkFor(ctx, n, e, m, w)
	}
}

// WorkChan arranges for the group of workers provided by channel, g,
// to be executed and waits for the channel to be closed and all
// workers to complete. See documention for Work() for details.
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

	i := 0
	for w := range g {
		wg.Add(1)
		i++
		index := i
		worker := w
		e.Execute(ctx, func(ctx Ctx) {
			defer wg.Done()

			var err error
			defer m.Manage(ctx, CancellerFunc(cancel), index, &err)
			err = worker(ctx)
		})
	}

	wg.Wait()

	return m.Error()
}

// GroupChan returns a worker that immediately calls
// WorkChan to execute the group of workers provided
// by the channel.
func GroupChan(e Executer, m Manager, g <-chan Worker) Worker {
	return func(ctx Ctx) error {
		return WorkChan(ctx, e, m, g)
	}
}
