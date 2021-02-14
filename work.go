package workgroup

import (
	"context"
	"sync"
)

// Worker is a function that performs work
type Worker func(context.Context) error

// IdxWorker is a function that performs work with for a given index
type IdxWorker func(context.Context, int) error

// Work arranges for a group of workers to be executed
// and then waits for these workers to complete.
// The executer, e, is responsible for executing these workers
// with various levels of concurrancy. The manager, m, determines
// when the context will be canceled and which error is returned.
// If executer, e, is not provided then DefaultExecuter
// is called to obtain the default. If manager, m, is not provied
// then DefaultManager is called be obtain the default manager.
func Work(ctx context.Context, e Executer, m Manager, g ...Worker) error {
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
		e.Execute(ctx, func(ctx context.Context) {
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
	return func(ctx context.Context) error {
		return Work(ctx, e, m, g...)
	}
}

// WorkFor arranges for the worker, w, to be executed n times
// and waits for these workers to complete before returning.
// See documention for Work() for details.
func WorkFor(ctx context.Context, e Executer, m Manager, n int, w IdxWorker) error {
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
		e.Execute(ctx, func(ctx context.Context) {
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
func GroupFor(e Executer, m Manager, n int, w IdxWorker) Worker {
	return func(ctx context.Context) error {
		return WorkFor(ctx, e, m, n, w)
	}
}

// WorkChan arranges for the group of workers provided by channel, g,
// to be executed and waits for the channel to be closed and all
// workers to complete. See documention for Work() for details.
func WorkChan(ctx context.Context, e Executer, m Manager, g <-chan Worker) error {
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
		e.Execute(ctx, func(ctx context.Context) {
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
	return func(ctx context.Context) error {
		return WorkChan(ctx, e, m, g)
	}
}
