package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
	"github.com/getio0909/voice-asset-mcp/internal/config"
	"github.com/getio0909/voice-asset-mcp/internal/mcpserver"
)

type capabilityStub struct{}

func (capabilityStub) GetSystemCapabilities(context.Context) (backend.Capabilities, error) {
	return backend.Capabilities{
		ServerVersion:   "0.1.0-dev",
		APIVersion:      backend.SupportedAPIVersion,
		ContractVersion: backend.SupportedContractVersion,
		Features:        []string{"capability_negotiation"},
	}, nil
}

func TestBearerAuth(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := bearerAuth(next, "expected")

	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer expected")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("authenticated status = %d", response.Code)
	}
}

func TestHTTPHandlerRejectsCrossOriginPost(t *testing.T) {
	t.Parallel()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	handler := newHTTPHandler(server, config.Config{HTTPBearerToken: "expected"})
	request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer expected")
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestHTTPHandlerRejectsOversizedBody(t *testing.T) {
	t.Parallel()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	handler := newHTTPHandler(server, config.Config{})
	request := httptest.NewRequest(
		http.MethodPost,
		"http://localhost/mcp",
		bytes.NewReader(make([]byte, maxMCPRequestBytes+1)),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestStreamableHTTPIntegration(t *testing.T) {
	t.Parallel()
	server := mcpserver.New(capabilityStub{}, "test")
	httpServer := httptest.NewServer(newHTTPHandler(server, config.Config{}))
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "http-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL + "/mcp",
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer session.Close()
	assertCapabilitiesTool(t, ctx, session)
}

func TestStdioIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess integration test in short mode")
	}
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"server_version":"0.1.0-dev","api_version":"v1","contract_version":"0.1.0","features":["capability_negotiation"]}`))
	}))
	defer backendServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{
		Command: exec.Command("go", "run", ".", "--transport=stdio", "--server-url="+backendServer.URL),
	}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer session.Close()
	assertCapabilitiesTool(t, ctx, session)
}

func assertCapabilitiesTool(t *testing.T, ctx context.Context, session *mcp.ClientSession) {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_system_capabilities"})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError || result.StructuredContent == nil {
		t.Fatalf("unexpected tool result: %#v", result)
	}
}
