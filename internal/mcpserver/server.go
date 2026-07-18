package mcpserver

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
)

type voiceAssetReader interface {
	GetSystemCapabilities(context.Context) (backend.Capabilities, error)
	ListAssets(context.Context, backend.ListAssetsInput) (backend.AssetList, error)
	GetAsset(context.Context, string) (backend.Asset, error)
	GetCollection(context.Context, string) (backend.Collection, error)
	ListCollections(context.Context, backend.ListPageInput) (backend.CollectionList, error)
	ListTags(context.Context, backend.ListPageInput) (backend.TagList, error)
	ListAnnotations(context.Context, string, backend.ListPageInput) (backend.AnnotationList, error)
	GetProcessingStatus(context.Context, string) (backend.ProcessingStatus, error)
	GetTranscriptionJob(context.Context, string) (backend.TranscriptionJob, error)
	ListTranscripts(context.Context, string) (backend.TranscriptList, error)
	GetTranscriptRevision(context.Context, string) (backend.TranscriptRevision, error)
}

type voiceAssetWriter interface {
	voiceAssetReader
	StartTranscription(context.Context, string, string) (backend.TranscriptionJob, error)
	StartCorrection(context.Context, string, string) (backend.TranscriptionJob, error)
	UpdateAssetMetadata(context.Context, string, backend.UpdateAssetMetadataInput) (backend.Asset, error)
	AddTags(context.Context, string, []string) (backend.TagMutationResult, error)
	RemoveTags(context.Context, string, []string) (backend.TagMutationResult, error)
	CreateAnnotation(context.Context, string, backend.CreateAnnotationInput) (backend.Annotation, error)
	CreateAudioClip(context.Context, string, int64, int64) (backend.AudioClip, error)
	ExportTranscript(context.Context, string, string) (backend.TranscriptExport, error)
	ApproveTranscriptRevision(context.Context, string, bool) (backend.ApprovalResult, error)
}

type Options struct {
	EnableWrites bool
}

type getSystemCapabilitiesInput struct{}

type GetSystemCapabilitiesOutput struct {
	ServerVersion   string   `json:"server_version" jsonschema:"VoiceAsset Server version"`
	APIVersion      string   `json:"api_version" jsonschema:"REST API version"`
	ContractVersion string   `json:"contract_version" jsonschema:"OpenAPI contract version"`
	Features        []string `json:"features" jsonschema:"Advertised server capabilities"`
}

type ListAssetsInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"Page size from 1 to 100; defaults to 20"`
	Cursor string `json:"cursor,omitempty" jsonschema:"Opaque cursor from the preceding page"`
}

type SearchAssetsInput struct {
	Query      string `json:"query" jsonschema:"Required PostgreSQL full-text terms for titles and latest Transcript Segments"`
	ProviderID string `json:"provider_id,omitempty" jsonschema:"Optional mock_asr, aliyun_asr, or tencent_asr filter"`
	Speaker    string `json:"speaker,omitempty" jsonschema:"Optional case-insensitive exact Speaker label"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Page size from 1 to 100; defaults to 20"`
	Cursor     string `json:"cursor,omitempty" jsonschema:"Opaque cursor from the preceding page for the same complete filter set"`
}

type AssetListOutput struct {
	Items      []backend.Asset `json:"items" jsonschema:"Workspace-scoped assets in stable descending creation order"`
	NextCursor *string         `json:"next_cursor,omitempty" jsonschema:"Opaque cursor for the next page"`
}

type AssetIDInput struct {
	AssetID string `json:"asset_id" jsonschema:"VoiceAsset asset UUID"`
}

type GetAssetOutput struct {
	Asset backend.Asset `json:"asset" jsonschema:"Workspace-scoped asset record"`
}

type ListOrganizationInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"Page size from 1 to 100; defaults to 50"`
	Cursor string `json:"cursor,omitempty" jsonschema:"Opaque cursor from the preceding page"`
}

type CollectionListOutput struct {
	Items      []backend.Collection `json:"items" jsonschema:"Workspace collections in stable descending creation order"`
	NextCursor *string              `json:"next_cursor,omitempty" jsonschema:"Opaque cursor for the next page"`
}

type TagListOutput struct {
	Items      []backend.Tag `json:"items" jsonschema:"Workspace tags in stable descending creation order"`
	NextCursor *string       `json:"next_cursor,omitempty" jsonschema:"Opaque cursor for the next page"`
}

type GetAnnotationsInput struct {
	AssetID string `json:"asset_id" jsonschema:"VoiceAsset asset UUID"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Page size from 1 to 100; defaults to 50"`
	Cursor  string `json:"cursor,omitempty" jsonschema:"Opaque cursor from the preceding page for the same asset"`
}

type AnnotationListOutput struct {
	Items      []backend.Annotation `json:"items" jsonschema:"Annotations with exact integer-millisecond time citations"`
	NextCursor *string              `json:"next_cursor,omitempty" jsonschema:"Opaque cursor for the next page"`
}

type GetProcessingStatusOutput struct {
	Status backend.ProcessingStatus `json:"status" jsonschema:"Asset state and at most the 20 most recent processing jobs"`
}

type RevisionIDInput struct {
	RevisionID string `json:"revision_id" jsonschema:"Immutable transcript revision UUID"`
}

type GetTranscriptOutput struct {
	Revision backend.TranscriptRevision `json:"revision" jsonschema:"Immutable transcript revision and complete segment timeline"`
}

type ListTranscriptRevisionsInput struct {
	AssetID string `json:"asset_id" jsonschema:"VoiceAsset asset UUID"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Maximum lineage revisions to return from 1 to 100; defaults to 20"`
}

type TranscriptRevisionInfo struct {
	ID               string `json:"id"`
	TranscriptID     string `json:"transcript_id"`
	AssetID          string `json:"asset_id"`
	ParentRevisionID string `json:"parent_revision_id,omitempty"`
	Kind             string `json:"kind"`
	Language         string `json:"language"`
	CreatedByType    string `json:"created_by_type"`
	ReviewStatus     string `json:"review_status"`
	CreatedAt        string `json:"created_at"`
	SegmentCount     int    `json:"segment_count"`
}

type ListTranscriptRevisionsOutput struct {
	Items     []TranscriptRevisionInfo `json:"items" jsonschema:"Newest-to-oldest revision lineage entries"`
	Truncated bool                     `json:"truncated" jsonschema:"True when the requested limit stopped lineage traversal"`
}

type GetTranscriptSegmentsInput struct {
	RevisionID string `json:"revision_id" jsonschema:"Immutable transcript revision UUID"`
	StartMS    int64  `json:"start_ms" jsonschema:"Inclusive range start in milliseconds"`
	EndMS      int64  `json:"end_ms" jsonschema:"Exclusive range end in milliseconds; must exceed start_ms"`
}

type TranscriptSegmentCitation struct {
	AssetID      string   `json:"asset_id"`
	RevisionID   string   `json:"revision_id"`
	SegmentID    string   `json:"segment_id"`
	Ordinal      int      `json:"ordinal"`
	StartMS      int64    `json:"start_ms"`
	EndMS        int64    `json:"end_ms"`
	OverlapStart int64    `json:"overlap_start_ms"`
	OverlapEnd   int64    `json:"overlap_end_ms"`
	Speaker      *string  `json:"speaker"`
	Text         string   `json:"text"`
	Confidence   *float64 `json:"confidence"`
}

type GetTranscriptSegmentsOutput struct {
	AssetID    string                      `json:"asset_id"`
	RevisionID string                      `json:"revision_id"`
	StartMS    int64                       `json:"start_ms"`
	EndMS      int64                       `json:"end_ms"`
	Segments   []TranscriptSegmentCitation `json:"segments"`
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func New(client voiceAssetReader, version string) *mcp.Server {
	return newReadServer(client, version)
}

func NewWithOptions(client voiceAssetWriter, version string, options Options) *mcp.Server {
	server := newReadServer(client, version)
	if options.EnableWrites {
		addWriteTools(server, client)
	}
	return server
}

func newReadServer(client voiceAssetReader, version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "voice-asset-mcp", Version: version}, nil)
	addCapabilityTool(server, client)
	addAssetTools(server, client)
	addOrganizationTools(server, client)
	addTranscriptTools(server, client)
	addResourceTemplates(server, client)
	addPrompts(server, client)
	return server
}

