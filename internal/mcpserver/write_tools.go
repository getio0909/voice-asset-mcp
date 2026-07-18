package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getio0909/voice-asset-mcp/internal/backend"
)

type StartTranscriptionInput struct {
	AssetID        string `json:"asset_id" jsonschema:"VoiceAsset asset UUID"`
	IdempotencyKey string `json:"idempotency_key" jsonschema:"Caller-generated retry key, 1 to 200 characters"`
}

type StartCorrectionInput struct {
	RevisionID     string `json:"revision_id" jsonschema:"Source transcript revision UUID"`
	IdempotencyKey string `json:"idempotency_key" jsonschema:"Caller-generated retry key, 1 to 200 characters"`
}

type JobOutput struct {
	Job backend.TranscriptionJob `json:"job" jsonschema:"Accepted durable background job"`
}

type UpdateAssetMetadataInput struct {
	AssetID         string  `json:"asset_id" jsonschema:"VoiceAsset asset UUID"`
	ExpectedVersion int64   `json:"expected_version" jsonschema:"Current positive asset version from get_asset_metadata"`
	Title           string  `json:"title" jsonschema:"Complete replacement title"`
	Language        string  `json:"language" jsonschema:"Complete replacement BCP 47 language tag"`
	CollectionID    *string `json:"collection_id,omitempty" jsonschema:"Collection UUID; omit to remove the collection assignment"`
}

type UpdateAssetMetadataOutput struct {
	Asset backend.Asset `json:"asset" jsonschema:"Updated asset and incremented version"`
}

type MutateTagsInput struct {
	AssetID string   `json:"asset_id" jsonschema:"VoiceAsset asset UUID"`
	TagIDs  []string `json:"tag_ids" jsonschema:"One to 100 unique workspace tag UUIDs"`
}

type MutateTagsOutput struct {
	Result backend.TagMutationResult `json:"result" jsonschema:"Requested tag IDs and changed assignment count"`
}

type CreateAnnotationInput struct {
	AssetID string `json:"asset_id" jsonschema:"VoiceAsset asset UUID"`
	Kind    string `json:"kind" jsonschema:"Annotation kind: bookmark or note"`
	StartMS int64  `json:"start_ms" jsonschema:"Inclusive non-negative start time in milliseconds"`
	EndMS   *int64 `json:"end_ms,omitempty" jsonschema:"Optional exclusive end time in milliseconds"`
	Body    string `json:"body" jsonschema:"Annotation text; required for notes"`
}

type CreateAnnotationOutput struct {
	Annotation backend.Annotation `json:"annotation" jsonschema:"Created annotation with exact millisecond range"`
}

type CreateAudioClipInput struct {
	AssetID string `json:"asset_id" jsonschema:"VoiceAsset asset UUID"`
	StartMS int64  `json:"start_ms" jsonschema:"Inclusive non-negative start time in milliseconds"`
	EndMS   int64  `json:"end_ms" jsonschema:"Exclusive end time; at most five minutes after start_ms"`
}

type CreateAudioClipOutput struct {
	Clip backend.AudioClip `json:"clip" jsonschema:"Clip metadata and one-hour authenticated download URL; audio is never embedded"`
}

type ExportTranscriptInput struct {
	RevisionID string `json:"revision_id" jsonschema:"Immutable transcript revision UUID"`
	Format     string `json:"format" jsonschema:"Export format: json, markdown, srt, or vtt"`
}

type ExportTranscriptOutput struct {
	Export backend.TranscriptExport `json:"export" jsonschema:"Export metadata and one-hour authenticated download URL"`
}

type ApproveTranscriptRevisionInput struct {
	RevisionID    string `json:"revision_id" jsonschema:"Corrected transcript revision UUID"`
	AcceptPending bool   `json:"accept_pending,omitempty" jsonschema:"Accept undecided changes; defaults to conservative false"`
}

type ApproveTranscriptRevisionOutput struct {
	Approval backend.ApprovalResult `json:"approval" jsonschema:"Human and immutable approved transcript revisions"`
}

