package concurrency

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/semaphore"
)

// A Tree manages calling a set of functions returning errors, with optional
// concurrency limits. trees can be arranged in a tree.
//
// Panics in functions are recovered and cause the tree to be cancelled.
type Tree struct {
	ctx              context.Context
	cancel           context.CancelCauseFunc
	wg               sync.WaitGroup
	concurrencyLimit *semaphore.Weighted
}

type Option func(*Tree)

// WithConcurrencyLimit sets the maximum number of goroutines that will be
// executed concurrently by the tree before blocking.
func WithConcurrencyLimit(n int) Option {
	return func(o *Tree) {
		o.concurrencyLimit = semaphore.NewWeighted(int64(n))
	}
}

// New creates a new [Tree].
func New(ctx context.Context, options ...Option) (*Tree, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	g := &Tree{ctx: ctx, cancel: cancel}
	for _, option := range options {
		option(g)
	}
	return g, ctx
}

// Go runs fn in a goroutine, and cancels the tree if any function returns an
// error.
//
// The context passed to fn is a child of the context passed to New. A new
// sub-tree can be created from this context by calling treeFromContext.
func (g *Tree) Go(fn func(context.Context) error) {
	g.wg.Add(1)
	go func() {
		defer g.recovery()
		defer g.wg.Done()
		if g.concurrencyLimit != nil {
			if err := g.concurrencyLimit.Acquire(g.ctx, 1); err != nil {
				g.cancel(err)
				return
			}
			defer g.concurrencyLimit.Release(1)
		}
		err := fn(g.ctx)
		if err != nil {
			g.cancel(err)
		}
	}()
}

// Sub creates a new sub-tree and calls fn in a separate goroutine with it.
//
// Wait() is automatically called on the sub-tree when fn returns.
func (g *Tree) Sub(fn func(context.Context, *Tree) error) {
	sub, ctx := New(g.ctx)
	g.wg.Add(1)
	go func() {
		defer g.recovery()
		defer g.wg.Done()
		err := fn(ctx, sub)
		if err != nil {
			g.cancel(err)
		}
		err = sub.Wait()
		if err != nil {
			g.cancel(err)
		}
	}()
}

// Wait for the tree to finish, and return the results of all successful calls.
//
// Results will be returned in the order in which Go() was called. Failing taks
// will leave zero values in the result slice.
//
// Unlike errtree this will return the first error returned by a user function,
// not context.Canceled.
func (g *Tree) Wait() error {
	g.wg.Wait()
	err := g.ctx.Err()
	if err == nil {
		return nil
	} else if errors.Is(err, context.Canceled) && context.Cause(g.ctx) != nil {
		return context.Cause(g.ctx)
	}
	return err
}

func (g *Tree) recovery() {
	if r := recover(); r != nil {
		if err, ok := r.(error); ok {
			g.cancel(err)
		} else {
			g.cancel(fmt.Errorf("worktree: panic: %v", r))
		}
	}
}