func addCapabilityTool(server *mcp.Server, client voiceAssetReader) {
	mcp.AddTool(server, readOnlyTool(
		"get_system_capabilities",
		"Get system capabilities",
		"Read the connected VoiceAsset Server capability contract.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, _ getSystemCapabilitiesInput) (*mcp.CallToolResult, GetSystemCapabilitiesOutput, error) {
		capabilities, err := client.GetSystemCapabilities(ctx)
		if err != nil {
			return nil, GetSystemCapabilitiesOutput{}, err
		}
		return nil, GetSystemCapabilitiesOutput{
			ServerVersion: capabilities.ServerVersion, APIVersion: capabilities.APIVersion,
			ContractVersion: capabilities.ContractVersion, Features: capabilities.Features,
		}, nil
	})
}

func addAssetTools(server *mcp.Server, client voiceAssetReader) {
	mcp.AddTool(server, readOnlyTool(
		"list_assets", "List assets", "List workspace assets with stable opaque-cursor pagination.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input ListAssetsInput) (*mcp.CallToolResult, AssetListOutput, error) {
		if err := validateLimit(input.Limit); err != nil {
			return nil, AssetListOutput{}, err
		}
		result, err := client.ListAssets(ctx, backend.ListAssetsInput{Limit: input.Limit, Cursor: input.Cursor})
		if err != nil {
			return nil, AssetListOutput{}, err
		}
		return nil, AssetListOutput{Items: result.Items, NextCursor: result.NextCursor}, nil
	})

	mcp.AddTool(server, readOnlyTool(
		"search_assets", "Search assets", "Search titles and latest Transcript Segments with Provider/Speaker filters, bounded timecode hits, and opaque-cursor pagination.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input SearchAssetsInput) (*mcp.CallToolResult, AssetListOutput, error) {
		query := strings.TrimSpace(input.Query)
		if query == "" {
			return nil, AssetListOutput{}, fmt.Errorf("query is required")
		}
		if err := validateLimit(input.Limit); err != nil {
			return nil, AssetListOutput{}, err
		}
		result, err := client.ListAssets(ctx, backend.ListAssetsInput{
			Query: query, ProviderID: input.ProviderID, Speaker: input.Speaker,
			Limit: input.Limit, Cursor: input.Cursor,
		})
		if err != nil {
			return nil, AssetListOutput{}, err
		}
		return nil, AssetListOutput{Items: result.Items, NextCursor: result.NextCursor}, nil
	})

	getAsset := func(ctx context.Context, _ *mcp.CallToolRequest, input AssetIDInput) (*mcp.CallToolResult, GetAssetOutput, error) {
		if err := validateUUID("asset_id", input.AssetID); err != nil {
			return nil, GetAssetOutput{}, err
		}
		result, err := client.GetAsset(ctx, input.AssetID)
		if err != nil {
			return nil, GetAssetOutput{}, err
		}
		return nil, GetAssetOutput{Asset: result}, nil
	}
	mcp.AddTool(server, readOnlyTool(
		"get_asset", "Get asset", "Read one workspace-scoped VoiceAsset asset.",
	), getAsset)
	mcp.AddTool(server, readOnlyTool(
		"get_asset_metadata", "Get asset metadata", "Read the language, state, duration, version, and timestamps for one asset.",
	), getAsset)
}

