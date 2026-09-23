package jwt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	qcrypto "github.com/qpubio/qpub-go/internal/crypto"
)

var ErrInvalidJWT = errors.New("Invalid JWT format")

// Header matches QPub JWT header.
type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	AKI string `json:"aki"`
}

// Payload matches QPub JWT payload.
type Payload struct {
	Alias      string                 `json:"alias,omitempty"`
	Permission map[string][]string    `json:"permission,omitempty"`
	Exp        int64                  `json:"exp"`
}

// Decoded token parts.
type Decoded struct {
	Header  Header
	Payload Payload
}

func base64URLDecode(s string) ([]byte, error) {
	// Accept both URL and standard base64 (tests use standard).
	if rem := len(s) % 4; rem > 0 {
		s += strings.Repeat("=", 4-rem)
	}
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(s)
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// Decode parses and validates required fields (aki, exp).
func Decode(token string) (Decoded, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return Decoded{}, ErrInvalidJWT
	}
	hb, err := base64URLDecode(parts[0])
	if err != nil {
		return Decoded{}, ErrInvalidJWT
	}
	pb, err := base64URLDecode(parts[1])
	if err != nil {
		return Decoded{}, ErrInvalidJWT
	}
	var h Header
	var p Payload
	if err := json.Unmarshal(hb, &h); err != nil {
		return Decoded{}, ErrInvalidJWT
	}
	if err := json.Unmarshal(pb, &p); err != nil {
		return Decoded{}, ErrInvalidJWT
	}
	if h.AKI == "" || p.Exp == 0 {
		return Decoded{}, errors.New("Missing required JWT fields")
	}
	return Decoded{Header: h, Payload: p}, nil
}

// IsExpired reports whether the token is expired.
func IsExpired(token string) bool {
	d, err := Decode(token)
	if err != nil {
		return true
	}
	return d.Payload.Exp*1000 <= time.Now().UnixMilli()
}

// Sign creates HS256 JWT (qpub-js JWT.sign).
func Sign(payload Payload, apiKeyPublicID, apiKeySecret string) (string, error) {
	header := Header{Alg: "HS256", Typ: "JWT", AKI: apiKeyPublicID}
	hb, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encHeader := base64URLEncode(hb)
	encPayload := base64URLEncode(pb)
	dataToSign := encHeader + "." + encPayload
	sigB64, err := qcrypto.HMACSign(dataToSign, apiKeySecret)
	if err != nil {
		return "", fmt.Errorf("Failed to sign JWT: %w", err)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return "", fmt.Errorf("Failed to sign JWT: %w", err)
	}
	encSig := base64URLEncode(sigBytes)
	return dataToSign + "." + encSig, nil
}
