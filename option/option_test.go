package option_test

import (
	"testing"

	"github.com/qpubio/qpub-go/option"
)

func TestNewManagerDefaults(t *testing.T) {
	m := option.NewManager()
	got := m.Get()
	want := option.DefaultOption()
	if got != want {
		t.Fatalf("defaults mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestNewManagerMergeOptionFuncs(t *testing.T) {
	m := option.NewManager(
		option.WithAPIKey("test-key"),
		option.WithAutoConnect(false),
	)
	got := m.Get()
	if got.APIKey != "test-key" {
		t.Fatalf("apiKey=%q", got.APIKey)
	}
	if got.AutoConnect {
		t.Fatal("expected autoConnect false")
	}
	if got.WSHost != option.DefaultOption().WSHost {
		t.Fatalf("wsHost=%q", got.WSHost)
	}
}

func TestManagerSetMergePreservesFields(t *testing.T) {
	m := option.NewManager(option.WithAPIKey("test-key"))
	m.Set(option.Option{HTTPHost: "custom-host"})
	got := m.Get()
	if got.APIKey != "test-key" {
		t.Fatalf("apiKey=%q", got.APIKey)
	}
	if got.HTTPHost != "custom-host" {
		t.Fatalf("httpHost=%q", got.HTTPHost)
	}
}

func TestManagerSetDoesNotClearBoolsViaZeroValue(t *testing.T) {
	m := option.NewManager(option.WithAutoConnect(false))
	m.Set(option.Option{APIKey: "new-key"})
	got := m.Get()
	if got.APIKey != "new-key" {
		t.Fatalf("apiKey=%q", got.APIKey)
	}
	if got.AutoConnect {
		t.Fatal("Set must not flip autoConnect via zero-value partial Option; use OptionFunc")
	}
}

func TestManagerReset(t *testing.T) {
	m := option.NewManager(
		option.WithAPIKey("custom-key"),
		option.WithAutoConnect(false),
		func(o *option.Option) { o.WSHost = "custom-host" },
	)
	m.Reset()
	got := m.Get()
	want := option.DefaultOption()
	if got != want {
		t.Fatalf("after reset:\n got  %+v\n want %+v", got, want)
	}
}

func TestBuildRestBaseURL(t *testing.T) {
	t.Parallel()
	def := option.DefaultOption()
	if option.BuildRestBaseURL(def) != "https://rest.qpub.io/v1" {
		t.Fatal("default secure URL")
	}
	insecure := def
	insecure.IsSecure = false
	if option.BuildRestBaseURL(insecure) != "http://rest.qpub.io/v1" {
		t.Fatal("insecure URL")
	}
	port := 8080
	custom := option.DefaultOption()
	custom.HTTPHost = "localhost"
	custom.HTTPPort = &port
	custom.IsSecure = false
	if option.BuildRestBaseURL(custom) != "http://localhost:8080/v1" {
		t.Fatal("custom port URL")
	}
}

func TestGetReturnsCopy(t *testing.T) {
	m := option.NewManager(option.WithAPIKey("a"))
	a := m.Get()
	a.APIKey = "mutated"
	if m.Get().APIKey != "a" {
		t.Fatal("Get should return a copy")
	}
}
