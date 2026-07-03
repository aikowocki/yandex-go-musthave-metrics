package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
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

func TestLoadPublicKey(t *testing.T) {
	t.Run("valid PEM file", func(t *testing.T) {
		// Генерируем тестовый ключ
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Создаём временный файл с публичным ключом
		tmpFile := t.TempDir() + "/public.pem"
		publicKeyBytes := x509.MarshalPKCS1PublicKey(&key.PublicKey)
		pemBlock := &pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: publicKeyBytes,
		}
		err = os.WriteFile(tmpFile, pem.EncodeToMemory(pemBlock), 0600)
		require.NoError(t, err)

		// Загружаем ключ
		loadedKey, err := LoadPublicKey(tmpFile)
		require.NoError(t, err)
		assert.NotNil(t, loadedKey)
		assert.Equal(t, key.PublicKey.N, loadedKey.N)
		assert.Equal(t, key.PublicKey.E, loadedKey.E)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadPublicKey("/nonexistent/path/key.pem")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "чтение файла публичного ключа")
	})

	t.Run("invalid PEM format", func(t *testing.T) {
		tmpFile := t.TempDir() + "/invalid.pem"
		err := os.WriteFile(tmpFile, []byte("not a valid PEM"), 0600)
		require.NoError(t, err)

		_, err = LoadPublicKey(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось декодировать PEM-блок")
	})

	t.Run("invalid key data", func(t *testing.T) {
		tmpFile := t.TempDir() + "/bad_key.pem"
		pemBlock := &pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: []byte("invalid key bytes"),
		}
		err := os.WriteFile(tmpFile, pem.EncodeToMemory(pemBlock), 0600)
		require.NoError(t, err)

		_, err = LoadPublicKey(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "парсинг публичного ключа")
	})
}

func TestLoadPrivateKey(t *testing.T) {
	t.Run("valid PEM file", func(t *testing.T) {
		// Генерируем тестовый ключ
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Создаём временный файл с приватным ключом
		tmpFile := t.TempDir() + "/private.pem"
		privateKeyBytes := x509.MarshalPKCS1PrivateKey(key)
		pemBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		}
		err = os.WriteFile(tmpFile, pem.EncodeToMemory(pemBlock), 0600)
		require.NoError(t, err)

		// Загружаем ключ
		loadedKey, err := LoadPrivateKey(tmpFile)
		require.NoError(t, err)
		assert.NotNil(t, loadedKey)
		assert.Equal(t, key.N, loadedKey.N)
		assert.Equal(t, key.D, loadedKey.D)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadPrivateKey("/nonexistent/path/key.pem")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "чтение файла приватного ключа")
	})

	t.Run("invalid PEM format", func(t *testing.T) {
		tmpFile := t.TempDir() + "/invalid.pem"
		err := os.WriteFile(tmpFile, []byte("not a valid PEM"), 0600)
		require.NoError(t, err)

		_, err = LoadPrivateKey(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось декодировать PEM-блок")
	})

	t.Run("invalid key data", func(t *testing.T) {
		tmpFile := t.TempDir() + "/bad_key.pem"
		pemBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: []byte("invalid key bytes"),
		}
		err := os.WriteFile(tmpFile, pem.EncodeToMemory(pemBlock), 0600)
		require.NoError(t, err)

		_, err = LoadPrivateKey(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "парсинг приватного ключа")
	})
}

func TestLoadKeys_Integration(t *testing.T) {
	// Генерируем ключевую пару
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpDir := t.TempDir()

	// Сохраняем публичный ключ
	publicFile := tmpDir + "/public.pem"
	publicKeyBytes := x509.MarshalPKCS1PublicKey(&key.PublicKey)
	publicPEM := &pem.Block{Type: "RSA PUBLIC KEY", Bytes: publicKeyBytes}
	err = os.WriteFile(publicFile, pem.EncodeToMemory(publicPEM), 0600)
	require.NoError(t, err)

	// Сохраняем приватный ключ
	privateFile := tmpDir + "/private.pem"
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(key)
	privatePEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}
	err = os.WriteFile(privateFile, pem.EncodeToMemory(privatePEM), 0600)
	require.NoError(t, err)

	// Загружаем ключи
	publicKey, err := LoadPublicKey(publicFile)
	require.NoError(t, err)

	privateKey, err := LoadPrivateKey(privateFile)
	require.NoError(t, err)

	// Проверяем что можем зашифровать/расшифровать с загруженными ключами
	message := []byte("test message")
	encrypted, err := Encrypt(publicKey, message)
	require.NoError(t, err)

	decrypted, err := Decrypt(privateKey, encrypted)
	require.NoError(t, err)

	assert.Equal(t, message, decrypted)
}
