package retry

import (
	"context"
	"errors"
	"time"

	memapp "github.com/iWorld-y/owui-memory-daemon/internal/memoryops/application"
)

type Policy struct {
	MaxAttempts int
	// Delays is used between attempts: attempt1->attempt2 uses Delays[0], etc.
	Delays []time.Duration
}

func Do(ctx context.Context, p Policy, fn func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := fn(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt == p.MaxAttempts {
			break
		}
		delayIdx := attempt - 1
		var d time.Duration
		if delayIdx >= 0 && delayIdx < len(p.Delays) {
			d = p.Delays[delayIdx]
		}
		if d > 0 {
			t := time.NewTimer(d)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
		}
	}

	if lastErr == nil {
		return errors.New("retry: no attempts executed")
	}
	return lastErr
}

// Adapter implements memoryops/application.RetryPort.
type Adapter struct{ Policy Policy }

func (a Adapter) Do(ctx context.Context, fn func(context.Context) error) error {
	return Do(ctx, a.Policy, fn)
}

var _ memapp.RetryPort = Adapter{}
