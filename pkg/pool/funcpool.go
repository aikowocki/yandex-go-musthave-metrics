package pool

import "sync"

// FuncPool — универсальный generic-пул для любого типа.
// В отличие от Pool, не требует constraint Reset() — логика очистки
// задаётся через функцию resetFn при создании.
// Подходит для типов вроде gzip.Writer (Reset(w) с аргументом)
type FuncPool[T any] struct {
	pool    sync.Pool
	resetFn func(T)
}

// NewFunc создаёт FuncPool с фабрикой создания и функцией очистки.
func NewFunc[T any](newFn func() T, resetFn func(T)) *FuncPool[T] {
	return &FuncPool[T]{
		pool: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
		resetFn: resetFn,
	}
}

// Get возвращает объект из пула. Если пул пуст — создаёт новый через фабрику.
func (p *FuncPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put вызывает функцию очистки и помещает объект обратно в пул.
func (p *FuncPool[T]) Put(obj T) {
	p.resetFn(obj)
	p.pool.Put(obj)
}
