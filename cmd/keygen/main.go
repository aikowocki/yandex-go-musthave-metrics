// Keygen генерирует пару RSA-ключей и сохраняет их в PEM-файлы.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	var (
		bits       int
		publicOut  string
		privateOut string
	)

	flag.IntVar(&bits, "bits", 4096, "размер RSA-ключа в битах")
	flag.StringVar(&publicOut, "public", "keys/public.pem", "путь для сохранения публичного ключа")
	flag.StringVar(&privateOut, "private", "keys/private.pem", "путь для сохранения приватного ключа")
	flag.Parse()

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		log.Fatalf("не удалось сгенерировать RSA-ключ: %v", err)
	}

	// Создаём директории для выходных файлов.
	if err := os.MkdirAll(filepath.Dir(publicOut), 0o755); err != nil {
		log.Fatalf("не удалось создать директорию для публичного ключа: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(privateOut), 0o755); err != nil {
		log.Fatalf("не удалось создать директорию для приватного ключа: %v", err)
	}

	// Сохраняем приватный ключ.
	privateKeyFile, err := os.Create(privateOut)
	if err != nil {
		log.Fatalf("не удалось создать файл приватного ключа: %v", err)
	}
	defer privateKeyFile.Close()

	err = pem.Encode(privateKeyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		log.Fatalf("не удалось записать приватный ключ: %v", err)
	}

	// Сохраняем публичный ключ.
	pubFile, err := os.Create(publicOut)
	if err != nil {
		log.Fatalf("не удалось создать файл публичного ключа: %v", err)
	}
	defer pubFile.Close()

	err = pem.Encode(pubFile, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	})
	if err != nil {
		log.Fatalf("не удалось записать публичный ключ: %v", err)
	}

	fmt.Printf("Ключи сгенерированы:\n  Публичный:  %s\n  Приватный: %s\n", publicOut, privateOut)
}
