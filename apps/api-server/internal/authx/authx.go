package authx

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password with bcrypt at default cost.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", errors.New("password is empty")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword returns nil when the plaintext matches the stored hash.
func VerifyPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

// NewToken returns a cryptographically random hex token of `bytes*2` characters.
func NewToken(bytesN int) (string, error) {
	if bytesN <= 0 {
		bytesN = 32
	}
	buf := make([]byte, bytesN)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// HashToken returns a stable hex SHA256 of the given token. We never store raw
// session tokens; only their hashes go to the DB.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
