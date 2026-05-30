package authx

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// MaxPasswordBytes caps password input to avoid bcrypt CPU exhaustion. bcrypt
// itself silently truncates to 72 bytes, but we reject earlier so callers see
// an explicit error and an attacker can't pin a CPU spinning on a 1 MB body.
const MaxPasswordBytes = 128

// HashPassword hashes a plaintext password with bcrypt at default cost.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", errors.New("password is empty")
	}
	if len(plain) > MaxPasswordBytes {
		return "", errors.New("password too long")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword returns nil when the plaintext matches the stored hash.
// Inputs longer than MaxPasswordBytes are rejected without invoking bcrypt.
func VerifyPassword(hash, plain string) error {
	if len(plain) > MaxPasswordBytes {
		return bcrypt.ErrMismatchedHashAndPassword
	}
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
