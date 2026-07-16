package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
)

type capabilityReader interface {
	GetSystemCapabilities(context.Context) (backend.Capabilities, error)
}

type getSystemCapabilitiesInput struct{}

type GetSystemCapabilitiesOutput struct {
	ServerVersion   string   `json:"server_version" jsonschema:"VoiceAsset Server version"`
	APIVersion      string   `json:"api_version" jsonschema:"REST API version"`
	ContractVersion string   `json:"contract_version" jsonschema:"OpenAPI contract version"`
	Features        []string `json:"features" jsonschema:"advertised server capabilities"`
}

func New(client capabilityReader, version string) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "voice-asset-mcp", Version: version},
		nil,
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_system_capabilities",
			Description: "Read the connected VoiceAsset Server capability contract.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ getSystemCapabilitiesInput) (*mcp.CallToolResult, GetSystemCapabilitiesOutput, error) {
			capabilities, err := client.GetSystemCapabilities(ctx)
			if err != nil {
				return nil, GetSystemCapabilitiesOutput{}, err
			}
			return nil, GetSystemCapabilitiesOutput{
				ServerVersion:   capabilities.ServerVersion,
				APIVersion:      capabilities.APIVersion,
				ContractVersion: capabilities.ContractVersion,
				Features:        capabilities.Features,
			}, nil
		},
	)
	return server
}
