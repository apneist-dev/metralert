package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"path/filepath"
)

func main() {

	// создаём новый приватный RSA-ключ длиной 4096 бит
	// для генерации ключа и сертификата
	// используется rand.Reader в качестве источника случайных данных
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatal(err)
	}
	publicKey := privateKey.PublicKey

	// кодируем ключи в формате PEM, который
	// используется для хранения и обмена криптографическими ключами

	var privateKeyPEM bytes.Buffer
	err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		log.Fatal(err)
	}

	var publicKeyPEM bytes.Buffer
	err = pem.Encode(&publicKeyPEM, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&publicKey),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Сохраняем сертификат и приватный ключ в файлы private.pem и public.pem
	wDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if err = os.WriteFile(filepath.Join(wDir, "private.pem"), privateKeyPEM.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}

	if err = os.WriteFile(filepath.Join(wDir, "public.pem"), publicKeyPEM.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
}
