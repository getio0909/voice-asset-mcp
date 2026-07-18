package backend

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type UpdateAssetMetadataInput struct {
	Title           string
	Language        string
	CollectionID    *string
	ExpectedVersion int64
}

type TagMutationResult struct {
	AssetID      string   `json:"asset_id"`
	TagIDs       []string `json:"tag_ids"`
	ChangedCount int      `json:"changed_count"`
}

type CreateAnnotationInput struct {
	Kind    string
	StartMS int64
	EndMS   *int64
	Body    string
}

type AudioClip struct {
	ID          string    `json:"id"`
	AssetID     string    `json:"asset_id"`
	StartMS     int64     `json:"start_ms"`
	EndMS       int64     `json:"end_ms"`
	DurationMS  int64     `json:"duration_ms"`
	MIMEType    string    `json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	SHA256      string    `json:"sha256"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type TranscriptExport struct {
	ID          string    `json:"id"`
	AssetID     string    `json:"asset_id"`
	RevisionID  string    `json:"revision_id"`
	Format      string    `json:"format"`
	MIMEType    string    `json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	SHA256      string    `json:"sha256"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type ReviewRecord struct {
	ID                  string    `json:"id"`
	RevisionID          string    `json:"revision_id"`
	ReviewerID          string    `json:"reviewer_id"`
	Action              string    `json:"action"`
	ChangeIndex         *int      `json:"change_index,omitempty"`
	ResultingRevisionID string    `json:"resulting_revision_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

type ApprovalResult struct {
	ReviewRecord     ReviewRecord       `json:"review"`
	HumanRevision    TranscriptRevision `json:"human_revision"`
	ApprovedRevision TranscriptRevision `json:"approved_revision"`
}

func (client *Client) StartTranscription(
	ctx context.Context,
	assetID,
	idempotencyKey string,
) (TranscriptionJob, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil || !validIdempotencyKey(idempotencyKey) {
		return TranscriptionJob{}, fmt.Errorf("start transcription: invalid input")
	}
	var result TranscriptionJob
	if err := client.writeJSON(
		ctx, http.MethodPost, "/api/v1/assets/"+assetID+"/transcriptions", nil,
		map[string]string{"Idempotency-Key": strings.TrimSpace(idempotencyKey)},
		http.StatusAccepted, &result, "start transcription",
	); err != nil {
		return TranscriptionJob{}, err
	}
	return result, nil
}

func (client *Client) StartCorrection(
	ctx context.Context,
	revisionID,
	idempotencyKey string,
) (TranscriptionJob, error) {
	revisionID, err := normalizeUUID(revisionID)
	if err != nil || !validIdempotencyKey(idempotencyKey) {
		return TranscriptionJob{}, fmt.Errorf("start correction: invalid input")
	}
	var result TranscriptionJob
	if err := client.writeJSON(
		ctx, http.MethodPost, "/api/v1/transcript-revisions/"+revisionID+"/corrections", nil,
		map[string]string{"Idempotency-Key": strings.TrimSpace(idempotencyKey)},
		http.StatusAccepted, &result, "start correction",
	); err != nil {
		return TranscriptionJob{}, err
	}
	return result, nil
}

func (client *Client) UpdateAssetMetadata(
	ctx context.Context,
	assetID string,
	input UpdateAssetMetadataInput,
) (Asset, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil || input.ExpectedVersion < 1 {
		return Asset{}, fmt.Errorf("update asset metadata: invalid input")
	}
	var collectionID *string
	if input.CollectionID != nil {
		value, normalizeErr := normalizeUUID(*input.CollectionID)
		if normalizeErr != nil {
			return Asset{}, fmt.Errorf("update asset metadata: invalid collection_id")
		}
		collectionID = &value
	}
	payload := struct {
		Title        string  `json:"title"`
		Language     string  `json:"language"`
		CollectionID *string `json:"collection_id"`
	}{Title: input.Title, Language: input.Language, CollectionID: collectionID}
	var result Asset
	if err := client.writeJSON(
		ctx, http.MethodPut, "/api/v1/assets/"+assetID+"/metadata", payload,
		map[string]string{"If-Match": strconv.Quote(strconv.FormatInt(input.ExpectedVersion, 10))},
		http.StatusOK, &result, "update asset metadata",
	); err != nil {
		return Asset{}, err
	}
	return result, nil
}

func (client *Client) AddTags(ctx context.Context, assetID string, tagIDs []string) (TagMutationResult, error) {
	return client.mutateTags(ctx, http.MethodPost, assetID, tagIDs, "add asset tags")
}

func (client *Client) RemoveTags(ctx context.Context, assetID string, tagIDs []string) (TagMutationResult, error) {
	return client.mutateTags(ctx, http.MethodDelete, assetID, tagIDs, "remove asset tags")
}

func (client *Client) mutateTags(
	ctx context.Context,
	method,
	assetID string,
	tagIDs []string,
	operation string,
) (TagMutationResult, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil || len(tagIDs) < 1 || len(tagIDs) > 100 {
		return TagMutationResult{}, fmt.Errorf("%s: invalid input", operation)
	}
	canonicalIDs := make([]string, 0, len(tagIDs))
	seen := make(map[string]struct{}, len(tagIDs))
	for _, tagID := range tagIDs {
		tagID, err = normalizeUUID(tagID)
		if err != nil {
			return TagMutationResult{}, fmt.Errorf("%s: invalid tag_id", operation)
		}
		if _, duplicate := seen[tagID]; duplicate {
			return TagMutationResult{}, fmt.Errorf("%s: duplicate tag_id", operation)
		}
		seen[tagID] = struct{}{}
		canonicalIDs = append(canonicalIDs, tagID)
	}
	payload := struct {
		TagIDs []string `json:"tag_ids"`
	}{TagIDs: canonicalIDs}
	var result TagMutationResult
	if err := client.writeJSON(
		ctx, method, "/api/v1/assets/"+assetID+"/tags", payload, nil,
		http.StatusOK, &result, operation,
	); err != nil {
		return TagMutationResult{}, err
	}
	if result.TagIDs == nil {
		result.TagIDs = make([]string, 0)
	}
	return result, nil
}

func (client *Client) CreateAnnotation(
	ctx context.Context,
	assetID string,
	input CreateAnnotationInput,
) (Annotation, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil || input.StartMS < 0 || (input.EndMS != nil && *input.EndMS <= input.StartMS) {
		return Annotation{}, fmt.Errorf("create annotation: invalid input")
	}
	payload := struct {
		Kind    string `json:"kind"`
		StartMS int64  `json:"start_ms"`
		EndMS   *int64 `json:"end_ms"`
		Body    string `json:"body"`
	}{Kind: input.Kind, StartMS: input.StartMS, EndMS: input.EndMS, Body: input.Body}
	var result Annotation
	if err := client.writeJSON(
		ctx, http.MethodPost, "/api/v1/assets/"+assetID+"/annotations", payload, nil,
		http.StatusCreated, &result, "create annotation",
	); err != nil {
		return Annotation{}, err
	}
	return result, nil
}

func (client *Client) CreateAudioClip(
	ctx context.Context,
	assetID string,
	startMS,
	endMS int64,
) (AudioClip, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil || startMS < 0 || endMS <= startMS || endMS-startMS > 5*60*1000 {
		return AudioClip{}, fmt.Errorf("create audio clip: invalid input")
	}
	payload := struct {
		StartMS int64 `json:"start_ms"`
		EndMS   int64 `json:"end_ms"`
	}{StartMS: startMS, EndMS: endMS}
	var result AudioClip
	if err := client.writeJSON(
		ctx, http.MethodPost, "/api/v1/assets/"+assetID+"/clips", payload, nil,
		http.StatusCreated, &result, "create audio clip",
	); err != nil {
		return AudioClip{}, err
	}
	return result, nil
}

func (client *Client) ExportTranscript(
	ctx context.Context,
	revisionID,
	format string,
) (TranscriptExport, error) {
	revisionID, err := normalizeUUID(revisionID)
	format = strings.ToLower(strings.TrimSpace(format))
	if err != nil || (format != "json" && format != "markdown" && format != "srt" && format != "vtt") {
		return TranscriptExport{}, fmt.Errorf("export transcript: invalid input")
	}
	payload := struct {
		Format string `json:"format"`
	}{Format: format}
	var result TranscriptExport
	if err := client.writeJSON(
		ctx, http.MethodPost, "/api/v1/transcript-revisions/"+revisionID+"/exports", payload, nil,
		http.StatusCreated, &result, "export transcript",
	); err != nil {
		return TranscriptExport{}, err
	}
	return result, nil
}

func (client *Client) ApproveTranscriptRevision(
	ctx context.Context,
	revisionID string,
	acceptPending bool,
) (ApprovalResult, error) {
	revisionID, err := normalizeUUID(revisionID)
	if err != nil {
		return ApprovalResult{}, fmt.Errorf("approve transcript revision: invalid revision_id")
	}
	payload := struct {
		AcceptPending bool `json:"accept_pending"`
	}{AcceptPending: acceptPending}
	var result ApprovalResult
	if err := client.writeJSON(
		ctx, http.MethodPost, "/api/v1/transcript-revisions/"+revisionID+"/approve", payload, nil,
		http.StatusCreated, &result, "approve transcript revision",
	); err != nil {
		return ApprovalResult{}, err
	}
	return result, nil
}

func validIdempotencyKey(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 200 && !containsControl(value)
}
