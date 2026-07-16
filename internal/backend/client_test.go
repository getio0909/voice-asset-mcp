package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSystemCapabilities(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/capabilities" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"server_version":"0.1.0","api_version":"v1","contract_version":"0.1.0","features":["mock_asr"]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.GetSystemCapabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ContractVersion != "0.1.0" || len(got.Features) != 1 {
		t.Fatalf("unexpected capabilities: %#v", got)
	}
}

func TestGetSystemCapabilitiesRedactsResponseBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "provider-secret", http.StatusBadGateway)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetSystemCapabilities(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "request capabilities: server returned HTTP 502" {
		t.Fatalf("error = %q", got)
	}
}

func TestNewClientRejectsInsecureRemoteURL(t *testing.T) {
	t.Parallel()
	if _, err := NewClient("http://server.example", "token", nil); err == nil {
		t.Fatal("NewClient() accepted insecure remote URL")
	}
}

func TestGetSystemCapabilitiesRejectsIncompatibleResponses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{name: "missing fields", body: `{}`},
		{name: "unsupported API", body: `{"server_version":"0.1.0","api_version":"v2","contract_version":"0.1.0","features":[]}`},
		{name: "unsupported contract", body: `{"server_version":"0.1.0","api_version":"v1","contract_version":"0.2.0","features":[]}`},
		{name: "unsorted features", body: `{"server_version":"0.1.0","api_version":"v1","contract_version":"0.1.0","features":["zeta","alpha"]}`},
		{name: "trailing JSON", body: `{"server_version":"0.1.0","api_version":"v1","contract_version":"0.1.0","features":[]} {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.GetSystemCapabilities(context.Background()); err == nil {
				t.Fatal("GetSystemCapabilities() accepted incompatible response")
			}
		})
	}
}
