package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
	"github.com/getio0909/voice-asset-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRemoteStreamableHTTP(t *testing.T) {
	if os.Getenv("VOICE_ASSET_MCP_REMOTE_E2E") != "1" {
		t.Skip("set VOICE_ASSET_MCP_REMOTE_E2E=1 for the isolated remote deployment")
	}
	endpoint := requiredRemoteEnvironment(t, "VOICE_ASSET_MCP_REMOTE_URL")
	token := requiredRemoteEnvironment(t, "VOICE_ASSET_MCP_HTTP_TOKEN")
	serverToken := requiredRemoteEnvironment(t, "VOICE_ASSET_SERVER_REMOTE_TOKEN")
	baseTransport := newRemoteTransport(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	unauthorizedRequest, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, bytes.NewBufferString(`{}`),
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

	authorizedClient := &http.Client{Transport: bearerRoundTripper{
		base: baseTransport, token: token,
	}}
	client := mcp.NewClient(&mcp.Implementation{Name: "remote-http-test", Version: "test"}, nil)
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
	writes := map[string]bool{
		"start_transcription":         false,
		"start_llm_correction":        false,
		"update_asset_metadata":       false,
		"add_tags":                    false,
		"remove_tags":                 false,
		"create_annotation":           false,
		"approve_transcript_revision": false,
		"create_audio_clip":           false,
		"export_transcript":           false,
	}
	if len(tools.Tools) != 21 {
		t.Fatalf("remote tool count = %d, want 21", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if _, ok := writes[tool.Name]; ok {
			writes[tool.Name] = true
		}
	}
	for name, found := range writes {
		if !found {
			t.Fatalf("remote write tool %q is missing", name)
		}
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

	assets := callRemoteTool[mcpserver.AssetListOutput](t, ctx, session, "list_assets", map[string]any{"limit": 100})
	var selected backend.Asset
	var revision mcpserver.TranscriptRevisionInfo
	for _, candidate := range assets.Items {
		if candidate.Status != "ready" || candidate.DurationMS == nil || *candidate.DurationMS <= 0 {
			continue
		}
		lineage := callRemoteTool[mcpserver.ListTranscriptRevisionsOutput](
			t, ctx, session, "list_transcript_revisions", map[string]any{"asset_id": candidate.ID, "limit": 20},
		)
		if len(lineage.Items) == 0 {
			continue
		}
		selected = candidate
		revision = lineage.Items[0]
		break
	}
	if selected.ID == "" {
		t.Fatal("remote deployment has no ready audio asset with a transcript revision")
	}

	clipEnd := min(*selected.DurationMS, int64(1000))
	segments := callRemoteTool[mcpserver.GetTranscriptSegmentsOutput](
		t, ctx, session, "get_transcript_segments",
		map[string]any{"revision_id": revision.ID, "start_ms": int64(0), "end_ms": clipEnd},
	)
	if segments.AssetID != selected.ID || segments.RevisionID != revision.ID {
		t.Fatal("remote transcript segment citation does not match the selected asset and revision")
	}

	createdClip := callRemoteTool[mcpserver.CreateAudioClipOutput](
		t, ctx, session, "create_audio_clip",
		map[string]any{"asset_id": selected.ID, "start_ms": int64(0), "end_ms": clipEnd},
	).Clip
	if createdClip.ID == "" || createdClip.AssetID != selected.ID || createdClip.DurationMS != clipEnd ||
		createdClip.MIMEType != "audio/wav" || createdClip.FileSize <= 44 || len(createdClip.SHA256) != 64 {
		t.Fatal("remote audio clip metadata is incomplete or inconsistent")
	}
	createdExport := callRemoteTool[mcpserver.ExportTranscriptOutput](
		t, ctx, session, "export_transcript", map[string]any{"revision_id": revision.ID, "format": "vtt"},
	).Export
	if createdExport.ID == "" || createdExport.AssetID != selected.ID || createdExport.RevisionID != revision.ID ||
		createdExport.Format != "vtt" || createdExport.MIMEType != "text/vtt; charset=utf-8" ||
		createdExport.FileSize <= 0 || len(createdExport.SHA256) != 64 {
		t.Fatal("remote transcript export metadata is incomplete or inconsistent")
	}

	gatewayURL, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("parse remote endpoint: %v", err)
	}
	gatewayURL.Path, gatewayURL.RawPath, gatewayURL.RawQuery, gatewayURL.Fragment = "/", "", "", ""
	serverClient := &http.Client{Transport: bearerRoundTripper{base: baseTransport, token: serverToken}}
	clipBody := downloadRemoteArtifact(
		t, ctx, serverClient, gatewayURL, createdClip.DownloadURL,
		createdClip.FileSize, createdClip.SHA256, "audio/wav",
	)
	if len(clipBody) < 12 || string(clipBody[:4]) != "RIFF" || string(clipBody[8:12]) != "WAVE" {
		t.Fatal("remote audio clip is not a RIFF/WAVE file")
	}
	validateRemoteRange(t, ctx, serverClient, gatewayURL, createdClip.DownloadURL, createdClip.FileSize)
	exportBody := downloadRemoteArtifact(
		t, ctx, serverClient, gatewayURL, createdExport.DownloadURL,
		createdExport.FileSize, createdExport.SHA256, "text/vtt",
	)
	if !bytes.HasPrefix(exportBody, []byte("WEBVTT")) {
		t.Fatal("remote transcript export is not WebVTT")
	}
}

func callRemoteTool[T any](
	t *testing.T,
	ctx context.Context,
	session *mcp.ClientSession,
	name string,
	arguments map[string]any,
) T {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}
	if result.IsError || result.StructuredContent == nil {
		t.Fatalf("CallTool(%s) returned an error result", name)
	}
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("encode CallTool(%s) structured result: %v", name, err)
	}
	var output T
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("decode CallTool(%s) structured result: %v", name, err)
	}
	return output
}

