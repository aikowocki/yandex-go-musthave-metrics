// Package pool generic-обёртка над sync.Pool с constraint
// на наличие метода Reset().
package pool

import "sync"

// Resetter — constraint для типов, поддерживающих сброс состояния.
type Resetter interface {
	Reset()
}

// Pool — generic-контейнер для переиспользования объектов одного типа.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт и возвращает указатель на Pool.
// Паникует, если newFn равен nil
func New[T Resetter](newFunc func() T) *Pool[T] {
	if newFunc == nil {
		panic("pool: New requires a non-nil newFunc")
	}
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return newFunc()
			},
		},
	}
}

// Get возвращает объект из пула. Если пул пуст — создаёт новый через фабрику.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put сбрасывает объект через Reset() и помещает его обратно в пул
// для повторного использования.
func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
