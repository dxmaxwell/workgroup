package workgroup


import (
    "context"
    "container/list"
)

type loopWork struct {
    f func(int) error
    total int
    next int
}

type sliceWork struct {
    f []func() error
    next int
}

type Group struct {
    limit int
    activate chan struct{}
    work chan interface{}
    errs chan error
    done chan error
    cancel func()
}


func New() *Group {
    return WithLimit(0)
}


func WithLimit(limit int) *Group {
    g := &Group{
      limit: limit,
      activate: make(chan struct{}, 1),
      work: make(chan interface{}),
      done: make(chan error),
    }
    g.reactivate()
    return g
}


func WithErrors(limit int) *Group {
    g := WithLimit(limit)
    g.errs = make(chan error)
    return g
}


func WithContext(ctx context.Context, limit int, errors bool) (*Group, context.Context) {
    var wg *Group
    if errors {
      wg = WithErrors(limit)
    } else {
      wg = WithLimit(limit)
    }
    ctx, wg.cancel = context.WithCancel(ctx)
    //wg.cancel = cancel
    return wg, ctx
}


func (g *Group) Go(f ...func() error) {
    if len(f) == 0 {
        return
    }
    w := &sliceWork{ f:f }
    for {
        select {
        case g.work <- w:
            return
        case <- g.activate:
            go g.process()
        }
    }
}


func (g *Group) GoN(n int, f func(int) error) {
    if n < 1 {
      return
    }
    w := &loopWork{ f:f, total:n }
    for {
        select {
        case g.work <- w:
            return
        case <- g.activate:
            go g.process()
        }
    }
}

func (g *Group) Done() <-chan error {
    return g.done
}

func (g *Group) Errors() <-chan error {
    return g.errs
}

func (g *Group) reactivate() {
    g.activate <- struct{}{}
}

func (g *Group) process() {
    defer g.reactivate()

    count := 0
    work := list.New()
    done := make(chan error)

    dispatch := func(w interface{}) bool {
        if w == nil {
            count -= 1
        } else {
            work.PushBack(w)
        }
        for (g.limit <= 0 || g.limit > count) && work.Len() > 0 {
            e := work.Front()
            switch w := e.Value.(type) {
            case *sliceWork:
                f := w.f[w.next]
                go func() {
                    err := f()
                    if g.errs != nil && err != nil {
                        g.errs <- err
                    }
                    done <- err
                }()
                count += 1
                w.next += 1
                if w.next == len(w.f) {
                    work.Remove(e)
                }

            case *loopWork:
                f := w.f
                n := w.next
                go func() {
                    err := f(n)
                    if g.errs != nil && err != nil {
                        g.errs <- err
                    }
                    done <- err
                }()
                count += 1
                w.next += 1
                if w.next == w.total {
                    work.Remove(e)
                }

            default:
                panic("expecting type *loopWork or *sliceWork")
            }
        }
        return (count > 0)
    }

    var first error
    var finish chan error

    for {
        select {
        case w := <- g.work:
            if dispatch(w) {
                finish = nil
            }

        case err := <- done:
            if first == nil && err != nil {
                first = err
                if g.cancel != nil {
                    g.cancel()
                }
            }
            if !dispatch(nil) {
                finish = g.done
            }

        case finish <- first:
            if g.cancel != nil {
                g.cancel()
            }
            return
        }
    }
}