func downloadRemoteArtifact(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	baseURL *url.URL,
	downloadURL string,
	expectedSize int64,
	expectedSHA,
	expectedContentType string,
) []byte {
	t.Helper()
	reference, err := url.Parse(downloadURL)
	if err != nil || reference.IsAbs() || reference.Host != "" || reference.RawQuery != "" || reference.Fragment != "" ||
		!strings.HasPrefix(reference.Path, "/api/v1/") {
		t.Fatal("artifact download URL must be an authenticated relative API path without query parameters")
	}
	endpoint := baseURL.ResolveReference(reference).String()
	headRequest, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint, nil)
	if err != nil {
		t.Fatalf("create artifact HEAD request: %v", err)
	}
	headResponse, err := client.Do(headRequest)
	if err != nil {
		t.Fatalf("send artifact HEAD request: %v", err)
	}
	_ = headResponse.Body.Close()
	if headResponse.StatusCode != http.StatusOK || headResponse.ContentLength != expectedSize ||
		!strings.HasPrefix(headResponse.Header.Get("Content-Type"), expectedContentType) ||
		headResponse.Header.Get("Accept-Ranges") != "bytes" {
		t.Fatalf("artifact HEAD metadata is inconsistent: status=%d", headResponse.StatusCode)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create artifact GET request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send artifact GET request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), expectedContentType) {
		t.Fatalf("artifact GET metadata is inconsistent: status=%d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<20+1))
	if err != nil {
		t.Fatalf("read artifact body: %v", err)
	}
	if int64(len(body)) != expectedSize {
		t.Fatalf("artifact size = %d, want %d", len(body), expectedSize)
	}
	digest := sha256.Sum256(body)
	if fmt.Sprintf("%x", digest) != expectedSHA {
		t.Fatal("artifact SHA-256 does not match its metadata")
	}
	return body
}

func validateRemoteRange(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	baseURL *url.URL,
	downloadURL string,
	totalSize int64,
) {
	t.Helper()
	reference, err := url.Parse(downloadURL)
	if err != nil {
		t.Fatalf("parse artifact range URL: %v", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.ResolveReference(reference).String(), nil)
	if err != nil {
		t.Fatalf("create artifact range request: %v", err)
	}
	request.Header.Set("Range", "bytes=0-15")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send artifact range request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read artifact range body: %v", err)
	}
	wantContentRange := fmt.Sprintf("bytes 0-15/%d", totalSize)
	if response.StatusCode != http.StatusPartialContent || len(body) != 16 ||
		response.Header.Get("Content-Range") != wantContentRange || string(body[:4]) != "RIFF" {
		t.Fatalf("artifact range response is inconsistent: status=%d", response.StatusCode)
	}
}

type bearerRoundTripper struct {
	base  http.RoundTripper
	token string
}

func (roundTripper bearerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if roundTripper.base == nil {
		return nil, fmt.Errorf("base transport is required")
	}
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+roundTripper.token)
	return roundTripper.base.RoundTrip(clone)
}

func requiredRemoteEnvironment(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}

func newRemoteTransport(t *testing.T) *http.Transport {
	t.Helper()
	var roots *x509.CertPool
	caFile := strings.TrimSpace(os.Getenv("VOICE_ASSET_MCP_CA_FILE"))
	if caFile != "" {
		var err error
		roots, err = x509.SystemCertPool()
		if err != nil {
			t.Fatalf("load system CA pool: %v", err)
		}
		if roots == nil {
			roots = x509.NewCertPool()
		}
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			t.Fatalf("read remote CA: %v", err)
		}
		if !roots.AppendCertsFromPEM(caPEM) {
			t.Fatal("remote CA file contains no certificate")
		}
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    roots,
	}}
	t.Cleanup(transport.CloseIdleConnections)
	return transport
}
