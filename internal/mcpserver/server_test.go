package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
)

type stubClient struct {
	listInput       backend.ListAssetsInput
	collectionInput backend.ListPageInput
	tagInput        backend.ListPageInput
	annotationInput backend.ListPageInput
	annotationAsset string
	processingAsset string
	writeCalls      map[string]int
	writeAssetID    string
	writeRevisionID string
	writeKey        string
	metadataInput   backend.UpdateAssetMetadataInput
	writeTagIDs     []string
	annotationWrite backend.CreateAnnotationInput
	clipStartMS     int64
	clipEndMS       int64
	exportFormat    string
	acceptPending   bool
}

func (*stubClient) GetSystemCapabilities(context.Context) (backend.Capabilities, error) {
	return backend.Capabilities{
		ServerVersion:   "0.2.0",
		APIVersion:      "v1",
		ContractVersion: "0.22.0",
		Features:        []string{"mock_asr"},
	}, nil
}

func (client *stubClient) ListAssets(_ context.Context, input backend.ListAssetsInput) (backend.AssetList, error) {
	client.listInput = input
	result := backend.Asset{
		ID: "30000000-0000-4000-8000-000000000001", WorkspaceID: "10000000-0000-4000-8000-000000000001",
		Title: "Quarterly recording", Language: "en-US", Status: "ready", Version: 2,
		CreatedAt: time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
	}
	if input.Query != "" {
		result.Search = &backend.AssetSearchMatch{
			ProviderIDs: []string{"mock_asr"},
			Segments: []backend.AssetSearchSegmentHit{{
				TranscriptID: "40000000-0000-4000-8000-000000000001",
				RevisionID:   "50000000-0000-4000-8000-000000000001",
				SegmentID:    "60000000-0000-4000-8000-000000000001",
				StartMS:      1200, EndMS: 2400, Text: "Quarterly result",
			}},
		}
	}
	return backend.AssetList{Items: []backend.Asset{result}}, nil
}

func (*stubClient) GetAsset(context.Context, string) (backend.Asset, error) {
	return backend.Asset{
		ID: "30000000-0000-4000-8000-000000000001", WorkspaceID: "10000000-0000-4000-8000-000000000001",
		Title: "Quarterly recording", Language: "en-US", Status: "ready", Version: 2,
		CreatedAt: time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (*stubClient) GetCollection(context.Context, string) (backend.Collection, error) {
	return backend.Collection{
		ID: "70000000-0000-4000-8000-000000000001", WorkspaceID: "10000000-0000-4000-8000-000000000001",
		Name: "Interviews", AssetCount: 2, Version: 1,
		CreatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC),
	}, nil
}

func (client *stubClient) ListCollections(_ context.Context, input backend.ListPageInput) (backend.CollectionList, error) {
	client.collectionInput = input
	return backend.CollectionList{Items: []backend.Collection{{
		ID: "70000000-0000-4000-8000-000000000001", WorkspaceID: "10000000-0000-4000-8000-000000000001",
		Name: "Interviews", AssetCount: 2, Version: 1,
		CreatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC),
	}}}, nil
}

func (client *stubClient) ListTags(_ context.Context, input backend.ListPageInput) (backend.TagList, error) {
	client.tagInput = input
	color := "#FF8800"
	return backend.TagList{Items: []backend.Tag{{
		ID: "71000000-0000-4000-8000-000000000001", WorkspaceID: "10000000-0000-4000-8000-000000000001",
		Name: "Important", Color: &color, AssetCount: 1, CreatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
	}}}, nil
}

func (client *stubClient) ListAnnotations(
	_ context.Context,
	assetID string,
	input backend.ListPageInput,
) (backend.AnnotationList, error) {
	client.annotationAsset = assetID
	client.annotationInput = input
	endMS := int64(1200)
	return backend.AnnotationList{Items: []backend.Annotation{{
		ID: "72000000-0000-4000-8000-000000000001", WorkspaceID: "10000000-0000-4000-8000-000000000001",
		AssetID: assetID, Kind: "bookmark", StartMS: 500, EndMS: &endMS, Body: "Decision", Version: 1,
		CreatedBy: "20000000-0000-4000-8000-000000000001",
		CreatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
	}}}, nil
}