func addOrganizationTools(server *mcp.Server, client voiceAssetReader) {
	mcp.AddTool(server, readOnlyTool(
		"list_collections", "List collections", "List workspace collections and non-trashed asset counts with stable pagination.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input ListOrganizationInput) (*mcp.CallToolResult, CollectionListOutput, error) {
		if err := validateLimit(input.Limit); err != nil {
			return nil, CollectionListOutput{}, err
		}
		result, err := client.ListCollections(ctx, backend.ListPageInput{Limit: input.Limit, Cursor: input.Cursor})
		if err != nil {
			return nil, CollectionListOutput{}, err
		}
		return nil, CollectionListOutput{Items: result.Items, NextCursor: result.NextCursor}, nil
	})

	mcp.AddTool(server, readOnlyTool(
		"list_tags", "List tags", "List workspace tags and non-trashed asset counts with stable pagination.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input ListOrganizationInput) (*mcp.CallToolResult, TagListOutput, error) {
		if err := validateLimit(input.Limit); err != nil {
			return nil, TagListOutput{}, err
		}
		result, err := client.ListTags(ctx, backend.ListPageInput{Limit: input.Limit, Cursor: input.Cursor})
		if err != nil {
			return nil, TagListOutput{}, err
		}
		return nil, TagListOutput{Items: result.Items, NextCursor: result.NextCursor}, nil
	})

	mcp.AddTool(server, readOnlyTool(
		"get_annotations", "Get annotations", "Read an asset's bookmarks and notes with exact integer-millisecond ranges.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input GetAnnotationsInput) (*mcp.CallToolResult, AnnotationListOutput, error) {
		if err := validateUUID("asset_id", input.AssetID); err != nil {
			return nil, AnnotationListOutput{}, err
		}
		if err := validateLimit(input.Limit); err != nil {
			return nil, AnnotationListOutput{}, err
		}
		result, err := client.ListAnnotations(
			ctx, input.AssetID, backend.ListPageInput{Limit: input.Limit, Cursor: input.Cursor},
		)
		if err != nil {
			return nil, AnnotationListOutput{}, err
		}
		return nil, AnnotationListOutput{Items: result.Items, NextCursor: result.NextCursor}, nil
	})

	mcp.AddTool(server, readOnlyTool(
		"get_processing_status", "Get processing status", "Read an asset's current state and at most its 20 most recent durable jobs.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input AssetIDInput) (*mcp.CallToolResult, GetProcessingStatusOutput, error) {
		if err := validateUUID("asset_id", input.AssetID); err != nil {
			return nil, GetProcessingStatusOutput{}, err
		}
		result, err := client.GetProcessingStatus(ctx, input.AssetID)
		if err != nil {
			return nil, GetProcessingStatusOutput{}, err
		}
		return nil, GetProcessingStatusOutput{Status: result}, nil
	})
}

func addTranscriptTools(server *mcp.Server, client voiceAssetReader) {
	mcp.AddTool(server, readOnlyTool(
		"get_transcript", "Get transcript", "Read a specified immutable transcript revision and its complete segment timeline.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input RevisionIDInput) (*mcp.CallToolResult, GetTranscriptOutput, error) {
		if err := validateUUID("revision_id", input.RevisionID); err != nil {
			return nil, GetTranscriptOutput{}, err
		}
		revision, err := client.GetTranscriptRevision(ctx, input.RevisionID)
		if err != nil {
			return nil, GetTranscriptOutput{}, err
		}
		return nil, GetTranscriptOutput{Revision: revision}, nil
	})

	mcp.AddTool(server, readOnlyTool(
		"list_transcript_revisions", "List transcript revisions", "Follow each asset transcript's latest immutable parent lineage, newest first.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input ListTranscriptRevisionsInput) (*mcp.CallToolResult, ListTranscriptRevisionsOutput, error) {
		if err := validateUUID("asset_id", input.AssetID); err != nil {
			return nil, ListTranscriptRevisionsOutput{}, err
		}
		limit := input.Limit
		if limit == 0 {
			limit = 20
		}
		if err := validateLimit(limit); err != nil {
			return nil, ListTranscriptRevisionsOutput{}, err
		}
		return listTranscriptRevisionLineages(ctx, client, strings.ToLower(input.AssetID), limit)
	})

	mcp.AddTool(server, readOnlyTool(
		"get_transcript_segments", "Get transcript segments", "Read segment citations overlapping the exact half-open millisecond range [start_ms, end_ms).",
	), func(ctx context.Context, _ *mcp.CallToolRequest, input GetTranscriptSegmentsInput) (*mcp.CallToolResult, GetTranscriptSegmentsOutput, error) {
		if err := validateUUID("revision_id", input.RevisionID); err != nil {
			return nil, GetTranscriptSegmentsOutput{}, err
		}
		if input.StartMS < 0 || input.EndMS <= input.StartMS {
			return nil, GetTranscriptSegmentsOutput{}, fmt.Errorf("time range must satisfy 0 <= start_ms < end_ms")
		}
		revision, err := client.GetTranscriptRevision(ctx, input.RevisionID)
		if err != nil {
			return nil, GetTranscriptSegmentsOutput{}, err
		}
		segments := make([]TranscriptSegmentCitation, 0)
		for _, segment := range revision.Segments {
			if segment.EndMS <= input.StartMS || segment.StartMS >= input.EndMS {
				continue
			}
			segments = append(segments, TranscriptSegmentCitation{
				AssetID: revision.AssetID, RevisionID: revision.ID, SegmentID: segment.ID,
				Ordinal: segment.Ordinal, StartMS: segment.StartMS, EndMS: segment.EndMS,
				OverlapStart: max(segment.StartMS, input.StartMS), OverlapEnd: min(segment.EndMS, input.EndMS),
				Speaker: segment.Speaker, Text: segment.Text, Confidence: segment.Confidence,
			})
		}
		return nil, GetTranscriptSegmentsOutput{
			AssetID: revision.AssetID, RevisionID: revision.ID, StartMS: input.StartMS,
			EndMS: input.EndMS, Segments: segments,
		}, nil
	})
}