func addWriteTools(server *mcp.Server, client voiceAssetWriter) {
	mcp.AddTool(server, writeTool(
		"start_transcription", "Start transcription",
		"Queue transcription for a ready asset. Retries are safe only with the same idempotency_key.", false, true,
	), func(ctx context.Context, _ *mcp.CallToolRequest, input StartTranscriptionInput) (*mcp.CallToolResult, JobOutput, error) {
		if err := validateUUID("asset_id", input.AssetID); err != nil {
			return nil, JobOutput{}, err
		}
		if err := validateIdempotencyKey(input.IdempotencyKey); err != nil {
			return nil, JobOutput{}, err
		}
		job, err := client.StartTranscription(ctx, input.AssetID, input.IdempotencyKey)
		return nil, JobOutput{Job: job}, err
	})

	mcp.AddTool(server, writeTool(
		"start_llm_correction", "Start LLM correction",
		"Queue LLM correction from an immutable transcript revision. Retries require the same idempotency_key.", false, true,
	), func(ctx context.Context, _ *mcp.CallToolRequest, input StartCorrectionInput) (*mcp.CallToolResult, JobOutput, error) {
		if err := validateUUID("revision_id", input.RevisionID); err != nil {
			return nil, JobOutput{}, err
		}
		if err := validateIdempotencyKey(input.IdempotencyKey); err != nil {
			return nil, JobOutput{}, err
		}
		job, err := client.StartCorrection(ctx, input.RevisionID, input.IdempotencyKey)
		return nil, JobOutput{Job: job}, err
	})

	mcp.AddTool(server, writeTool(
		"update_asset_metadata", "Update asset metadata",
		"Replace title, language, and optional collection using optimistic version control.", true, true,
	), func(ctx context.Context, _ *mcp.CallToolRequest, input UpdateAssetMetadataInput) (*mcp.CallToolResult, UpdateAssetMetadataOutput, error) {
		if err := validateUUID("asset_id", input.AssetID); err != nil {
			return nil, UpdateAssetMetadataOutput{}, err
		}
		if input.ExpectedVersion < 1 || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Language) == "" {
			return nil, UpdateAssetMetadataOutput{}, fmt.Errorf("expected_version, title, and language are required")
		}
		if input.CollectionID != nil {
			if err := validateUUID("collection_id", *input.CollectionID); err != nil {
				return nil, UpdateAssetMetadataOutput{}, err
			}
		}
		updated, err := client.UpdateAssetMetadata(ctx, input.AssetID, backend.UpdateAssetMetadataInput{
			Title: input.Title, Language: input.Language, CollectionID: input.CollectionID,
			ExpectedVersion: input.ExpectedVersion,
		})
		return nil, UpdateAssetMetadataOutput{Asset: updated}, err
	})

	addTagMutationTool(server, client, true)
	addTagMutationTool(server, client, false)

	mcp.AddTool(server, writeTool(
		"create_annotation", "Create annotation",
		"Create an audited bookmark or note with exact integer-millisecond timing.", false, false,
	), func(ctx context.Context, _ *mcp.CallToolRequest, input CreateAnnotationInput) (*mcp.CallToolResult, CreateAnnotationOutput, error) {
		if err := validateUUID("asset_id", input.AssetID); err != nil {
			return nil, CreateAnnotationOutput{}, err
		}
		kind := strings.TrimSpace(input.Kind)
		body := strings.TrimSpace(input.Body)
		if (kind != "bookmark" && kind != "note") || input.StartMS < 0 ||
			(input.EndMS != nil && *input.EndMS <= input.StartMS) || (kind == "note" && body == "") {
			return nil, CreateAnnotationOutput{}, fmt.Errorf("annotation kind, range, or body is invalid")
		}
		created, err := client.CreateAnnotation(ctx, input.AssetID, backend.CreateAnnotationInput{
			Kind: kind, StartMS: input.StartMS, EndMS: input.EndMS, Body: body,
		})
		return nil, CreateAnnotationOutput{Annotation: created}, err
	})

	mcp.AddTool(server, writeTool(
		"create_audio_clip", "Create audio clip",
		"Create an audited clip of at most five minutes. Returns metadata and an expiring authenticated download URL, never Base64 audio.", false, false,
	), func(ctx context.Context, _ *mcp.CallToolRequest, input CreateAudioClipInput) (*mcp.CallToolResult, CreateAudioClipOutput, error) {
		if err := validateUUID("asset_id", input.AssetID); err != nil {
			return nil, CreateAudioClipOutput{}, err
		}
		if input.StartMS < 0 || input.EndMS <= input.StartMS || input.EndMS-input.StartMS > 5*60*1000 {
			return nil, CreateAudioClipOutput{}, fmt.Errorf("clip range must be positive and no longer than five minutes")
		}
		created, err := client.CreateAudioClip(ctx, input.AssetID, input.StartMS, input.EndMS)
		return nil, CreateAudioClipOutput{Clip: created}, err
	})

	mcp.AddTool(server, writeTool(
		"export_transcript", "Export transcript",
		"Serialize an immutable revision as JSON, Markdown, SRT, or WebVTT and return an expiring authenticated download URL.", false, false,
	), func(ctx context.Context, _ *mcp.CallToolRequest, input ExportTranscriptInput) (*mcp.CallToolResult, ExportTranscriptOutput, error) {
		if err := validateUUID("revision_id", input.RevisionID); err != nil {
			return nil, ExportTranscriptOutput{}, err
		}
		format := strings.ToLower(strings.TrimSpace(input.Format))
		if format != "json" && format != "markdown" && format != "srt" && format != "vtt" {
			return nil, ExportTranscriptOutput{}, fmt.Errorf("format must be json, markdown, srt, or vtt")
		}
		exported, err := client.ExportTranscript(ctx, input.RevisionID, format)
		return nil, ExportTranscriptOutput{Export: exported}, err
	})

	mcp.AddTool(server, writeTool(
		"approve_transcript_revision", "Approve transcript revision",
		"Create immutable human and approved revisions. By default, undecided changes remain rejected.", true, false,
	), func(ctx context.Context, _ *mcp.CallToolRequest, input ApproveTranscriptRevisionInput) (*mcp.CallToolResult, ApproveTranscriptRevisionOutput, error) {
		if err := validateUUID("revision_id", input.RevisionID); err != nil {
			return nil, ApproveTranscriptRevisionOutput{}, err
		}
		approved, err := client.ApproveTranscriptRevision(ctx, input.RevisionID, input.AcceptPending)
		return nil, ApproveTranscriptRevisionOutput{Approval: approved}, err
	})
}

