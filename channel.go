package concurrency

import (
	"context"
)

// Channel[T] utilises a tree to produce values and send them to a channel.
type Channel[T any] struct {
	tree *Tree
	dest chan<- T
}

// ToChannel creates a new Channel[T] instance.
func ToChannel[T any](ctx context.Context, dest chan<- T, options ...Option) (*Channel[T], context.Context) {
	tree, ctx := New(ctx, options...)
	return &Channel[T]{tree: tree, dest: dest}, ctx
}

func (v *Channel[T]) Go(fn func(context.Context) (T, error)) {
	v.tree.Go(func(ctx context.Context) error {
		value, err := fn(ctx)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()

		case v.dest <- value:
			return nil
		}
	})
}

func (v *Channel[T]) Sub(fn func(context.Context, *Channel[T]) error) {
	v.tree.Sub(func(ctx context.Context, sg *Tree) error {
		sub := &Channel[T]{tree: sg, dest: v.dest}
		return fn(ctx, sub)
	})
}

func (v *Channel[T]) Wait() error {
	return v.tree.Wait()
}
