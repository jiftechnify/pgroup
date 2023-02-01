package pgroup

import (
	"context"
	"sync"
)

// Group is a collection of goroutines (or, "tasks") in the same cancellation scope.
// When any task in a Group returned error, all other tasks in the Group are canceled immediately.
// When the parent context of a Group is canceled, all tasks in the Group are also canceled.
//
// You can launch tasks on a Group which get some result values, or which perform some side-effects returning no result.
type Group struct {
	wg sync.WaitGroup

	err     error
	errOnce sync.Once

	ctx    context.Context
	cancel func()
}

// New returns a new Group whose parent context is an empty context.
func New() *Group {
	return WithContext(context.Background())
}

// WithContext returns a new Group with the "parent context".
// When the parent context is canceled, all tasks run in the Group will be canceled.
func WithContext(ctx context.Context) *Group {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Wait blocks until all tasks have completed or canceled.
//
// Promise.Get returns meaningful value only after the call to Wait() returned nil (no error).
func (pg *Group) Wait() error {
	pg.wg.Wait()
	pg.cancel()

	return pg.err
}

// Go launches the given function in a new goroutine to get some result.
// Result of the function will be available via the Promise returned after the call to Group's Wait() returned nil (no error).
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

// GoAndForget launches the given function in a new goroutine to perform some side-effects.
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

// Promise is a place for the result of a task that will be available at some point.
type Promise[T any] struct {
	res T
}

// Get returns the result of the corresponding task.
//
// It should be called after the relevant Group's Wait() returned nil (no error). There is no guarantees about its return value if it called before Wait()-ing on the Group or after the Group's Wait() returned non-nil error.
func (p *Promise[T]) Get() T {
	return p.res
}
