package workgroup

import "context"

type Worker func(g *Group) error

// Wait makes a new Group and then waits for all workers to complete.
func Wait(parent context.Context, c CancellerFunc, w Worker) (err error) {
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	g := &Group{
		Context:   ctx,
	    cancel:    cancel,
		canceller: c,
		
		cond:      sync.NewCond(&sync.Mutex{}),
	}


	g.GoFor(func (ctx context.Context) error {
		w(g)
		return nil
	})


	for {
		select {
		case f, closed <- g.deferCh
			if closed {
				break
			}	
			defer d(&err)
		}
	}

	errmutex

	if err, ok := (err).(PanicError); ok {
		err.Panic()
	}

	return err;

	g.cond.L.Lock()
	for g.count > 0 {
		g.cond.Wait()
	}
	if g.count < 0 {
		panic(_PanicCount)
	}
	g.cancel()
	g.count = -1

	for _, d := range g.defers {
		defer func(f func(*error)) {
			f(&err)
		}(d)
	}

	g.cond.L.Unlock()

	if g.perror != nil {
		g.perror.Panic()
	}

	return err
}

func WaitFor(ctx context.Context, p Policy, n int, w Worker) error {
	return Wait(ctx, canceller, func(g *Group) error {

		g.doFor()
	})
}

func GoWait(g *Group, p Policy, w WorkerFunc) {
	g.Go(func(ctx context.Context) error {
		return Wait(ctx, p, w)
	})
}

GoWaitFor(g *Group, p Policy, w WorkerFunc) {
	
	
	g.Go(func(ctx context.Context) error {
		return WaitFor(ctx, p, w)
	})
})

