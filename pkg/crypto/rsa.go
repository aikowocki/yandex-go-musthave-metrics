// Package crypto  для асимметричного шифрования сообщений между агентом и сервером.
package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// LoadPublicKey загружает RSA публичный ключ из PEM-файла по указанному пути.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("crypto: чтение файла публичного ключа: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("crypto: не удалось декодировать PEM-блок публичного ключа")
	}

	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto: парсинг публичного ключа: %w", err)
	}

	return key, nil
}

// LoadPrivateKey загружает RSA приватный ключ из PEM-файла по указанному пути.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("crypto: чтение файла приватного ключа: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("crypto: не удалось декодировать PEM-блок приватного ключа")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto: парсинг приватного ключа: %w", err)
	}

	return key, nil
}
