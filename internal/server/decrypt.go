package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

func RetrieveDecrypt(body []byte, privateKeyPath string) ([]byte, error) {
	privateKeyPEM, err := os.ReadFile(privateKeyPath)

	if err != nil {
		return nil, err
	}

	return Decrypt(body, privateKeyPEM)
}

func Decrypt(body []byte, privateKeyPEM []byte) ([]byte, error) {

	if len(body) == 0 || len(privateKeyPEM) == 0 {
		return nil, errors.New("publicKeyPEM or body are empty")
	}

	privateKeyStruct, rest := pem.Decode(privateKeyPEM)
	if privateKeyStruct == nil && len(rest) != 0 {
		return nil, errors.New("unable to decode PEM PublicKey")
	}

	if privateKeyStruct.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("the key is not private")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyStruct.Bytes)
	if err != nil {
		return nil, err
	}

	encryptedData, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, body)
	if err != nil {
		return nil, err
	}

	return encryptedData, nil
}
