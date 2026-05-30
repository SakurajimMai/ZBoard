package subtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
)

const (
	StoredPrefix             = "v1."
	DevelopmentSigningSecret = "dev-token-secret"
)

var ErrWeakSigningSecret = errors.New("signing secret must be set and must not use the development default")

func ValidateSigningSecret(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" || secret == DevelopmentSigningSecret {
		return ErrWeakSigningSecret
	}
	return nil
}

func StoredToken(salt string) string {
	return StoredPrefix + salt
}

func PublicToken(userID int64, salt, secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if err := ValidateSigningSecret(secret); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strconv.FormatInt(userID, 10)))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write([]byte(salt))
	return StoredPrefix + salt + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

func Materialize(userID int64, storedToken, secret string) (string, error) {
	if strings.HasPrefix(storedToken, StoredPrefix) {
		salt := strings.TrimPrefix(storedToken, StoredPrefix)
		if salt != "" && !strings.Contains(salt, ".") {
			return PublicToken(userID, salt, secret)
		}
	}
	return storedToken, nil
}

func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
