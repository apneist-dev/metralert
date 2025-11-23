package agent

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncrypt(t *testing.T) {
	// Генерируем тестовые RSA ключи
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey

	// Кодируем публичный ключ в PEM формат
	publicKeyBytes := x509.MarshalPKCS1PublicKey(publicKey)

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Тестовые данные для шифрования
	testBody := []byte("test data for encryption")

	tests := []struct {
		name         string
		body         []byte
		publicKeyPEM []byte
		expectError  bool
	}{
		{
			name:         "Valid encryption",
			body:         testBody,
			publicKeyPEM: publicKeyPEM,
			expectError:  false,
		},
		{
			name:         "Invalid PEM - nil publicKeyUntyped",
			body:         testBody,
			publicKeyPEM: []byte("invalid pem data"),
			expectError:  true,
		},
		{
			name:         "Empty body",
			body:         []byte{},
			publicKeyPEM: publicKeyPEM,
			expectError:  true,
		},
		{
			name:         "Nil body",
			body:         nil,
			publicKeyPEM: publicKeyPEM,
			expectError:  true,
		},
		{
			name:         "Empty PEM",
			body:         testBody,
			publicKeyPEM: []byte{},
			expectError:  true,
		},
		{
			name:         "Nil PEM",
			body:         testBody,
			publicKeyPEM: nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Encrypt(tt.body, tt.publicKeyPEM)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}
