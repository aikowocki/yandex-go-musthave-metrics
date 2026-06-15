package hash_test

import (
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeHMAC(t *testing.T) {
	key := "secret"
	data := []byte("hello world")

	result := hash.ComputeHMAC(key, data)
	require.NotEmpty(t, result)

	// Детерминированность: один и тот же ввод → один и тот же вывод.
	assert.Equal(t, result, hash.ComputeHMAC(key, data))

	// Разные ключи → разный результат.
	assert.NotEqual(t, result, hash.ComputeHMAC("other", data))

	// Разные данные → разный результат.
	assert.NotEqual(t, result, hash.ComputeHMAC(key, []byte("other")))
}

func TestValidateHMAC_Valid(t *testing.T) {
	key := "secret"
	data := []byte("test data")

	mac := hash.ComputeHMAC(key, data)
	assert.True(t, hash.ValidateHMAC(key, data, mac))
}

func TestValidateHMAC_InvalidHex(t *testing.T) {
	assert.False(t, hash.ValidateHMAC("key", []byte("data"), "not-hex!"))
}

func TestValidateHMAC_WrongKey(t *testing.T) {
	key := "secret"
	data := []byte("test data")

	mac := hash.ComputeHMAC(key, data)
	assert.False(t, hash.ValidateHMAC("wrong-key", data, mac))
}

func TestValidateHMAC_TamperedData(t *testing.T) {
	key := "secret"
	data := []byte("original")

	mac := hash.ComputeHMAC(key, data)
	assert.False(t, hash.ValidateHMAC(key, []byte("tampered"), mac))
}