func (client *stubClient) GetProcessingStatus(_ context.Context, assetID string) (backend.ProcessingStatus, error) {
	client.processingAsset = assetID
	return backend.ProcessingStatus{
		AssetID: assetID, AssetStatus: "processing", Active: true,
		Jobs: []backend.ProcessingJob{{
			ID: "73000000-0000-4000-8000-000000000001", Kind: "mock_transcribe", State: "queued", MaxAttempts: 3,
			CreatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
		}},
		UpdatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
	}, nil
}

func (*stubClient) GetTranscriptionJob(context.Context, string) (backend.TranscriptionJob, error) {
	return backend.TranscriptionJob{
		ID: "73000000-0000-4000-8000-000000000001", WorkspaceID: "10000000-0000-4000-8000-000000000001",
		AssetID: "30000000-0000-4000-8000-000000000001", CreatedBy: "20000000-0000-4000-8000-000000000001",
		Kind: "mock_transcribe", State: "succeeded", Payload: []byte(`{"asset_id":"30000000-0000-4000-8000-000000000001"}`),
		Attempts: 1, MaxAttempts: 3, AvailableAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC),
	}, nil
}

func (*stubClient) ListTranscripts(context.Context, string) (backend.TranscriptList, error) {
	return backend.TranscriptList{Items: []backend.TranscriptSummary{{
		ID:               "40000000-0000-4000-8000-000000000001",
		AssetID:          "30000000-0000-4000-8000-000000000001",
		LatestRevisionID: "50000000-0000-4000-8000-000000000003",
	}}}, nil
}

func (*stubClient) GetTranscriptRevision(_ context.Context, revisionID string) (backend.TranscriptRevision, error) {
	parentID := ""
	if revisionID == "50000000-0000-4000-8000-000000000003" {
		parentID = "50000000-0000-4000-8000-000000000002"
	}
	confidence := 0.98
	return backend.TranscriptRevision{
		ID: revisionID, TranscriptID: "40000000-0000-4000-8000-000000000001",
		AssetID: "30000000-0000-4000-8000-000000000001", ParentRevisionID: parentID,
		Kind: "normalized", Language: "en-US", Text: "Quarterly recording",
		CreatedByType: "system", ReviewStatus: "pending", CreatedAt: time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
		Segments: []backend.TranscriptSegment{
			{ID: "60000000-0000-4000-8000-000000000001", Ordinal: 0, StartMS: 0, EndMS: 500, Text: "Before"},
			{ID: "60000000-0000-4000-8000-000000000002", Ordinal: 1, StartMS: 500, EndMS: 1200, Text: "Inside", Confidence: &confidence},
			{ID: "60000000-0000-4000-8000-000000000003", Ordinal: 2, StartMS: 1000, EndMS: 1500, Text: "After"},
		},
	}, nil
}

func (client *stubClient) recordWrite(name string) {
	if client.writeCalls == nil {
		client.writeCalls = make(map[string]int)
	}
	client.writeCalls[name]++
}

func (client *stubClient) StartTranscription(_ context.Context, assetID, key string) (backend.TranscriptionJob, error) {
	client.recordWrite("start_transcription")
	client.writeAssetID, client.writeKey = assetID, key
	return backend.TranscriptionJob{ID: "73000000-0000-4000-8000-000000000011", AssetID: assetID, State: "queued"}, nil
}

func (client *stubClient) StartCorrection(_ context.Context, revisionID, key string) (backend.TranscriptionJob, error) {
	client.recordWrite("start_llm_correction")
	client.writeRevisionID, client.writeKey = revisionID, key
	return backend.TranscriptionJob{ID: "73000000-0000-4000-8000-000000000012", State: "queued"}, nil
}

func (client *stubClient) UpdateAssetMetadata(
	_ context.Context,
	assetID string,
	input backend.UpdateAssetMetadataInput,
) (backend.Asset, error) {
	client.recordWrite("update_asset_metadata")
	client.writeAssetID, client.metadataInput = assetID, input
	return backend.Asset{ID: assetID, Title: input.Title, Language: input.Language, CollectionID: input.CollectionID, Version: input.ExpectedVersion + 1}, nil
}

