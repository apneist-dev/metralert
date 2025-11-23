package agent

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

func RetrieveEncrypt(body []byte, publicKeyPath string) ([]byte, error) {
	var publicKeyPEM []byte
	f, err := os.OpenFile(publicKeyPath, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	return Encrypt(body, publicKeyPEM)
}

func Encrypt(body []byte, publicKeyPEM []byte) ([]byte, error) {

	if len(body) == 0 || len(publicKeyPEM) == 0 {
		return nil, errors.New("publicKeyPEM or body are empty")
	}

	publicKeyStruct, rest := pem.Decode(publicKeyPEM)
	if publicKeyStruct == nil && len(rest) != 0 {
		return nil, errors.New("unable to decode PEM PublicKey")
	}

	if publicKeyStruct.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("the key is not public")
	}

	// PKCS#1 format
	publicKey, err := x509.ParsePKCS1PublicKey(publicKeyStruct.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS1 public key: %w", err)
	}

	// Encrypt the data
	encryptedData, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, body)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	return encryptedData, nil
}
