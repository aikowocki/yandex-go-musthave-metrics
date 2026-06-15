package pool_test

import (
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStruct — тестовая структура, реализующая Resetter.
type testStruct struct {
	Name  string
	Count int
}

func (t *testStruct) Reset() {
	t.Name = ""
	t.Count = 0
}

func TestPool_GetPut(t *testing.T) {
	p := pool.New(func() *testStruct {
		return &testStruct{}
	})

	// Get возвращает чистый объект.
	obj := p.Get()
	require.Equal(t, "", obj.Name)
	require.Equal(t, 0, obj.Count)

	// Мутируем и возвращаем в пул.
	obj.Name = "hello"
	obj.Count = 42
	p.Put(obj)

	// Get повторно — должен быть сброшен.
	obj2 := p.Get()
	assert.Equal(t, "", obj2.Name)
	assert.Equal(t, 0, obj2.Count)
}

func TestPool_ConstraintEnforced(t *testing.T) {
	p := pool.New(func() *testStruct {
		return &testStruct{Name: "initial"}
	})

	obj := p.Get()
	require.Equal(t, "initial", obj.Name)
	p.Put(obj)
}

func TestFuncPool_GetPut(t *testing.T) {
	p := pool.NewFunc(
		func() *testStruct { return &testStruct{} },
		func(ts *testStruct) { ts.Name = ""; ts.Count = 0 },
	)

	obj := p.Get()
	require.Equal(t, "", obj.Name)
	require.Equal(t, 0, obj.Count)

	obj.Name = "world"
	obj.Count = 99
	p.Put(obj)

	obj2 := p.Get()
	assert.Equal(t, "", obj2.Name)
	assert.Equal(t, 0, obj2.Count)
}
