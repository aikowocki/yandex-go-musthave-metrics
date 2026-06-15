// Package audit implements the Observer pattern for audit event delivery.
package audit

import (
	"context"
	"sync"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	"go.uber.org/zap"
)

const queueSize = 1024

type subscription struct {
	observer port.AuditObserver
	ch       chan entity.AuditEvent
}

// Publisher рассылает события наблюдателям через горутины для каждого наблюдателя.
type Publisher struct {
	subs []*subscription
	wg   sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	closed bool
}

func NewPublisher(observers ...port.AuditObserver) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Publisher{ctx: ctx, cancel: cancel}
	for _, o := range observers {
		if o == nil {
			continue
		}
		sub := &subscription{
			observer: o,
			ch:       make(chan entity.AuditEvent, queueSize),
		}
		p.subs = append(p.subs, sub)
		p.wg.Add(1)
		go p.dispatch(sub)
	}
	return p
}

// Publish отправляет событие всем наблюдателям. Неблокирующий метод;
// при переполнении сбрасывает данные.
func (p *Publisher) Publish(event entity.AuditEvent) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	for _, sub := range p.subs {
		select {
		case sub.ch <- event:
		default:
			zap.S().Warnw("audit: queue full, event dropped", "observer", sub.observer.Name())
		}
	}
}

func (p *Publisher) Close(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	for _, sub := range p.subs {
		close(sub.ch)
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.cancel()
		return nil
	case <-ctx.Done():
		// Дедлайн shutdown истёк: форсируем отмену контекста паблишера,
		// чтобы прервать зависшие Notify-вызовы у наблюдателей.
		p.cancel()
		return ctx.Err()
	}
}

func (p *Publisher) dispatch(sub *subscription) {
	defer p.wg.Done()
	for event := range sub.ch {
		if err := sub.observer.Notify(p.ctx, event); err != nil {
			zap.S().Warnw("audit: notify failed", "observer", sub.observer.Name(), "err", err)
		}
	}
}
