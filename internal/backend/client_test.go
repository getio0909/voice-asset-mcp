package backend

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGetSystemCapabilities(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/capabilities" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"server_version":"0.2.0","api_version":"v1","contract_version":"0.22.0","features":["mock_asr"]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.GetSystemCapabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ContractVersion != SupportedContractVersion || len(got.Features) != 1 {
		t.Fatalf("unexpected capabilities: %#v", got)
	}
}

func TestGetSystemCapabilitiesRedactsResponseBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "provider-secret", http.StatusBadGateway)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetSystemCapabilities(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "request capabilities: server returned HTTP 502" {
		t.Fatalf("error = %q", got)
	}
}

func TestNewClientRejectsInsecureRemoteURL(t *testing.T) {
	t.Parallel()
	if _, err := NewClient("http://server.example", "token", nil); err == nil {
		t.Fatal("NewClient() accepted insecure remote URL")
	}
}

func TestGetSystemCapabilitiesRejectsIncompatibleResponses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{name: "missing fields", body: `{}`},
		{name: "unsupported API", body: `{"server_version":"0.2.0","api_version":"v2","contract_version":"0.22.0","features":[]}`},
		{name: "unsupported contract", body: `{"server_version":"0.2.0","api_version":"v1","contract_version":"0.1.0","features":[]}`},
		{name: "unsorted features", body: `{"server_version":"0.2.0","api_version":"v1","contract_version":"0.22.0","features":["zeta","alpha"]}`},
		{name: "trailing JSON", body: `{"server_version":"0.2.0","api_version":"v1","contract_version":"0.22.0","features":[]} {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.GetSystemCapabilities(context.Background()); err == nil {
				t.Fatal("GetSystemCapabilities() accepted incompatible response")
			}
		})
	}
}

func TestSupportedContractVersionMatchesRepositoryPin(t *testing.T) {
	t.Parallel()
	pinned, err := os.ReadFile("../../CONTRACT_VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(pinned)); got != SupportedContractVersion {
		t.Fatalf("CONTRACT_VERSION = %q, supported = %q", got, SupportedContractVersion)
	}
}

func TestResourceMethodsUseOnlyPublicRESTEndpoints(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer scoped-token" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/assets":
			if r.URL.Query().Get("q") != "Quarterly" || r.URL.Query().Get("provider_id") != "mock_asr" ||
				r.URL.Query().Get("speaker") != "Alice" || r.URL.Query().Get("limit") != "5" || r.URL.Query().Get("cursor") != "next" {
				t.Errorf("asset query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"30000000-0000-4000-8000-000000000001","workspace_id":"10000000-0000-4000-8000-000000000001","title":"Quarterly","language":"en-US","status":"ready","duration_ms":1200,"version":2,"created_at":"2026-07-16T11:00:00Z","updated_at":"2026-07-16T12:00:00Z"}],"next_cursor":"opaque"}`))
		case "/api/v1/assets/30000000-0000-4000-8000-000000000001":
			_, _ = w.Write([]byte(`{"id":"30000000-0000-4000-8000-000000000001","workspace_id":"10000000-0000-4000-8000-000000000001","title":"Quarterly","language":"en-US","status":"ready","duration_ms":1200,"version":2,"created_at":"2026-07-16T11:00:00Z","updated_at":"2026-07-16T12:00:00Z"}`))
		case "/api/v1/collections":
			if r.URL.Query().Get("limit") != "6" || r.URL.Query().Get("cursor") != "collection-next" {
				t.Errorf("collection query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"70000000-0000-4000-8000-000000000001","workspace_id":"10000000-0000-4000-8000-000000000001","name":"Interviews","description":"Customer interviews","version":1,"asset_count":2,"created_at":"2026-07-16T10:00:00Z","updated_at":"2026-07-16T11:00:00Z"}],"next_cursor":"collection-opaque"}`))
		case "/api/v1/collections/70000000-0000-4000-8000-000000000001":
			_, _ = w.Write([]byte(`{"id":"70000000-0000-4000-8000-000000000001","workspace_id":"10000000-0000-4000-8000-000000000001","name":"Interviews","description":"Customer interviews","version":1,"asset_count":2,"created_at":"2026-07-16T10:00:00Z","updated_at":"2026-07-16T11:00:00Z"}`))
		case "/api/v1/tags":
			if r.URL.Query().Get("limit") != "7" || r.URL.Query().Get("cursor") != "tag-next" {
				t.Errorf("tag query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"71000000-0000-4000-8000-000000000001","workspace_id":"10000000-0000-4000-8000-000000000001","name":"Important","color":"#FF8800","asset_count":1,"created_at":"2026-07-16T10:00:00Z"}],"next_cursor":"tag-opaque"}`))
		case "/api/v1/assets/30000000-0000-4000-8000-000000000001/annotations":
			if r.URL.Query().Get("limit") != "8" || r.URL.Query().Get("cursor") != "annotation-next" {
				t.Errorf("annotation query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"72000000-0000-4000-8000-000000000001","workspace_id":"10000000-0000-4000-8000-000000000001","asset_id":"30000000-0000-4000-8000-000000000001","kind":"bookmark","start_ms":500,"end_ms":1200,"body":"Decision","version":1,"created_by":"20000000-0000-4000-8000-000000000001","created_at":"2026-07-16T10:00:00Z","updated_at":"2026-07-16T10:00:00Z"}],"next_cursor":"annotation-opaque"}`))
		case "/api/v1/assets/30000000-0000-4000-8000-000000000001/processing-status":
			_, _ = w.Write([]byte(`{"asset_id":"30000000-0000-4000-8000-000000000001","asset_status":"processing","active":true,"jobs":[{"id":"73000000-0000-4000-8000-000000000001","kind":"mock_transcribe","state":"queued","attempts":0,"max_attempts":3,"last_error_code":null,"result_revision_id":null,"created_at":"2026-07-16T10:00:00Z","updated_at":"2026-07-16T10:00:00Z"}],"updated_at":"2026-07-16T10:00:00Z"}`))
		case "/api/v1/transcription-jobs/73000000-0000-4000-8000-000000000001":
			_, _ = w.Write([]byte(`{"id":"73000000-0000-4000-8000-000000000001","workspace_id":"10000000-0000-4000-8000-000000000001","asset_id":"30000000-0000-4000-8000-000000000001","created_by":"20000000-0000-4000-8000-000000000001","kind":"mock_transcribe","state":"succeeded","payload":{"asset_id":"30000000-0000-4000-8000-000000000001"},"attempts":1,"max_attempts":3,"available_at":"2026-07-16T10:00:00Z","created_at":"2026-07-16T10:00:00Z","updated_at":"2026-07-16T11:00:00Z"}`))
		case "/api/v1/assets/30000000-0000-4000-8000-000000000001/transcripts":
			_, _ = w.Write([]byte(`{"items":[{"id":"40000000-0000-4000-8000-000000000001","asset_id":"30000000-0000-4000-8000-000000000001","language":"en-US","latest_revision_id":"50000000-0000-4000-8000-000000000001","latest_kind":"normalized","latest_text":"Quarterly","created_at":"2026-07-16T11:00:00Z","revision_created_at":"2026-07-16T12:00:00Z"}]}`))
		case "/api/v1/transcript-revisions/50000000-0000-4000-8000-000000000001":
			_, _ = w.Write([]byte(`{"id":"50000000-0000-4000-8000-000000000001","transcript_id":"40000000-0000-4000-8000-000000000001","asset_id":"30000000-0000-4000-8000-000000000001","kind":"normalized","language":"en-US","text":"Quarterly","provider_snapshot":{},"hotword_snapshot":{},"glossary_snapshot":{},"diff":{},"validation_result":{},"created_by_type":"system","review_status":"pending","created_at":"2026-07-16T12:00:00Z","segments":[{"id":"60000000-0000-4000-8000-000000000001","ordinal":0,"start_ms":0,"end_ms":1200,"speaker":null,"text":"Quarterly","confidence":0.9,"words":[]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "scoped-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	assets, err := client.ListAssets(ctx, ListAssetsInput{
		Query: " Quarterly ", ProviderID: "mock_asr", Speaker: " Alice ", Limit: 5, Cursor: "next",
	})
	if err != nil || len(assets.Items) != 1 || assets.NextCursor == nil {
		t.Fatalf("ListAssets() = (%+v, %v)", assets, err)
	}
	asset, err := client.GetAsset(ctx, "30000000-0000-4000-8000-000000000001")
	if err != nil || asset.DurationMS == nil || *asset.DurationMS != 1200 {
		t.Fatalf("GetAsset() = (%+v, %v)", asset, err)
	}
	collections, err := client.ListCollections(ctx, ListPageInput{Limit: 6, Cursor: "collection-next"})
	if err != nil || len(collections.Items) != 1 || collections.NextCursor == nil || collections.Items[0].AssetCount != 2 {
		t.Fatalf("ListCollections() = (%+v, %v)", collections, err)
	}
	collection, err := client.GetCollection(ctx, collections.Items[0].ID)
	if err != nil || collection.ID != collections.Items[0].ID || collection.AssetCount != 2 {
		t.Fatalf("GetCollection() = (%+v, %v)", collection, err)
	}
	tags, err := client.ListTags(ctx, ListPageInput{Limit: 7, Cursor: "tag-next"})
	if err != nil || len(tags.Items) != 1 || tags.Items[0].Color == nil || *tags.Items[0].Color != "#FF8800" {
		t.Fatalf("ListTags() = (%+v, %v)", tags, err)
	}
	annotations, err := client.ListAnnotations(ctx, asset.ID, ListPageInput{Limit: 8, Cursor: "annotation-next"})
	if err != nil || len(annotations.Items) != 1 || annotations.Items[0].EndMS == nil || *annotations.Items[0].EndMS != 1200 {
		t.Fatalf("ListAnnotations() = (%+v, %v)", annotations, err)
	}
	status, err := client.GetProcessingStatus(ctx, asset.ID)
	if err != nil || !status.Active || len(status.Jobs) != 1 || status.Jobs[0].State != "queued" {
		t.Fatalf("GetProcessingStatus() = (%+v, %v)", status, err)
	}
	job, err := client.GetTranscriptionJob(ctx, "73000000-0000-4000-8000-000000000001")
	if err != nil || job.State != "succeeded" || len(job.Payload) == 0 {
		t.Fatalf("GetTranscriptionJob() = (%+v, %v)", job, err)
	}
	transcripts, err := client.ListTranscripts(ctx, asset.ID)
	if err != nil || len(transcripts.Items) != 1 {
		t.Fatalf("ListTranscripts() = (%+v, %v)", transcripts, err)
	}
	revision, err := client.GetTranscriptRevision(ctx, transcripts.Items[0].LatestRevisionID)
	if err != nil || len(revision.Segments) != 1 || revision.Segments[0].EndMS != 1200 {
		t.Fatalf("GetTranscriptRevision() = (%+v, %v)", revision, err)
	}
}

func TestListAssetsRejectsInvalidSearchFiltersBeforeNetwork(t *testing.T) {
	t.Parallel()
	client, err := NewClient("http://127.0.0.1:1", "", &http.Client{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	for name, input := range map[string]ListAssetsInput{
		"provider": {ProviderID: "unknown_asr"},
		"speaker":  {Speaker: "bad\nspeaker"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, listErr := client.ListAssets(context.Background(), input); listErr == nil {
				t.Fatalf("ListAssets(%s) error = nil", name)
			}
		})
	}
}

func TestResourceErrorsRedactServerBodyAndPreserveCancellation(t *testing.T) {
	t.Parallel()
	t.Run("scope denial", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "token-and-provider-secret", http.StatusForbidden)
		}))
		defer server.Close()
		client, err := NewClient(server.URL, "token", server.Client())
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.ListAssets(context.Background(), ListAssetsInput{})
		if err == nil || err.Error() != "list assets: server returned HTTP 403" {
			t.Fatalf("ListAssets() error = %v", err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()
		client, err := NewClient(server.URL, "", &http.Client{Timeout: 5 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = client.ListAssets(ctx, ListAssetsInput{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ListAssets() error = %v, want context.Canceled", err)
		}
	})
}

func TestOrganizationMethodsRejectUnboundedInputsBeforeNetwork(t *testing.T) {
	t.Parallel()
	client, err := NewClient("http://127.0.0.1:1", "", &http.Client{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		call func() error
	}{
		{name: "collection limit", call: func() error {
			_, err := client.ListCollections(context.Background(), ListPageInput{Limit: 101})
			return err
		}},
		{name: "tag cursor", call: func() error {
			_, err := client.ListTags(context.Background(), ListPageInput{Cursor: " padded "})
			return err
		}},
		{name: "annotation asset", call: func() error {
			_, err := client.ListAnnotations(context.Background(), "not-a-uuid", ListPageInput{})
			return err
		}},
		{name: "processing asset", call: func() error {
			_, err := client.GetProcessingStatus(context.Background(), "not-a-uuid")
			return err
		}},
		{name: "collection", call: func() error {
			_, err := client.GetCollection(context.Background(), "not-a-uuid")
			return err
		}},
		{name: "job", call: func() error {
			_, err := client.GetTranscriptionJob(context.Background(), "not-a-uuid")
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); err == nil {
				t.Fatal("invalid input was accepted")
			}
		})
	}
}
