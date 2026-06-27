package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_SmallMessage(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	original := []byte("hello, world")
	encrypted, err := Encrypt(&key.PublicKey, original)
	require.NoError(t, err)

	decrypted, err := Decrypt(key, encrypted)
	require.NoError(t, err)

	assert.Equal(t, original, decrypted)
}

func TestEncryptDecrypt_LargeMessage(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	// Сообщение больше одного RSA-блока (проверяем что гибридная схема работает).
	original := make([]byte, 2048)
	_, err = rand.Read(original)
	require.NoError(t, err)

	encrypted, err := Encrypt(&key.PublicKey, original)
	require.NoError(t, err)

	decrypted, err := Decrypt(key, encrypted)
	require.NoError(t, err)

	assert.Equal(t, original, decrypted)
}

func TestEncryptDecrypt_EmptyMessage(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	encrypted, err := Encrypt(&key.PublicKey, []byte{})
	require.NoError(t, err)

	decrypted, err := Decrypt(key, encrypted)
	require.NoError(t, err)

	assert.Empty(t, decrypted)
}
