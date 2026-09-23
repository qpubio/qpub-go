package jwt_test

import (
	"strings"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/internal/jwt"
)

func TestSignAndDecode(t *testing.T) {
	payload := jwt.Payload{
		Alias: "user-1",
		Exp:   time.Now().Unix() + 3600,
		Permission: map[string][]string{"room.*": {"subscribe"}},
	}
	tok, err := jwt.Sign(payload, "key-1", "secret-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tok, ".") {
		t.Fatal("expected jwt segments")
	}
	dec, err := jwt.Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Header.AKI != "key-1" || dec.Payload.Alias != "user-1" {
		t.Fatalf("%+v", dec)
	}
}
