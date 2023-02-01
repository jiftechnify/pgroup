package pgroup

import (
	"context"
	"sync"
)

type Group struct {
	wg sync.WaitGroup

	err     error
	errOnce sync.Once

	ctx    context.Context
	cancel func()
}

type Promise[T any] struct {
	res T
}

func (p *Promise[T]) Get() T {
	return p.res
}

func New() *Group {
	return WithContext(context.Background())
}

func WithContext(ctx context.Context) *Group {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (pg *Group) Wait() error {
	pg.wg.Wait()
	pg.cancel()

	return pg.err
}

func Go[T any](pg *Group, f func(ctx context.Context) (T, error)) *Promise[T] {
	pg.wg.Add(1)
	p := &Promise[T]{}

	run := func() {
		defer pg.wg.Done()

		res, err := f(pg.ctx)
		if err != nil {
			pg.errOnce.Do(func() {
				pg.err = err
				pg.cancel()
			})
			return
		}
		p.res = res
	}
	go run()

	return p
}

func GoAndForget(pg *Group, f func(ctx context.Context) error) {
	pg.wg.Add(1)

	run := func() {
		defer pg.wg.Done()

		if err := f(pg.ctx); err != nil {
			pg.errOnce.Do(func() {
				pg.err = err
				pg.cancel()
			})
		}
	}
	go run()
}
