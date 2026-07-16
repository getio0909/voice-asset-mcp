package mcpserver

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
)

type stubClient struct{}

func (stubClient) GetSystemCapabilities(context.Context) (backend.Capabilities, error) {
	return backend.Capabilities{
		ServerVersion:   "0.1.0",
		APIVersion:      "v1",
		ContractVersion: "0.1.0",
		Features:        []string{"mock_asr"},
	}, nil
}

func TestGetSystemCapabilitiesTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- New(stubClient{}, "test").Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		select {
		case <-serverDone:
		case <-time.After(time.Second):
			t.Error("server did not stop")
		}
	})

	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_system_capabilities"})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() returned tool error: %#v", result.Content)
	}
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var output GetSystemCapabilitiesOutput
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	if output.ContractVersion != "0.1.0" || len(output.Features) != 1 {
		t.Fatalf("unexpected output: %#v", output)
	}
}
