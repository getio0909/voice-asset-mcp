package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteMethodsUsePublicRESTWithRequiredHeadersAndBodies(t *testing.T) {
	t.Parallel()
	const (
		assetID      = "30000000-0000-4000-8000-000000000001"
		revisionID   = "50000000-0000-4000-8000-000000000001"
		collectionID = "70000000-0000-4000-8000-000000000001"
		tagID        = "71000000-0000-4000-8000-000000000001"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer scoped-write-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/assets/"+assetID+"/transcriptions":
			if r.Header.Get("Idempotency-Key") != "transcribe-1" {
				t.Errorf("transcription idempotency key = %q", r.Header.Get("Idempotency-Key"))
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"id":"73000000-0000-4000-8000-000000000001","asset_id":"` + assetID + `","state":"queued"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/transcript-revisions/"+revisionID+"/corrections":
			if r.Header.Get("Idempotency-Key") != "correct-1" {
				t.Errorf("correction idempotency key = %q", r.Header.Get("Idempotency-Key"))
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"id":"73000000-0000-4000-8000-000000000002","state":"queued"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/assets/"+assetID+"/metadata":
			if r.Header.Get("If-Match") != `"2"` {
				t.Errorf("If-Match = %q", r.Header.Get("If-Match"))
			}
			body := decodeRequestMap(t, r)
			if body["collection_id"] != collectionID || body["title"] != "Updated" {
				t.Errorf("metadata body = %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":"` + assetID + `","collection_id":"` + collectionID + `","title":"Updated","language":"en-US","version":3}`))
		case (r.Method == http.MethodPost || r.Method == http.MethodDelete) && r.URL.Path == "/api/v1/assets/"+assetID+"/tags":
			body := decodeRequestMap(t, r)
			if got, ok := body["tag_ids"].([]any); !ok || len(got) != 1 || got[0] != tagID {
				t.Errorf("tag body = %#v", body)
			}
			_, _ = w.Write([]byte(`{"asset_id":"` + assetID + `","tag_ids":["` + tagID + `"],"changed_count":1}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/assets/"+assetID+"/annotations":
			body := decodeRequestMap(t, r)
			if body["start_ms"] != float64(250) || body["kind"] != "note" {
				t.Errorf("annotation body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"72000000-0000-4000-8000-000000000001","asset_id":"` + assetID + `","kind":"note","start_ms":250,"body":"Decision"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/assets/"+assetID+"/clips":
			body := decodeRequestMap(t, r)
			if body["start_ms"] != float64(500) || body["end_ms"] != float64(2000) {
				t.Errorf("clip body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"76000000-0000-4000-8000-000000000001","asset_id":"` + assetID + `","start_ms":500,"end_ms":2000,"duration_ms":1500,"mime_type":"audio/wav","download_url":"/api/v1/audio-clips/76000000-0000-4000-8000-000000000001"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/transcript-revisions/"+revisionID+"/exports":
			body := decodeRequestMap(t, r)
			if body["format"] != "vtt" {
				t.Errorf("export body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"77000000-0000-4000-8000-000000000001","asset_id":"` + assetID + `","revision_id":"` + revisionID + `","format":"vtt","mime_type":"text/vtt; charset=utf-8","download_url":"/api/v1/transcript-exports/77000000-0000-4000-8000-000000000001"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/transcript-revisions/"+revisionID+"/approve":
			body := decodeRequestMap(t, r)
			if body["accept_pending"] != true {
				t.Errorf("approval body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"review":{"id":"74000000-0000-4000-8000-000000000001"},"human_revision":{"id":"` + revisionID + `"},"approved_revision":{"id":"75000000-0000-4000-8000-000000000001"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "scoped-write-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if job, err := client.StartTranscription(ctx, assetID, "transcribe-1"); err != nil || job.State != "queued" {
		t.Fatalf("StartTranscription() = (%+v, %v)", job, err)
	}
	if job, err := client.StartCorrection(ctx, revisionID, "correct-1"); err != nil || job.State != "queued" {
		t.Fatalf("StartCorrection() = (%+v, %v)", job, err)
	}
	collection := collectionID
	if updated, err := client.UpdateAssetMetadata(ctx, assetID, UpdateAssetMetadataInput{
		Title: "Updated", Language: "en-US", CollectionID: &collection, ExpectedVersion: 2,
	}); err != nil || updated.Version != 3 || updated.CollectionID == nil {
		t.Fatalf("UpdateAssetMetadata() = (%+v, %v)", updated, err)
	}
	if result, err := client.AddTags(ctx, assetID, []string{tagID}); err != nil || result.ChangedCount != 1 {
		t.Fatalf("AddTags() = (%+v, %v)", result, err)
	}
	if result, err := client.RemoveTags(ctx, assetID, []string{tagID}); err != nil || result.ChangedCount != 1 {
		t.Fatalf("RemoveTags() = (%+v, %v)", result, err)
	}
	if annotation, err := client.CreateAnnotation(ctx, assetID, CreateAnnotationInput{
		Kind: "note", StartMS: 250, Body: "Decision",
	}); err != nil || annotation.Kind != "note" {
		t.Fatalf("CreateAnnotation() = (%+v, %v)", annotation, err)
	}
	if audioClip, err := client.CreateAudioClip(ctx, assetID, 500, 2_000); err != nil ||
		audioClip.DurationMS != 1_500 || !strings.HasPrefix(audioClip.DownloadURL, "/api/v1/audio-clips/") {
		t.Fatalf("CreateAudioClip() = (%+v, %v)", audioClip, err)
	}
	if exported, err := client.ExportTranscript(ctx, revisionID, " VTT "); err != nil ||
		exported.Format != "vtt" || !strings.HasPrefix(exported.DownloadURL, "/api/v1/transcript-exports/") {
		t.Fatalf("ExportTranscript() = (%+v, %v)", exported, err)
	}
	if approval, err := client.ApproveTranscriptRevision(ctx, revisionID, true); err != nil || approval.ApprovedRevision.ID == "" {
		t.Fatalf("ApproveTranscriptRevision() = (%+v, %v)", approval, err)
	}
}

func TestWriteMethodsRedactErrorBodies(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "provider-secret", http.StatusForbidden)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.AddTags(context.Background(),
		"30000000-0000-4000-8000-000000000001",
		[]string{"71000000-0000-4000-8000-000000000001"},
	)
	if err == nil || strings.Contains(err.Error(), "provider-secret") || err.Error() != "add asset tags: server returned HTTP 403" {
		t.Fatalf("error = %v", err)
	}
}

func decodeRequestMap(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return result
}
