package workgroup


import (
    "context"
)


func For(g *Group, n int, f func(int) error) {
    for i := 0; i < n; i += 1 {
        idx := i
        g.Go(func() error {
            return f(idx)
        })
    }
}

func ForLimit(g *Group, n int, limit int, f func(int) error) {
    if n <= limit {
        For(g, n, f)
        return
    }

    l := make(chan struct{}, limit)
    push := func() {
        l <- struct{}{}
    }
    pop := func() {
        <- l
    }

    g.Go(func() error {
        for i := 0; i < n; i += 1 {
            push()
            idx := i
            g.Go(func() error {
                defer pop()
                return f(idx)
            })
        }
        return nil
    })
}


type Group struct {
    activate chan struct{}
    work chan func() error
    errs chan error
    done chan error
    cancel func()
}


func New() *Group {
    g := &Group{
      activate: make(chan struct{}, 1),
      work: make(chan func() error),
      done: make(chan error),
    }
    g.reactivate()
    return g
}


func WithErrors() *Group {
    g := New()
    g.errs = make(chan error)
    return g
}


func WithContext(ctx context.Context) (*Group, context.Context) {
    g := New()
    ctx, g.cancel = context.WithCancel(ctx)
    return g, ctx
}


func WithCtxErrors(ctx context.Context) (*Group, context.Context) {
    g, ctx := WithContext(ctx)
    g.errs = make(chan error)
    return g, ctx
}


func (g *Group) Go(f func() error) {
    for {
        select {
        case g.work <- f:
            return
        case <- g.activate:
            go g.process()
        }
    }
}

func (g *Group) Done() <-chan error {
    for {
        select {
        default:
            return g.done
        case <- g.activate:
            go g.process()
        }
    }
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
    fdone := make(chan error)

    finish := g.done
    var first error

    for {
        select {
        case f := <- g.work:
            count += 1
            go func() {
                err := f()
                if g.errs != nil && err != nil {
                    g.errs <- err
                }
                g.done <- err
            }()
            finish = nil

        case err := <- done:
            count -= 1
            if first == nil && err != nil {
                first = err
                if g.cancel != nil {
                    g.cancel()
                }
            }
            if count == 0 {
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
