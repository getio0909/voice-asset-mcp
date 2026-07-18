package backend

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ListPageInput struct {
	Limit  int
	Cursor string
}

type Collection struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     int64     `json:"version"`
	AssetCount  int64     `json:"asset_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CollectionList struct {
	Items      []Collection `json:"items"`
	NextCursor *string      `json:"next_cursor,omitempty"`
}

type Tag struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Color       *string   `json:"color"`
	AssetCount  int64     `json:"asset_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type TagList struct {
	Items      []Tag   `json:"items"`
	NextCursor *string `json:"next_cursor,omitempty"`
}

type Annotation struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	AssetID     string    `json:"asset_id"`
	Kind        string    `json:"kind"`
	StartMS     int64     `json:"start_ms"`
	EndMS       *int64    `json:"end_ms"`
	Body        string    `json:"body"`
	Version     int64     `json:"version"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AnnotationList struct {
	Items      []Annotation `json:"items"`
	NextCursor *string      `json:"next_cursor,omitempty"`
}

type ProcessingJob struct {
	ID               string    `json:"id"`
	Kind             string    `json:"kind"`
	State            string    `json:"state"`
	Attempts         int       `json:"attempts"`
	MaxAttempts      int       `json:"max_attempts"`
	LastErrorCode    *string   `json:"last_error_code"`
	ResultRevisionID *string   `json:"result_revision_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ProcessingStatus struct {
	AssetID     string          `json:"asset_id"`
	AssetStatus string          `json:"asset_status"`
	Active      bool            `json:"active"`
	Jobs        []ProcessingJob `json:"jobs"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (client *Client) GetCollection(ctx context.Context, collectionID string) (Collection, error) {
	collectionID, err := normalizeUUID(collectionID)
	if err != nil {
		return Collection{}, fmt.Errorf("get collection: invalid collection_id")
	}
	var result Collection
	if err := client.getJSON(ctx, "/api/v1/collections/"+collectionID, nil, &result, "get collection"); err != nil {
		return Collection{}, err
	}
	return result, nil
}

func (client *Client) ListCollections(ctx context.Context, input ListPageInput) (CollectionList, error) {
	values, err := listPageValues("list collections", input)
	if err != nil {
		return CollectionList{}, err
	}
	var result CollectionList
	if err := client.getJSON(ctx, "/api/v1/collections", values, &result, "list collections"); err != nil {
		return CollectionList{}, err
	}
	if result.Items == nil {
		result.Items = make([]Collection, 0)
	}
	return result, nil
}

func (client *Client) ListTags(ctx context.Context, input ListPageInput) (TagList, error) {
	values, err := listPageValues("list tags", input)
	if err != nil {
		return TagList{}, err
	}
	var result TagList
	if err := client.getJSON(ctx, "/api/v1/tags", values, &result, "list tags"); err != nil {
		return TagList{}, err
	}
	if result.Items == nil {
		result.Items = make([]Tag, 0)
	}
	return result, nil
}

func (client *Client) ListAnnotations(
	ctx context.Context,
	assetID string,
	input ListPageInput,
) (AnnotationList, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil {
		return AnnotationList{}, fmt.Errorf("list annotations: invalid asset_id")
	}
	values, err := listPageValues("list annotations", input)
	if err != nil {
		return AnnotationList{}, err
	}
	var result AnnotationList
	if err := client.getJSON(ctx, "/api/v1/assets/"+assetID+"/annotations", values, &result, "list annotations"); err != nil {
		return AnnotationList{}, err
	}
	if result.Items == nil {
		result.Items = make([]Annotation, 0)
	}
	return result, nil
}

func (client *Client) GetProcessingStatus(ctx context.Context, assetID string) (ProcessingStatus, error) {
	assetID, err := normalizeUUID(assetID)
	if err != nil {
		return ProcessingStatus{}, fmt.Errorf("get processing status: invalid asset_id")
	}
	var result ProcessingStatus
	if err := client.getJSON(
		ctx, "/api/v1/assets/"+assetID+"/processing-status", nil, &result, "get processing status",
	); err != nil {
		return ProcessingStatus{}, err
	}
	if result.Jobs == nil {
		result.Jobs = make([]ProcessingJob, 0)
	}
	return result, nil
}

func listPageValues(operation string, input ListPageInput) (url.Values, error) {
	if input.Limit < 0 || input.Limit > maxListLimit {
		return nil, fmt.Errorf("%s: limit must be between 1 and %d", operation, maxListLimit)
	}
	if len(input.Cursor) > maxCursorLength || strings.TrimSpace(input.Cursor) != input.Cursor {
		return nil, fmt.Errorf("%s: invalid cursor", operation)
	}
	values := make(url.Values)
	if input.Limit > 0 {
		values.Set("limit", strconv.Itoa(input.Limit))
	}
	if input.Cursor != "" {
		values.Set("cursor", input.Cursor)
	}
	return values, nil
}
