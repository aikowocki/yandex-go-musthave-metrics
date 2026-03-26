package retry

import (
	"context"
	"time"
)

type config struct {
	delays  []time.Duration
	retryIf func(error) bool
}

type Option func(*config)

func WithDelays(delays ...time.Duration) Option {
	return func(c *config) {
		c.delays = delays
	}
}

func WithRetryIf(fn func(error) bool) Option {
	return func(c *config) {
		c.retryIf = fn
	}
}

func Do(ctx context.Context, operation func() error, opts ...Option) error {
	cfg := &config{
		delays:  []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second},
		retryIf: func(err error) bool { return true },
	}
	for _, opt := range opts {
		opt(cfg)
	}

	err := operation()
	if err != nil {
		for _, delay := range cfg.delays {
			if !cfg.retryIf(err) {
				return err
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}

			if err = operation(); err == nil {
				return nil
			}
		}
	}
	return err
}
