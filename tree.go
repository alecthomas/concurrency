package concurrency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

func NoJitter() time.Duration { return 0 }

// A Waiter is a type that can wait for completion.
type Waiter interface {
	Wait() error
}

// A Tree manages calling a set of functions returning errors, with optional
// concurrency limits. trees can be arranged in a tree.
//
// Panics in functions are recovered and cause the tree to be cancelled.
type Tree struct {
	ctx              context.Context //nolint: containedctx
	cancel           context.CancelCauseFunc
	wg               sync.WaitGroup
	options          []Option
	concurrencyLimit *semaphore.Weighted
	jitter           func() time.Duration
}

type Option func(*Tree)

// WithJitter sets the jitter function used to delay the start of each goroutine.
func WithJitter(fn func() time.Duration) Option {
	return func(o *Tree) {
		o.jitter = fn
	}
}

// WithConcurrencyLimit sets the maximum number of goroutines that will be
// executed concurrently by the tree before blocking.
//
// A value of 0 disables the limit.
func WithConcurrencyLimit(n int) Option {
	return func(o *Tree) {
		if n == 0 {
			o.concurrencyLimit = nil
		} else {
			o.concurrencyLimit = semaphore.NewWeighted(int64(n))
		}
	}
}

// New creates a new [Tree].
func New(ctx context.Context, options ...Option) (*Tree, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	g := &Tree{ctx: ctx, cancel: cancel, options: options, jitter: NoJitter}
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
		time.Sleep(g.jitter())
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

// Link an existing Waiter to the tree.
//
// Useful for eg. syncing on an errgroup, or a separate Tree.
func (g *Tree) Link(waiter Waiter) {
	g.wg.Add(1)
	go func() {
		defer g.recovery()
		defer g.wg.Done()
		err := waiter.Wait()
		if err != nil {
			g.cancel(err)
		}
	}()
}

// Sub calls fn in a new goroutine with a new sub-tree.
//
// The sub-tree will inherit the options of the parent tree, but can override
// them.
//
// Wait() is automatically called on the sub-tree when fn returns.
func (g *Tree) Sub(fn func(context.Context, *Tree) error, options ...Option) {
	options = append(g.options, options...)
	sub, ctx := New(g.ctx, options...)
	g.wg.Add(1)
	go func() {
		defer g.recovery()
		defer g.wg.Done()
		time.Sleep(g.jitter())
		err := fn(ctx, sub)
		cancelled := false
		if err != nil {
			g.cancel(err)
			cancelled = true
		}
		err = sub.Wait()
		if err != nil && !cancelled {
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
