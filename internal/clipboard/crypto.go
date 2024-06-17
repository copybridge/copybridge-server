package clipboard

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	// "crypto/sha512"
	"encoding/base64"
	"io"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
	// "golang.org/x/crypto/pbkdf2"
)

// HashPassword hashes the given password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// Authenticate compares the given password with the stored password hash.
func (c *Clipboard) Authenticate(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(password)) == nil
}

// deriveKey generates a key from the given password and salt using scrypt.
func deriveKey(password, salt []byte) ([]byte, error) {
	return scrypt.Key(password, salt, 1<<15, 8, 1, 32)
	// return pbkdf2.Key(password, salt, 100000, 32, sha512.New), nil
}

// Encrypt encrypts the clipboard data using the given password with AES-GCM.
func (c *Clipboard) Encrypt(password string) error {
	if c.Salt == "" {
		salt := make([]byte, 16)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return err
		}
		c.Salt = base64.StdEncoding.EncodeToString(salt)
	}

	decodedSalt, err := base64.StdEncoding.DecodeString(c.Salt)
	if err != nil {
		return err
	}

	key, err := deriveKey([]byte(password), decodedSalt)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	c.Nonce = base64.StdEncoding.EncodeToString(nonce)

	ciphertext := aesgcm.Seal(nil, nonce, []byte(c.Data), nil)
	c.Data = base64.StdEncoding.EncodeToString(ciphertext)
	c.IsEncrypted = true

	return nil
}

// Decrypt decrypts the clipboard data using the given password with AES-GCM.
func (c *Clipboard) Decrypt(password string) error {
	decodedSalt, err := base64.StdEncoding.DecodeString(c.Salt)
	if err != nil {
		return err
	}

	key, err := deriveKey([]byte(password), decodedSalt)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	decodedNonce, err := base64.StdEncoding.DecodeString(c.Nonce)
	if err != nil {
		return err
	}

	decodedCiphertext, err := base64.StdEncoding.DecodeString(c.Data)
	if err != nil {
		return err
	}

	plaintext, err := aesgcm.Open(nil, decodedNonce, decodedCiphertext, nil)
	if err != nil {
		return err
	}

	c.Data = string(plaintext)
	c.IsEncrypted = false

	return nil
}
