package auth_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/qpubio/qpub-go/auth"
	"github.com/qpubio/qpub-go/option"
)

func TestCanonicalGoldenVectors(t *testing.T) {
	path := filepath.Join("testdata", "canonical_vectors.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name       string                 `json:"name"`
		AKI        string                 `json:"aki"`
		Timestamp  int64                  `json:"timestamp"`
		Alias      string                 `json:"alias"`
		Permission option.Permission      `json:"permission"`
		Canonical  string                 `json:"canonical"`
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		got := auth.BuildCanonicalString(tc.AKI, tc.Timestamp, tc.Alias, tc.Permission)
		if got != tc.Canonical {
			t.Fatalf("%s: got %q want %q", tc.Name, got, tc.Canonical)
		}
	}
}
