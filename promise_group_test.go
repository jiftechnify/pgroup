package pgroup

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type counter struct {
	cnt uint32
}

func (r *counter) incr() {
	atomic.AddUint32(&r.cnt, 1)
}

func TestGoAndForget(t *testing.T) {
	c := &counter{cnt: 0}

	task := delayedTask(time.Second, func() error { c.incr(); return nil })

	pg := New()

	GoAndForget(pg, task)
	GoAndForget(pg, task)
	GoAndForget(pg, task)

	if err := pg.Wait(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.cnt != 3 {
		t.Fatalf("unexpected result (want: %v, got: %v)", 3, c.cnt)
	}
}

func TestGoAndForget_err(t *testing.T) {
	task := delayedTask(time.Second, func() error { return nil })

	errExp := errors.New("error!")
	etask := delayedTask(time.Second, func() error { return errExp })

	e2task := delayedTask(2*time.Second, func() error { return errors.New("too late error") })

	pg := New()

	GoAndForget(pg, task)
	GoAndForget(pg, task)
	GoAndForget(pg, etask)
	GoAndForget(pg, e2task)

	err := pg.Wait()
	if err == nil {
		t.Fatal("error is expected")
	}
	if err != errExp {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGoAndForget_timeoutParentCtx(t *testing.T) {
	task := delayedTask(2*time.Second, func() error { return nil })
	etask := delayedTask(2*time.Second, func() error { return errors.New("never seen error") })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pg := WithContext(ctx)

	GoAndForget(pg, task)
	GoAndForget(pg, task)
	GoAndForget(pg, etask)

	err := pg.Wait()
	if err == nil {
		t.Fatal("error is expected")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGo(t *testing.T) {
	intTask := delayedResultTask(time.Second, func() (int, error) { return 42, nil })
	strTask := delayedResultTask(time.Second, func() (string, error) { return "result", nil })

	pg := New()

	intPromise := Go(pg, intTask)
	strPromise := Go(pg, strTask)

	err := pg.Wait()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if intPromise.Get() != 42 {
		t.Fatalf("unexpected int result (want: %v, got: %v)", 42, intPromise.Get())
	}
	if strPromise.Get() != "result" {
		t.Fatalf("unexpected str result (want: %v, got: %v)", "result", strPromise.Get())
	}
}

func TestGo_err(t *testing.T) {
	intTask := delayedResultTask(time.Second, func() (int, error) { return 42, nil })

	errExp := errors.New("error!")
	etask := delayedResultTask(time.Second, func() (int, error) { return 0, errExp })

	e2task := delayedResultTask(2*time.Second, func() (int, error) { return 0, errors.New("too late error") })

	pg := New()

	_ = Go(pg, intTask)
	_ = Go(pg, etask)
	_ = Go(pg, e2task)

	err := pg.Wait()
	if err == nil {
		t.Fatal("error is expected")
	}
	if err != errExp {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGo_timeoutParentCtx(t *testing.T) {
	intTask := delayedResultTask(2*time.Second, func() (int, error) { return 42, nil })
	etask := delayedResultTask(2*time.Second, func() (int, error) { return 0, errors.New("never seen error") })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pg := WithContext(ctx)

	_ = Go(pg, intTask)
	_ = Go(pg, etask)

	err := pg.Wait()
	if err == nil {
		t.Fatal("error is expected")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
}

func delayedTask(delay time.Duration, f func() error) func(context.Context) error {
	return func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			return f()
		}
	}
}

func delayedResultTask[T any](delay time.Duration, f func() (T, error)) func(context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(delay):
			return f()
		}
	}
}
