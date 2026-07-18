package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxListLimit    = 100
	maxQueryRunes   = 200
	maxCursorLength = 1024
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Asset struct {
	ID           string            `json:"id"`
	WorkspaceID  string            `json:"workspace_id"`
	CollectionID *string           `json:"collection_id"`
	Title        string            `json:"title"`
	Language     string            `json:"language"`
	Status       string            `json:"status"`
	DurationMS   *int64            `json:"duration_ms"`
	Version      int64             `json:"version"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Search       *AssetSearchMatch `json:"search,omitempty"`
}

type AssetSearchMatch struct {
	Title       bool                    `json:"title"`
	ProviderIDs []string                `json:"provider_ids"`
	Segments    []AssetSearchSegmentHit `json:"segments"`
}

type AssetSearchSegmentHit struct {
	TranscriptID string  `json:"transcript_id"`
	RevisionID   string  `json:"revision_id"`
	SegmentID    string  `json:"segment_id"`
	Ordinal      int     `json:"ordinal"`
	StartMS      int64   `json:"start_ms"`
	EndMS        int64   `json:"end_ms"`
	Speaker      *string `json:"speaker"`
	Text         string  `json:"text"`
}

type AssetList struct {
	Items      []Asset `json:"items"`
	NextCursor *string `json:"next_cursor,omitempty"`
}

type ListAssetsInput struct {
	Query      string
	ProviderID string
	Speaker    string
	Limit      int
	Cursor     string
}

type TranscriptList struct {
	Items []TranscriptSummary `json:"items"`
}

type TranscriptSummary struct {
	ID                string    `json:"id"`
	AssetID           string    `json:"asset_id"`
	Language          string    `json:"language"`
	LatestRevisionID  string    `json:"latest_revision_id"`
	LatestKind        string    `json:"latest_kind"`
	LatestText        string    `json:"latest_text"`
	CreatedAt         time.Time `json:"created_at"`
	RevisionCreatedAt time.Time `json:"revision_created_at"`
}

type TranscriptRevision struct {
	ID               string              `json:"id"`
	TranscriptID     string              `json:"transcript_id"`
	AssetID          string              `json:"asset_id"`
	ParentRevisionID string              `json:"parent_revision_id,omitempty"`
	Kind             string              `json:"kind"`
	Language         string              `json:"language"`
	Text             string              `json:"text"`
	CreatedByType    string              `json:"created_by_type"`
	ReviewStatus     string              `json:"review_status"`
	CreatedAt        time.Time           `json:"created_at"`
	Segments         []TranscriptSegment `json:"segments"`
}

type TranscriptSegment struct {
	ID         string   `json:"id"`
	Ordinal    int      `json:"ordinal"`
	StartMS    int64    `json:"start_ms"`
	EndMS      int64    `json:"end_ms"`
	Speaker    *string  `json:"speaker"`
	Text       string   `json:"text"`
	Confidence *float64 `json:"confidence"`
}

type TranscriptionJob struct {
	ID               string          `json:"id"`
	WorkspaceID      string          `json:"workspace_id"`
	AssetID          string          `json:"asset_id"`
	CreatedBy        string          `json:"created_by"`
	Kind             string          `json:"kind"`
	State            string          `json:"state"`
	Payload          json.RawMessage `json:"payload"`
	Attempts         int             `json:"attempts"`
	MaxAttempts      int             `json:"max_attempts"`
	AvailableAt      time.Time       `json:"available_at"`
	LeaseOwner       *string         `json:"lease_owner,omitempty"`
	LeaseExpiresAt   *time.Time      `json:"lease_expires_at,omitempty"`
	LastErrorCode    *string         `json:"last_error_code,omitempty"`
	ResultRevisionID *string         `json:"result_revision_id,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func (c *Client) ListAssets(ctx context.Context, input ListAssetsInput) (AssetList, error) {
	query := strings.TrimSpace(input.Query)
	if utf8.RuneCountInString(query) > maxQueryRunes || containsControl(query) {
		return AssetList{}, fmt.Errorf("list assets: invalid query")
	}
	if input.Limit < 0 || input.Limit > maxListLimit {
		return AssetList{}, fmt.Errorf("list assets: limit must be between 1 and %d", maxListLimit)
	}
	providerID := strings.TrimSpace(input.ProviderID)
	if providerID != "" && providerID != "mock_asr" && providerID != "aliyun_asr" && providerID != "tencent_asr" {
		return AssetList{}, fmt.Errorf("list assets: invalid provider_id")
	}
	speaker := strings.TrimSpace(input.Speaker)
	if utf8.RuneCountInString(speaker) > maxQueryRunes || containsControl(speaker) {
		return AssetList{}, fmt.Errorf("list assets: invalid speaker")
	}
	if len(input.Cursor) > maxCursorLength || strings.TrimSpace(input.Cursor) != input.Cursor {
		return AssetList{}, fmt.Errorf("list assets: invalid cursor")
	}
	values := make(url.Values)
	if query != "" {
		values.Set("q", query)
	}
	if providerID != "" {
		values.Set("provider_id", providerID)
	}
	if speaker != "" {
		values.Set("speaker", speaker)
	}
	if input.Limit > 0 {
		values.Set("limit", strconv.Itoa(input.Limit))
	}
	if input.Cursor != "" {
		values.Set("cursor", input.Cursor)
	}
	var result AssetList
	if err := c.getJSON(ctx, "/api/v1/assets", values, &result, "list assets"); err != nil {
		return AssetList{}, err
	}
	if result.Items == nil {
		result.Items = make([]Asset, 0)
	}
	return result, nil
}

