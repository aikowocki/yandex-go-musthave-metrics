package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := retry.Do(t.Context(), func() error {
		calls++
		return nil
	}, retry.WithDelays(10*time.Millisecond))

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_SuccessAfterRetry(t *testing.T) {
	calls := 0
	err := retry.Do(t.Context(), func() error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}
		return nil
	}, retry.WithDelays(10*time.Millisecond, 10*time.Millisecond, 10*time.Millisecond))

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_AllRetriesFailed(t *testing.T) {
	errPermanent := errors.New("permanent")
	calls := 0
	err := retry.Do(t.Context(), func() error {
		calls++
		return errPermanent
	}, retry.WithDelays(10*time.Millisecond, 10*time.Millisecond))

	assert.ErrorIs(t, err, errPermanent)
	assert.Equal(t, 3, calls) // 1 изначальный + 2 ретрая
}

func TestDo_RetryIfFalse_StopsImmediately(t *testing.T) {
	calls := 0
	errNonRetryable := errors.New("non-retryable")

	err := retry.Do(t.Context(), func() error {
		calls++
		return errNonRetryable
	},
		retry.WithDelays(10*time.Millisecond, 10*time.Millisecond),
		retry.WithRetryIf(func(err error) bool { return false }),
	)

	assert.ErrorIs(t, err, errNonRetryable)
	assert.Equal(t, 1, calls) // не было ретраев
}

func TestDo_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()

	err := retry.Do(ctx, func() error {
		calls++
		return errors.New("fail")
	}, retry.WithDelays(50*time.Millisecond, 50*time.Millisecond))

	assert.ErrorIs(t, err, context.Canceled)
}