func (client *stubClient) AddTags(_ context.Context, assetID string, tagIDs []string) (backend.TagMutationResult, error) {
	client.recordWrite("add_tags")
	client.writeAssetID, client.writeTagIDs = assetID, append([]string(nil), tagIDs...)
	return backend.TagMutationResult{AssetID: assetID, TagIDs: tagIDs, ChangedCount: 1}, nil
}

func (client *stubClient) RemoveTags(_ context.Context, assetID string, tagIDs []string) (backend.TagMutationResult, error) {
	client.recordWrite("remove_tags")
	client.writeAssetID, client.writeTagIDs = assetID, append([]string(nil), tagIDs...)
	return backend.TagMutationResult{AssetID: assetID, TagIDs: tagIDs, ChangedCount: 1}, nil
}

func (client *stubClient) CreateAnnotation(
	_ context.Context,
	assetID string,
	input backend.CreateAnnotationInput,
) (backend.Annotation, error) {
	client.recordWrite("create_annotation")
	client.writeAssetID, client.annotationWrite = assetID, input
	return backend.Annotation{ID: "72000000-0000-4000-8000-000000000011", AssetID: assetID, Kind: input.Kind, StartMS: input.StartMS, EndMS: input.EndMS, Body: input.Body}, nil
}

func (client *stubClient) CreateAudioClip(
	_ context.Context,
	assetID string,
	startMS,
	endMS int64,
) (backend.AudioClip, error) {
	client.recordWrite("create_audio_clip")
	client.writeAssetID, client.clipStartMS, client.clipEndMS = assetID, startMS, endMS
	return backend.AudioClip{
		ID: "76000000-0000-4000-8000-000000000011", AssetID: assetID,
		StartMS: startMS, EndMS: endMS, DurationMS: endMS - startMS,
		DownloadURL: "/api/v1/audio-clips/76000000-0000-4000-8000-000000000011",
	}, nil
}

func (client *stubClient) ExportTranscript(
	_ context.Context,
	revisionID,
	format string,
) (backend.TranscriptExport, error) {
	client.recordWrite("export_transcript")
	client.writeRevisionID, client.exportFormat = revisionID, format
	return backend.TranscriptExport{
		ID: "77000000-0000-4000-8000-000000000011", RevisionID: revisionID, Format: format,
		DownloadURL: "/api/v1/transcript-exports/77000000-0000-4000-8000-000000000011",
	}, nil
}

func (client *stubClient) ApproveTranscriptRevision(
	_ context.Context,
	revisionID string,
	acceptPending bool,
) (backend.ApprovalResult, error) {
	client.recordWrite("approve_transcript_revision")
	client.writeRevisionID, client.acceptPending = revisionID, acceptPending
	return backend.ApprovalResult{ApprovedRevision: backend.TranscriptRevision{ID: revisionID}}, nil
}

func TestGetSystemCapabilitiesTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- New(&stubClient{}, "test").Run(ctx, serverTransport)
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
	if output.ContractVersion != backend.SupportedContractVersion || len(output.Features) != 1 {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestReadToolsExposePaginationLineageAndExactTimeRange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stub := &stubClient{}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() { serverDone <- New(stub, "test").Run(ctx, serverTransport) }()
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

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 12 {
		t.Fatalf("tool count = %d, want 12", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatalf("tool %q is not marked read-only", tool.Name)
		}
	}

	searchResult := callTool(t, ctx, session, "search_assets", map[string]any{
		"query": " Quarterly ", "provider_id": "mock_asr", "speaker": " Alice ", "limit": 5,
	})
	search := decodeStructured[AssetListOutput](t, searchResult)
	if len(search.Items) != 1 || search.Items[0].Search == nil || len(search.Items[0].Search.Segments) != 1 ||
		stub.listInput.Query != "Quarterly" ||
		stub.listInput.ProviderID != "mock_asr" || stub.listInput.Speaker != " Alice " || stub.listInput.Limit != 5 {
		t.Fatalf("search output/input = (%+v, %+v)", search, stub.listInput)
	}

	assetResult := callTool(t, ctx, session, "get_asset_metadata", map[string]any{
		"asset_id": "30000000-0000-4000-8000-000000000001",
	})
	assetOutput := decodeStructured[GetAssetOutput](t, assetResult)
	if assetOutput.Asset.Status != "ready" {
		t.Fatalf("asset output = %+v", assetOutput)
	}

	collectionResult := callTool(t, ctx, session, "list_collections", map[string]any{
		"limit": 3, "cursor": "collections-next",
	})
	collections := decodeStructured[CollectionListOutput](t, collectionResult)
	if len(collections.Items) != 1 || stub.collectionInput != (backend.ListPageInput{Limit: 3, Cursor: "collections-next"}) {
		t.Fatalf("collection output/input = (%+v, %+v)", collections, stub.collectionInput)
	}

	tagResult := callTool(t, ctx, session, "list_tags", map[string]any{"limit": 4})
	tags := decodeStructured[TagListOutput](t, tagResult)
	if len(tags.Items) != 1 || stub.tagInput.Limit != 4 {
		t.Fatalf("tag output/input = (%+v, %+v)", tags, stub.tagInput)
	}

	annotationResult := callTool(t, ctx, session, "get_annotations", map[string]any{
		"asset_id": "30000000-0000-4000-8000-000000000001", "limit": 5,
	})
	annotations := decodeStructured[AnnotationListOutput](t, annotationResult)
	if len(annotations.Items) != 1 || annotations.Items[0].EndMS == nil || *annotations.Items[0].EndMS != 1200 ||
		stub.annotationAsset != "30000000-0000-4000-8000-000000000001" || stub.annotationInput.Limit != 5 {
		t.Fatalf("annotation output/input = (%+v, %q, %+v)", annotations, stub.annotationAsset, stub.annotationInput)
	}

	processingResult := callTool(t, ctx, session, "get_processing_status", map[string]any{
		"asset_id": "30000000-0000-4000-8000-000000000001",
	})
	processing := decodeStructured[GetProcessingStatusOutput](t, processingResult)
	if !processing.Status.Active || len(processing.Status.Jobs) != 1 || stub.processingAsset != processing.Status.AssetID {
		t.Fatalf("processing output/input = (%+v, %q)", processing, stub.processingAsset)
	}

	transcriptResult := callTool(t, ctx, session, "get_transcript", map[string]any{
		"revision_id": "50000000-0000-4000-8000-000000000003",
	})
	transcriptOutput := decodeStructured[GetTranscriptOutput](t, transcriptResult)
	if len(transcriptOutput.Revision.Segments) != 3 {
		t.Fatalf("transcript output = %+v", transcriptOutput)
	}

	segmentsResult := callTool(t, ctx, session, "get_transcript_segments", map[string]any{
		"revision_id": "50000000-0000-4000-8000-000000000003", "start_ms": 500, "end_ms": 1000,
	})
	segments := decodeStructured[GetTranscriptSegmentsOutput](t, segmentsResult)
	if len(segments.Segments) != 1 || segments.Segments[0].SegmentID != "60000000-0000-4000-8000-000000000002" ||
		segments.Segments[0].OverlapStart != 500 || segments.Segments[0].OverlapEnd != 1000 {
		t.Fatalf("segment range output = %+v", segments)
	}

	lineageResult := callTool(t, ctx, session, "list_transcript_revisions", map[string]any{
		"asset_id": "30000000-0000-4000-8000-000000000001", "limit": 1,
	})
	lineage := decodeStructured[ListTranscriptRevisionsOutput](t, lineageResult)
	if len(lineage.Items) != 1 || !lineage.Truncated || lineage.Items[0].SegmentCount != 3 {
		t.Fatalf("lineage output = %+v", lineage)
	}
}

