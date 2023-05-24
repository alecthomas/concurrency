package concurrency

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func TestConcurrencyLimit(t *testing.T) {
	t.Parallel()
	wg, _ := New(context.Background(), WithConcurrencyLimit(2))
	start := time.Now()
	for i := 0; i < 10; i++ {
		wg.Go(func(ctx context.Context) error {
			time.Sleep(time.Millisecond * 10)
			return nil
		})
	}
	err := wg.Wait()
	assert.NoError(t, err)
	assert.True(t, time.Since(start) > time.Millisecond*50, "%s elapsed", time.Since(start))
}

func TestTree(t *testing.T) {
	t.Parallel()
	results := make(chan string, 3)
	wg, _ := ToChannel(context.Background(), results)
	wg.Go(func(ctx context.Context) (string, error) {
		return "hello", nil
	})
	wg.Sub(func(ctx context.Context, sg *Channel[string]) error {
		sg.Go(func(ctx context.Context) (string, error) {
			return "what's", nil
		})
		sg.Go(func(ctx context.Context) (string, error) {
			return "up", nil
		})
		return nil
	})
	err := wg.Wait()
	close(results)
	assert.NoError(t, err)
	actual := []string{}
	for value := range results {
		actual = append(actual, value)
	}
	sort.Strings(actual)
	assert.Equal(t, []string{"hello", "up", "what's"}, actual)
}

func TestMap(t *testing.T) {
	t.Parallel()
	wg, _ := New(context.Background())
	results, err := Map(wg, []string{"hello", "what's", "up"}, func(ctx context.Context, s string) (string, error) {
		return strings.ToUpper(s), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"HELLO", "WHAT'S", "UP"}, results)
}

func TestError(t *testing.T) {
	t.Parallel()
	wg, _ := New(context.Background())
	wg.Go(func(ctx context.Context) error {
		return fmt.Errorf("error")
	})
	err := wg.Wait()
	assert.EqualError(t, err, "error")
}

func TestSubtreeError(t *testing.T) {
	t.Parallel()
	wg, _ := New(context.Background())
	wg.Sub(func(ctx context.Context, sg *Tree) error {
		sg.Go(func(ctx context.Context) error {
			return fmt.Errorf("error")
		})
		return nil
	})
	err := wg.Wait()
	assert.EqualError(t, err, "error")
}

func TestCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	wg, _ := New(ctx)
	wg.Go(func(ctx context.Context) error {
		time.Sleep(time.Millisecond * 100)
		return nil
	})
	cancel()
	err := wg.Wait()
	assert.EqualError(t, err, "context canceled")
}

func TestCancelCause(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancelCause(context.Background())
	wg, _ := New(ctx)
	wg.Go(func(ctx context.Context) error {
		time.Sleep(time.Millisecond * 100)
		return nil
	})
	cancel(fmt.Errorf("error"))
	err := wg.Wait()
	assert.EqualError(t, err, "error")
}
