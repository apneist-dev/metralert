package agent

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

func RetrieveEncrypt(body []byte, publicKeyPath string) ([]byte, error) {
	publicKeyPEM, err := os.ReadFile(publicKeyPath)

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

	publicKey, err := x509.ParsePKCS1PublicKey(publicKeyStruct.Bytes)
	if err != nil {
		return nil, err
	}

	encryptedData, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, body)
	if err != nil {
		return nil, err
	}

	return encryptedData, nil
}