func TestWriteToolsAreDefaultDisabledAndExplicitlyEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stub := &stubClient{}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- NewWithOptions(stub, "test", Options{EnableWrites: true}).Run(ctx, serverTransport)
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

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	writeNames := map[string]bool{
		"start_transcription": false, "start_llm_correction": false,
		"update_asset_metadata": false, "add_tags": false, "remove_tags": false,
		"create_annotation": false, "approve_transcript_revision": false,
		"create_audio_clip": false, "export_transcript": false,
	}
	if len(tools.Tools) != 21 {
		t.Fatalf("tool count = %d, want 21", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if _, expected := writeNames[tool.Name]; !expected {
			continue
		}
		writeNames[tool.Name] = true
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil {
			t.Fatalf("write tool annotations for %q = %+v", tool.Name, tool.Annotations)
		}
	}
	for name, found := range writeNames {
		if !found {
			t.Fatalf("write tool %q is missing", name)
		}
	}

	const (
		assetID      = "30000000-0000-4000-8000-000000000001"
		revisionID   = "50000000-0000-4000-8000-000000000003"
		collectionID = "70000000-0000-4000-8000-000000000001"
		tagID        = "71000000-0000-4000-8000-000000000001"
	)
	callTool(t, ctx, session, "start_transcription", map[string]any{"asset_id": assetID, "idempotency_key": "transcribe-1"})
	callTool(t, ctx, session, "start_llm_correction", map[string]any{"revision_id": revisionID, "idempotency_key": "correct-1"})
	callTool(t, ctx, session, "update_asset_metadata", map[string]any{
		"asset_id": assetID, "expected_version": 2, "title": "Updated", "language": "en-US", "collection_id": collectionID,
	})
	callTool(t, ctx, session, "add_tags", map[string]any{"asset_id": assetID, "tag_ids": []string{tagID}})
	callTool(t, ctx, session, "remove_tags", map[string]any{"asset_id": assetID, "tag_ids": []string{tagID}})
	callTool(t, ctx, session, "create_annotation", map[string]any{
		"asset_id": assetID, "kind": "note", "start_ms": 250, "body": "Decision",
	})
	callTool(t, ctx, session, "create_audio_clip", map[string]any{
		"asset_id": assetID, "start_ms": 500, "end_ms": 2000,
	})
	callTool(t, ctx, session, "export_transcript", map[string]any{
		"revision_id": revisionID, "format": "vtt",
	})
	callTool(t, ctx, session, "approve_transcript_revision", map[string]any{
		"revision_id": revisionID, "accept_pending": true,
	})
	for name := range writeNames {
		if stub.writeCalls[name] != 1 {
			t.Fatalf("write calls for %q = %d", name, stub.writeCalls[name])
		}
	}
	if stub.metadataInput.CollectionID == nil || *stub.metadataInput.CollectionID != collectionID ||
		stub.clipStartMS != 500 || stub.clipEndMS != 2_000 || stub.exportFormat != "vtt" || !stub.acceptPending {
		t.Fatalf(
			"write inputs = metadata %+v clip %d..%d export %q accept_pending %t",
			stub.metadataInput, stub.clipStartMS, stub.clipEndMS, stub.exportFormat, stub.acceptPending,
		)
	}
}

func TestRequiredResourceTemplatesReturnPublicAPIModels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() { serverDone <- New(&stubClient{}, "test").Run(ctx, serverTransport) }()
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

	templates, err := session.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListResourceTemplates() error = %v", err)
	}
	wantTemplates := map[string]bool{
		"voice-asset://assets/{asset_id}":                   false,
		"voice-asset://assets/{asset_id}/transcript/latest": false,
		"voice-asset://transcripts/{revision_id}":           false,
		"voice-asset://collections/{collection_id}":         false,
		"voice-asset://jobs/{job_id}":                       false,
	}
	for _, template := range templates.ResourceTemplates {
		if _, expected := wantTemplates[template.URITemplate]; expected && template.MIMEType == resourceMIMEType {
			wantTemplates[template.URITemplate] = true
		}
	}
	for template, found := range wantTemplates {
		if !found {
			t.Fatalf("required resource template %q is missing: %+v", template, templates.ResourceTemplates)
		}
	}

	asset := readResourceJSON[backend.Asset](t, ctx, session,
		"voice-asset://assets/30000000-0000-4000-8000-000000000001")
	if asset.Status != "ready" {
		t.Fatalf("asset resource = %+v", asset)
	}
	latest := readResourceJSON[backend.TranscriptRevision](t, ctx, session,
		"voice-asset://assets/30000000-0000-4000-8000-000000000001/transcript/latest")
	if latest.ID != "50000000-0000-4000-8000-000000000003" {
		t.Fatalf("latest transcript resource = %+v", latest)
	}
	revision := readResourceJSON[backend.TranscriptRevision](t, ctx, session,
		"voice-asset://transcripts/50000000-0000-4000-8000-000000000002")
	if revision.ID != "50000000-0000-4000-8000-000000000002" {
		t.Fatalf("revision resource = %+v", revision)
	}
	collection := readResourceJSON[backend.Collection](t, ctx, session,
		"voice-asset://collections/70000000-0000-4000-8000-000000000001")
	if collection.Name != "Interviews" {
		t.Fatalf("collection resource = %+v", collection)
	}
	job := readResourceJSON[backend.TranscriptionJob](t, ctx, session,
		"voice-asset://jobs/73000000-0000-4000-8000-000000000001")
	if job.State != "succeeded" {
		t.Fatalf("job resource = %+v", job)
	}
	if _, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "voice-asset://assets/not-a-uuid"}); err == nil {
		t.Fatal("invalid resource UUID was accepted")
	}
}