func (c *Client) GetAsset(ctx context.Context, assetID string) (Asset, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil {
		return Asset{}, fmt.Errorf("get asset: invalid asset_id")
	}
	var result Asset
	if err := c.getJSON(ctx, "/api/v1/assets/"+assetID, nil, &result, "get asset"); err != nil {
		return Asset{}, err
	}
	return result, nil
}

func (c *Client) ListTranscripts(ctx context.Context, assetID string) (TranscriptList, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil {
		return TranscriptList{}, fmt.Errorf("list transcripts: invalid asset_id")
	}
	var result TranscriptList
	if err := c.getJSON(ctx, "/api/v1/assets/"+assetID+"/transcripts", nil, &result, "list transcripts"); err != nil {
		return TranscriptList{}, err
	}
	if result.Items == nil {
		result.Items = make([]TranscriptSummary, 0)
	}
	return result, nil
}

func (c *Client) GetTranscriptRevision(ctx context.Context, revisionID string) (TranscriptRevision, error) {
	revisionID, err := normalizeUUID(revisionID)
	if err != nil {
		return TranscriptRevision{}, fmt.Errorf("get transcript revision: invalid revision_id")
	}
	var result TranscriptRevision
	if err := c.getJSON(ctx, "/api/v1/transcript-revisions/"+revisionID, nil, &result, "get transcript revision"); err != nil {
		return TranscriptRevision{}, err
	}
	if result.Segments == nil {
		result.Segments = make([]TranscriptSegment, 0)
	}
	return result, nil
}

func (c *Client) GetTranscriptionJob(ctx context.Context, jobID string) (TranscriptionJob, error) {
	jobID, err := normalizeUUID(jobID)
	if err != nil {
		return TranscriptionJob{}, fmt.Errorf("get transcription job: invalid job_id")
	}
	var result TranscriptionJob
	if err := c.getJSON(ctx, "/api/v1/transcription-jobs/"+jobID, nil, &result, "get transcription job"); err != nil {
		return TranscriptionJob{}, err
	}
	return result, nil
}

func normalizeUUID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !uuidPattern.MatchString(value) {
		return "", fmt.Errorf("invalid UUID")
	}
	return strings.ToLower(value), nil
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}