func listTranscriptRevisionLineages(
	ctx context.Context,
	client voiceAssetReader,
	assetID string,
	limit int,
) (*mcp.CallToolResult, ListTranscriptRevisionsOutput, error) {
	transcripts, err := client.ListTranscripts(ctx, assetID)
	if err != nil {
		return nil, ListTranscriptRevisionsOutput{}, err
	}
	items := make([]TranscriptRevisionInfo, 0, min(limit, len(transcripts.Items)*4))
	seen := make(map[string]struct{})
	for transcriptIndex, summary := range transcripts.Items {
		revisionID := summary.LatestRevisionID
		for revisionID != "" {
			if _, duplicate := seen[revisionID]; duplicate {
				return nil, ListTranscriptRevisionsOutput{}, fmt.Errorf("server returned a cyclic transcript lineage")
			}
			seen[revisionID] = struct{}{}
			revision, getErr := client.GetTranscriptRevision(ctx, revisionID)
			if getErr != nil {
				return nil, ListTranscriptRevisionsOutput{}, getErr
			}
			if !strings.EqualFold(revision.AssetID, assetID) {
				return nil, ListTranscriptRevisionsOutput{}, fmt.Errorf("server returned a revision for another asset")
			}
			items = append(items, TranscriptRevisionInfo{
				ID: revision.ID, TranscriptID: revision.TranscriptID, AssetID: revision.AssetID,
				ParentRevisionID: revision.ParentRevisionID, Kind: revision.Kind, Language: revision.Language,
				CreatedByType: revision.CreatedByType, ReviewStatus: revision.ReviewStatus,
				CreatedAt:    revision.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
				SegmentCount: len(revision.Segments),
			})
			if len(items) == limit {
				truncated := revision.ParentRevisionID != "" || transcriptIndex < len(transcripts.Items)-1
				return nil, ListTranscriptRevisionsOutput{Items: items, Truncated: truncated}, nil
			}
			revisionID = revision.ParentRevisionID
		}
	}
	return nil, ListTranscriptRevisionsOutput{Items: items, Truncated: false}, nil
}

func readOnlyTool(name, title, description string) *mcp.Tool {
	closedWorld := false
	return &mcp.Tool{
		Name: name, Title: title, Description: description,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &closedWorld},
	}
}

func validateLimit(limit int) error {
	if limit < 0 || limit > 100 {
		return fmt.Errorf("limit must be 0 or between 1 and 100")
	}
	return nil
}

func validateUUID(name, value string) error {
	if strings.TrimSpace(value) != value || !uuidPattern.MatchString(value) {
		return fmt.Errorf("%s must be a UUID", name)
	}
	return nil
}
