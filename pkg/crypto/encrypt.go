package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Encrypt выполняет гибридное шифрование:
// 1. Генерирует случайный AES-256 ключ.
// 2. Шифрует plaintext с помощью AES-GCM.
// 3. Шифрует AES-ключ с помощью RSA-OAEP.
//
// Формат выходных данных:
//
//	[4 байта: длина зашифрованного ключа (big-endian)][зашифрованный AES-ключ][nonce][ciphertext+tag]
func Encrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	// Генерируем случайный AES-256 ключ.
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, fmt.Errorf("crypto: генерация AES-ключа: %w", err)
	}

	// Шифруем plaintext с помощью AES-GCM.
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: создание AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: создание GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: генерация nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Шифруем AES-ключ с помощью RSA-OAEP.
	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: RSA шифрование ключа: %w", err)
	}

	// Собираем результат: [keyLen(4)][encryptedKey][nonce][ciphertext]
	keyLen := len(encryptedKey)
	result := make([]byte, 4+keyLen+len(nonce)+len(ciphertext))
	binary.BigEndian.PutUint32(result[0:4], uint32(keyLen))
	copy(result[4:4+keyLen], encryptedKey)
	copy(result[4+keyLen:4+keyLen+len(nonce)], nonce)
	copy(result[4+keyLen+len(nonce):], ciphertext)

	return result, nil
}

// Decrypt выполняет гибридную расшифровку:
// 1. Извлекает и расшифровывает AES-ключ с помощью RSA-OAEP.
// 2. Расшифровывает payload с помощью AES-GCM.
func Decrypt(pk *rsa.PrivateKey, data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, errors.New("crypto: повреждённый пакет — отсутствует заголовок")
	}

	keyLen := int(binary.BigEndian.Uint32(data[0:4]))
	if len(data) < 4+keyLen {
		return nil, errors.New("crypto: повреждённый пакет — отсутствует зашифрованный ключ")
	}

	encryptedKey := data[4 : 4+keyLen]
	rest := data[4+keyLen:]

	// Расшифровываем AES-ключ.
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, pk, encryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: RSA расшифровка ключа: %w", err)
	}

	// Расшифровываем payload с помощью AES-GCM.
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: создание AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: создание GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(rest) < nonceSize {
		return nil, errors.New("crypto: повреждённый пакет — отсутствует nonce")
	}

	nonce := rest[:nonceSize]
	ciphertext := rest[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: AES-GCM расшифровка: %w", err)
	}

	return plaintext, nil
}