func addTagMutationTool(server *mcp.Server, client voiceAssetWriter, add bool) {
	name, title, description := "add_tags", "Add tags", "Add existing workspace tags to an asset."
	destructive := false
	if !add {
		name, title, description = "remove_tags", "Remove tags", "Remove existing workspace tags from an asset."
		destructive = true
	}
	mcp.AddTool(server, writeTool(name, title, description, destructive, false),
		func(ctx context.Context, _ *mcp.CallToolRequest, input MutateTagsInput) (*mcp.CallToolResult, MutateTagsOutput, error) {
			if err := validateUUID("asset_id", input.AssetID); err != nil {
				return nil, MutateTagsOutput{}, err
			}
			if len(input.TagIDs) < 1 || len(input.TagIDs) > 100 {
				return nil, MutateTagsOutput{}, fmt.Errorf("tag_ids must contain 1 to 100 UUIDs")
			}
			seen := make(map[string]struct{}, len(input.TagIDs))
			for _, tagID := range input.TagIDs {
				if err := validateUUID("tag_id", tagID); err != nil {
					return nil, MutateTagsOutput{}, err
				}
				canonical := strings.ToLower(tagID)
				if _, duplicate := seen[canonical]; duplicate {
					return nil, MutateTagsOutput{}, fmt.Errorf("tag_ids must be unique")
				}
				seen[canonical] = struct{}{}
			}
			var (
				result backend.TagMutationResult
				err    error
			)
			if add {
				result, err = client.AddTags(ctx, input.AssetID, input.TagIDs)
			} else {
				result, err = client.RemoveTags(ctx, input.AssetID, input.TagIDs)
			}
			return nil, MutateTagsOutput{Result: result}, err
		})
}

func writeTool(name, title, description string, destructive, idempotent bool) *mcp.Tool {
	closedWorld := false
	return &mcp.Tool{
		Name: name, Title: title, Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false, DestructiveHint: &destructive,
			IdempotentHint: idempotent, OpenWorldHint: &closedWorld,
		},
	}
}

func validateIdempotencyKey(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 200 {
		return fmt.Errorf("idempotency_key must contain 1 to 200 characters")
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf("idempotency_key must not contain control characters")
		}
	}
	return nil
}
