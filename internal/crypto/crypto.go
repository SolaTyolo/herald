package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

type Encryptor struct {
	gcm cipher.AEAD
}

func NewEncryptor(key string) (*Encryptor, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Encryptor{gcm: gcm}, nil
}

func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return e.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return e.gcm.Open(nil, nonce, data, nil)
}

func (e *Encryptor) EncryptJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return e.Encrypt(raw)
}

func (e *Encryptor) DecryptJSON(ciphertext []byte, v any) error {
	raw, err := e.Decrypt(ciphertext)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func HashAPIKey(key string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CompareAPIKey(key, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(key)) == nil
}
