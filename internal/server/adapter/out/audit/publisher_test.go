package audit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type spyObserver struct {
	mu     sync.Mutex
	events []entity.AuditEvent
}

func (s *spyObserver) Notify(_ context.Context, e entity.AuditEvent) error {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
	return nil
}

func (s *spyObserver) Name() string { return "spy" }

func (s *spyObserver) got() []entity.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]entity.AuditEvent, len(s.events))
	copy(out, s.events)
	return out
}

func TestPublisher_FanOut(t *testing.T) {
	obs1 := &spyObserver{}
	obs2 := &spyObserver{}
	pub := NewPublisher(obs1, obs2)

	event := entity.AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "10.0.0.1",
	}
	pub.Publish(event)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, pub.Close(ctx))

	assert.Equal(t, []entity.AuditEvent{event}, obs1.got())
	assert.Equal(t, []entity.AuditEvent{event}, obs2.got())
}

func TestPublisher_NilObserverIgnored(t *testing.T) {
	obs := &spyObserver{}
	pub := NewPublisher(nil, obs, nil)

	pub.Publish(entity.AuditEvent{Timestamp: 1, Metrics: []string{"m"}, IPAddress: "1.2.3.4"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, pub.Close(ctx))

	assert.Len(t, obs.got(), 1)
}

func TestPublisher_CloseIdempotent(t *testing.T) {
	pub := NewPublisher(&spyObserver{})
	ctx := t.Context()
	require.NoError(t, pub.Close(ctx))
	require.NoError(t, pub.Close(ctx))
}

func TestPublisher_PublishAfterClose(t *testing.T) {
	obs := &spyObserver{}
	pub := NewPublisher(obs)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, pub.Close(ctx))

	// Should not panic or deliver
	pub.Publish(entity.AuditEvent{Timestamp: 1, Metrics: []string{"x"}, IPAddress: "1.1.1.1"})
	assert.Empty(t, obs.got())
}

// erroringObserver всегда возвращает ошибку из Notify — проверяем, что
// dispatch не падает, а логирует и продолжает работу.
type erroringObserver struct {
	mu    sync.Mutex
	calls int
}

func (e *erroringObserver) Notify(context.Context, entity.AuditEvent) error {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()
	return assert.AnError
}

func (e *erroringObserver) Name() string { return "erroring" }

func TestPublisher_NotifyErrorDoesNotCrash(t *testing.T) {
	obs := &erroringObserver{}
	pub := NewPublisher(obs)

	pub.Publish(entity.AuditEvent{Timestamp: 1, Metrics: []string{"m"}, IPAddress: "1.2.3.4"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, pub.Close(ctx))

	obs.mu.Lock()
	defer obs.mu.Unlock()
	assert.Equal(t, 1, obs.calls, "observer should be notified despite returning error")
}
