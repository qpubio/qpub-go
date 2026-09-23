package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// HMACSign returns standard base64 HMAC-SHA256 (matches qpub-js Crypto.hmacSign).
func HMACSign(data, key string) (string, error) {
	mac := hmac.New(sha256.New, []byte(key))
	_, err := mac.Write([]byte(data))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}
