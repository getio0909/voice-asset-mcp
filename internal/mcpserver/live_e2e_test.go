package mcpserver

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
)

func TestLiveMCPReadWorkflow(t *testing.T) {
	if os.Getenv("VOICE_ASSET_MCP_LIVE_E2E") != "1" {
		t.Skip("set VOICE_ASSET_MCP_LIVE_E2E=1 for the isolated live workflow")
	}
	baseURL := requiredLiveEnvironment(t, "VOICE_ASSET_SERVER_URL")
	token := requiredLiveEnvironment(t, "VOICE_ASSET_SERVER_TOKEN")
	client, err := backend.NewClient(baseURL, token, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() { serverDone <- New(client, "live-test").Run(ctx, serverTransport) }()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "live-test-client", Version: "test"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		select {
		case <-serverDone:
		case <-time.After(time.Second):
			t.Error("MCP server did not stop")
		}
	})

	capabilityResult := callTool(t, ctx, session, "get_system_capabilities", nil)
	capabilities := decodeStructured[GetSystemCapabilitiesOutput](t, capabilityResult)
	if capabilities.ContractVersion != backend.SupportedContractVersion {
		t.Fatalf("contract version = %q", capabilities.ContractVersion)
	}

	listResult := callTool(t, ctx, session, "list_assets", map[string]any{"limit": 20})
	assets := decodeStructured[AssetListOutput](t, listResult)
	if len(assets.Items) == 0 {
		t.Fatal("live workspace has no assets")
	}
	selected := assets.Items[0]

	collectionsResult := callTool(t, ctx, session, "list_collections", map[string]any{"limit": 20})
	_ = decodeStructured[CollectionListOutput](t, collectionsResult)
	tagsResult := callTool(t, ctx, session, "list_tags", map[string]any{"limit": 20})
	_ = decodeStructured[TagListOutput](t, tagsResult)
	annotationsResult := callTool(t, ctx, session, "get_annotations", map[string]any{
		"asset_id": selected.ID, "limit": 20,
	})
	annotations := decodeStructured[AnnotationListOutput](t, annotationsResult)
	for _, annotation := range annotations.Items {
		if annotation.AssetID != selected.ID || annotation.StartMS < 0 ||
			(annotation.EndMS != nil && *annotation.EndMS <= annotation.StartMS) {
			t.Fatalf("invalid annotation citation = %+v", annotation)
		}
	}
	processingResult := callTool(t, ctx, session, "get_processing_status", map[string]any{"asset_id": selected.ID})
	processing := decodeStructured[GetProcessingStatusOutput](t, processingResult)
	if processing.Status.AssetID != selected.ID || len(processing.Status.Jobs) > 20 {
		t.Fatalf("processing status = %+v", processing.Status)
	}

	searchResult := callTool(t, ctx, session, "search_assets", map[string]any{
		"query": selected.Title, "limit": 20,
	})
	search := decodeStructured[AssetListOutput](t, searchResult)
	found := false
	for _, candidate := range search.Items {
		if candidate.ID == selected.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("search did not return selected asset %s", selected.ID)
	}

	lineageResult := callTool(t, ctx, session, "list_transcript_revisions", map[string]any{
		"asset_id": selected.ID, "limit": 20,
	})
	lineage := decodeStructured[ListTranscriptRevisionsOutput](t, lineageResult)
	if len(lineage.Items) == 0 {
		t.Fatalf("asset %s has no transcript revision lineage", selected.ID)
	}
	revisionID := lineage.Items[0].ID

	transcriptResult := callTool(t, ctx, session, "get_transcript", map[string]any{"revision_id": revisionID})
	transcript := decodeStructured[GetTranscriptOutput](t, transcriptResult)
	if transcript.Revision.ID != revisionID || len(transcript.Revision.Segments) == 0 {
		t.Fatalf("revision %s has no segment timeline", revisionID)
	}
	segment := transcript.Revision.Segments[0]
	if segment.EndMS <= segment.StartMS {
		t.Fatalf("segment has invalid range %d-%d", segment.StartMS, segment.EndMS)
	}
	transcriptSearchResult := callTool(t, ctx, session, "search_assets", map[string]any{
		"query": segment.Text, "limit": 20,
	})
	transcriptSearch := decodeStructured[AssetListOutput](t, transcriptSearchResult)
	matchedSegment := false
	for _, candidate := range transcriptSearch.Items {
		if candidate.ID != selected.ID || candidate.Search == nil {
			continue
		}
		for _, hit := range candidate.Search.Segments {
			if hit.SegmentID == segment.ID && hit.RevisionID == revisionID &&
				hit.StartMS == segment.StartMS && hit.EndMS == segment.EndMS {
				matchedSegment = true
				break
			}
		}
	}
	if !matchedSegment {
		t.Fatalf("transcript search did not return segment %s with its timecode", segment.ID)
	}
	startMS := segment.StartMS
	endMS := segment.EndMS
	if endMS-startMS > 2 {
		startMS++
		endMS--
	}
	segmentsResult := callTool(t, ctx, session, "get_transcript_segments", map[string]any{
		"revision_id": revisionID, "start_ms": startMS, "end_ms": endMS,
	})
	segments := decodeStructured[GetTranscriptSegmentsOutput](t, segmentsResult)
	if segments.AssetID != selected.ID || segments.RevisionID != revisionID ||
		segments.StartMS != startMS || segments.EndMS != endMS || len(segments.Segments) == 0 {
		t.Fatalf("exact range output = %+v", segments)
	}
	firstCitation := segments.Segments[0]
	if firstCitation.SegmentID == "" || firstCitation.OverlapStart != startMS || firstCitation.OverlapEnd != endMS {
		t.Fatalf("exact citation = %+v", firstCitation)
	}
}

func requiredLiveEnvironment(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}
