package concurrency

import (
	"context"
	"time"
)

// Map runs fn in tree for each value in values, and returns the results.
//
// Order is preserved. Each call will run in a separate [Tree.Go]() so use
// [WithConcurrencyLimit]() if necessary
func Map[U, T any](tree *Tree, values []U, fn func(context.Context, U) (T, error)) ([]T, error) {
	out := make([]T, len(values))
	for i, value := range values {
		i, value := i, value
		tree.Go(func(ctx context.Context) error {
			result, err := fn(ctx, value)
			if err != nil {
				return err
			}
			out[i] = result
			return nil
		})
	}
	return out, tree.Wait()
}

// Schedule calls fn every time interval until it returns an error or the
// context is cancelled.
func Schedule(tree *Tree, fn func(context.Context) (time.Duration, error)) error {
	tree.Go(func(ctx context.Context) error {
		var delay time.Duration
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-time.After(delay):
				var err error
				delay, err = fn(ctx)
				if err != nil {
					return err
				}
			}
		}
	})
	return nil
}

// Call runs fn in a separate goroutine and returns a context that will cancel
// when the function completes.
func Call(ctx context.Context, fn func() error) context.Context {
	ctx, cancel := context.WithCancelCause(ctx)
	go func() {
		cancel(fn())
	}()
	return ctx
}
