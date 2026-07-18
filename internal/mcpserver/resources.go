package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const resourceMIMEType = "application/json"

func addResourceTemplates(server *mcp.Server, client voiceAssetReader) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name: "asset", Title: "VoiceAsset asset", MIMEType: resourceMIMEType,
		Description: "Workspace-scoped asset metadata from the public Server API.",
		URITemplate: "voice-asset://assets/{asset_id}",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		assetID, err := parseResourceID(request.Params.URI, "assets")
		if err != nil {
			return nil, err
		}
		asset, err := client.GetAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		return jsonResource(request.Params.URI, asset)
	})

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name: "latest_asset_transcript", Title: "Latest asset transcript", MIMEType: resourceMIMEType,
		Description: "Latest immutable transcript revision for a workspace-scoped asset.",
		URITemplate: "voice-asset://assets/{asset_id}/transcript/latest",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		assetID, err := parseResourceID(request.Params.URI, "assets", "transcript", "latest")
		if err != nil {
			return nil, err
		}
		transcripts, err := client.ListTranscripts(ctx, assetID)
		if err != nil {
			return nil, err
		}
		if len(transcripts.Items) == 0 {
			return nil, fmt.Errorf("asset has no transcript revision")
		}
		latest := transcripts.Items[0]
		for _, candidate := range transcripts.Items[1:] {
			if candidate.RevisionCreatedAt.After(latest.RevisionCreatedAt) {
				latest = candidate
			}
		}
		revision, err := client.GetTranscriptRevision(ctx, latest.LatestRevisionID)
		if err != nil {
			return nil, err
		}
		return jsonResource(request.Params.URI, revision)
	})

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name: "transcript_revision", Title: "Transcript revision", MIMEType: resourceMIMEType,
		Description: "Specified immutable transcript revision with exact segment timeline.",
		URITemplate: "voice-asset://transcripts/{revision_id}",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		revisionID, err := parseResourceID(request.Params.URI, "transcripts")
		if err != nil {
			return nil, err
		}
		revision, err := client.GetTranscriptRevision(ctx, revisionID)
		if err != nil {
			return nil, err
		}
		return jsonResource(request.Params.URI, revision)
	})

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name: "collection", Title: "VoiceAsset collection", MIMEType: resourceMIMEType,
		Description: "Workspace-scoped collection and current non-trashed asset count.",
		URITemplate: "voice-asset://collections/{collection_id}",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		collectionID, err := parseResourceID(request.Params.URI, "collections")
		if err != nil {
			return nil, err
		}
		collection, err := client.GetCollection(ctx, collectionID)
		if err != nil {
			return nil, err
		}
		return jsonResource(request.Params.URI, collection)
	})

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name: "transcription_job", Title: "Transcription job", MIMEType: resourceMIMEType,
		Description: "Workspace-scoped durable job state from the public Server API.",
		URITemplate: "voice-asset://jobs/{job_id}",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		jobID, err := parseResourceID(request.Params.URI, "jobs")
		if err != nil {
			return nil, err
		}
		job, err := client.GetTranscriptionJob(ctx, jobID)
		if err != nil {
			return nil, err
		}
		return jsonResource(request.Params.URI, job)
	})
}

func jsonResource(uri string, value any) (*mcp.ReadResourceResult, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode resource: %w", err)
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
		URI: uri, MIMEType: resourceMIMEType, Text: string(encoded),
	}}}, nil
}

func parseResourceID(rawURI, host string, suffix ...string) (string, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil || parsed.Scheme != "voice-asset" || parsed.Host != host || parsed.User != nil ||
		parsed.Opaque != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid VoiceAsset resource URI")
	}
	segments := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	if len(segments) != len(suffix)+1 {
		return "", fmt.Errorf("invalid VoiceAsset resource URI")
	}
	if err := validateUUID("resource identifier", segments[0]); err != nil {
		return "", err
	}
	for index, expected := range suffix {
		if segments[index+1] != expected {
			return "", fmt.Errorf("invalid VoiceAsset resource URI")
		}
	}
	return strings.ToLower(segments[0]), nil
}
