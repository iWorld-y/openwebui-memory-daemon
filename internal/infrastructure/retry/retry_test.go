package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoRetriesThenSucceeds(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	attempts := 0
	err := Do(ctx, Policy{MaxAttempts: 3, Delays: []time.Duration{0, 0}}, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("fail")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts=%d want 3", attempts)
	}
}

func TestDoStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	err := Do(ctx, Policy{MaxAttempts: 3, Delays: []time.Duration{0, 0}}, func(ctx context.Context) error {
		attempts++
		return errors.New("fail")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if attempts != 0 {
		t.Fatalf("attempts=%d want 0", attempts)
	}
}
