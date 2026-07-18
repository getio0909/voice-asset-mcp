package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRemoteStreamableHTTPReadOnly(t *testing.T) {
	if os.Getenv("VOICE_ASSET_MCP_REMOTE_READ_E2E") != "1" {
		t.Skip("set VOICE_ASSET_MCP_REMOTE_READ_E2E=1 for the read-only deployment smoke test")
	}
	endpoint := requiredRemoteEnvironment(t, "VOICE_ASSET_MCP_REMOTE_URL")
	token := requiredRemoteEnvironment(t, "VOICE_ASSET_MCP_HTTP_TOKEN")
	baseTransport := newRemoteTransport(t)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	unauthorizedRequest, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, strings.NewReader(`{}`),
	)
	if err != nil {
		t.Fatalf("create unauthorized request: %v", err)
	}
	unauthorizedRequest.Header.Set("Content-Type", "application/json")
	unauthorizedResponse, err := (&http.Client{Transport: baseTransport}).Do(unauthorizedRequest)
	if err != nil {
		t.Fatalf("send unauthorized request: %v", err)
	}
	_ = unauthorizedResponse.Body.Close()
	if unauthorizedResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", unauthorizedResponse.StatusCode)
	}
	if got := unauthorizedResponse.Header.Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q, want Bearer", got)
	}

	authorizedClient := &http.Client{Transport: bearerRoundTripper{
		base: baseTransport, token: token,
	}}
	client := mcp.NewClient(&mcp.Implementation{Name: "remote-read-smoke", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: endpoint, HTTPClient: authorizedClient, DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("connect to remote MCP: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list remote tools: %v", err)
	}
	if len(tools.Tools) != 21 {
		t.Fatalf("remote tool count = %d, want 21", len(tools.Tools))
	}
	for _, tool := range []string{"get_system_capabilities", "list_assets"} {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tool})
		if err != nil {
			t.Fatalf("CallTool(%s) error = %v", tool, err)
		}
		if result.IsError || result.StructuredContent == nil {
			t.Fatalf("CallTool(%s) returned an error result", tool)
		}
	}
}