func TestRequiredPromptsEmbedUntrustedTranscriptDataSafely(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() { serverDone <- New(&stubClient{}, "test").Run(ctx, serverTransport) }()
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

	listed, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}
	want := map[string]bool{
		"summarize_recording": false, "extract_action_items": false,
		"extract_technical_terms": false, "compare_transcript_revisions": false,
		"prepare_meeting_minutes": false, "review_asr_quality": false,
	}
	if len(listed.Prompts) != len(want) {
		t.Fatalf("prompt count = %d, want %d", len(listed.Prompts), len(want))
	}
	for _, prompt := range listed.Prompts {
		if _, expected := want[prompt.Name]; expected {
			want[prompt.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("prompt %q is missing", name)
		}
	}

	const revisionID = "50000000-0000-4000-8000-000000000003"
	result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "summarize_recording", Arguments: map[string]string{
			"revision_id": revisionID, "focus": "decisions",
		},
	})
	if err != nil || len(result.Messages) != 1 {
		t.Fatalf("GetPrompt(summarize) = (%+v, %v)", result, err)
	}
	content, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(content.Text, untrustedTranscriptPolicy) ||
		!strings.Contains(content.Text, "BEGIN_UNTRUSTED_TRANSCRIPT_DATA") ||
		!strings.Contains(content.Text, revisionID) {
		t.Fatalf("summarize prompt content = %#v", result.Messages[0].Content)
	}

	comparison, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "compare_transcript_revisions", Arguments: map[string]string{
			"base_revision_id":      "50000000-0000-4000-8000-000000000002",
			"candidate_revision_id": revisionID,
		},
	})
	if err != nil || len(comparison.Messages) != 1 {
		t.Fatalf("GetPrompt(compare) = (%+v, %v)", comparison, err)
	}
	comparisonText := comparison.Messages[0].Content.(*mcp.TextContent).Text
	if !strings.Contains(comparisonText, "BEGIN_UNTRUSTED_BASE_TRANSCRIPT_DATA") ||
		!strings.Contains(comparisonText, "BEGIN_UNTRUSTED_CANDIDATE_TRANSCRIPT_DATA") {
		t.Fatalf("comparison prompt = %q", comparisonText)
	}

	if _, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "review_asr_quality", Arguments: map[string]string{"revision_id": "not-a-uuid"},
	}); err == nil {
		t.Fatal("invalid prompt revision UUID was accepted")
	}
}

func callTool(
	t *testing.T,
	ctx context.Context,
	session *mcp.ClientSession,
	name string,
	arguments any,
) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}
	if result.IsError {
		t.Fatalf("CallTool(%s) returned tool error: %#v", name, result.Content)
	}
	return result
}

func decodeStructured[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var output T
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	return output
}

func readResourceJSON[T any](
	t *testing.T,
	ctx context.Context,
	session *mcp.ClientSession,
	uri string,
) T {
	t.Helper()
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("ReadResource(%s) error = %v", uri, err)
	}
	if len(result.Contents) != 1 || result.Contents[0].MIMEType != resourceMIMEType || result.Contents[0].URI != uri {
		t.Fatalf("ReadResource(%s) contents = %+v", uri, result.Contents)
	}
	var output T
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &output); err != nil {
		t.Fatalf("decode resource %s: %v", uri, err)
	}
	return output
}
