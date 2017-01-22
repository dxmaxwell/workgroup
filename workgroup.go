package workgroup

import (
    "sync"
    "golang.org/x/net/context"
    "golang.org/x/sync/errgroup"
)

type Group struct {
    errgroup.Group
}

func WithContext(ctx context.Context) (*Group, context.Context) {
    g, c := errgroup.WithContext(ctx)
    return &Group{*g}, c
}


func (g *Group) GoFor(n int, f func(int) error) {
    for idx := 0; idx < n; idx += 1 {
        g.Go(func() error {
            return f(idx)
        })
    }
}


func (g *Group) GoForLimit(n, limit int, f func(int) error) {
    if limit >= n || limit <= 0 {
        g.GoFor(n, f)
        return
    }
    cond := sync.NewCond(&sync.Mutex{})
    g.Go(func() error {
        cond.L.Lock()
        defer cond.L.Unlock()
        for idx := 0; idx < n; idx += 1 {
            for limit == 0 {
                cond.Wait()
            }
            limit -= 1
            g.Go(func() error {
                err := f(idx)
                cond.L.Lock()
                defer cond.L.Unlock()
                limit += 1
                cond.Signal()
                return err
            })
        }
        return nil
    })
    return
}
