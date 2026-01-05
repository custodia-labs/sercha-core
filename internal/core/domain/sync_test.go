package domain

import (
	"testing"
	"time"
)

func TestSyncStatusConstants(t *testing.T) {
	if SyncStatusIdle != "idle" {
		t.Errorf("expected SyncStatusIdle = 'idle', got %s", SyncStatusIdle)
	}
	if SyncStatusRunning != "running" {
		t.Errorf("expected SyncStatusRunning = 'running', got %s", SyncStatusRunning)
	}
	if SyncStatusCompleted != "completed" {
		t.Errorf("expected SyncStatusCompleted = 'completed', got %s", SyncStatusCompleted)
	}
	if SyncStatusFailed != "failed" {
		t.Errorf("expected SyncStatusFailed = 'failed', got %s", SyncStatusFailed)
	}
}

func TestSyncState(t *testing.T) {
	now := time.Now()
	nextSync := now.Add(1 * time.Hour)

	state := &SyncState{
		SourceID:   "source-123",
		Status:     SyncStatusCompleted,
		LastSyncAt: &now,
		NextSyncAt: &nextSync,
		Cursor:     "cursor-abc123",
		Stats: SyncStats{
			DocumentsAdded:   10,
			DocumentsUpdated: 5,
			DocumentsDeleted: 2,
			ChunksIndexed:    50,
			Errors:           0,
		},
		Error:       "",
		StartedAt:   &now,
		CompletedAt: &now,
	}

	if state.SourceID != "source-123" {
		t.Errorf("expected SourceID source-123, got %s", state.SourceID)
	}
	if state.Status != SyncStatusCompleted {
		t.Errorf("expected Status completed, got %s", state.Status)
	}
	if state.Cursor != "cursor-abc123" {
		t.Errorf("expected Cursor cursor-abc123, got %s", state.Cursor)
	}
	if state.Stats.DocumentsAdded != 10 {
		t.Errorf("expected DocumentsAdded 10, got %d", state.Stats.DocumentsAdded)
	}
	if state.Stats.ChunksIndexed != 50 {
		t.Errorf("expected ChunksIndexed 50, got %d", state.Stats.ChunksIndexed)
	}
}

func TestSyncStats(t *testing.T) {
	stats := SyncStats{
		DocumentsAdded:   100,
		DocumentsUpdated: 50,
		DocumentsDeleted: 10,
		ChunksIndexed:    500,
		Errors:           3,
	}

	if stats.DocumentsAdded != 100 {
		t.Errorf("expected DocumentsAdded 100, got %d", stats.DocumentsAdded)
	}
	if stats.DocumentsUpdated != 50 {
		t.Errorf("expected DocumentsUpdated 50, got %d", stats.DocumentsUpdated)
	}
	if stats.DocumentsDeleted != 10 {
		t.Errorf("expected DocumentsDeleted 10, got %d", stats.DocumentsDeleted)
	}
	if stats.ChunksIndexed != 500 {
		t.Errorf("expected ChunksIndexed 500, got %d", stats.ChunksIndexed)
	}
	if stats.Errors != 3 {
		t.Errorf("expected Errors 3, got %d", stats.Errors)
	}
}

func TestChangeTypeConstants(t *testing.T) {
	if ChangeTypeAdded != "added" {
		t.Errorf("expected ChangeTypeAdded = 'added', got %s", ChangeTypeAdded)
	}
	if ChangeTypeModified != "modified" {
		t.Errorf("expected ChangeTypeModified = 'modified', got %s", ChangeTypeModified)
	}
	if ChangeTypeDeleted != "deleted" {
		t.Errorf("expected ChangeTypeDeleted = 'deleted', got %s", ChangeTypeDeleted)
	}
}

func TestChange(t *testing.T) {
	doc := &Document{
		ID:    "doc-123",
		Title: "New Document",
	}

	// Added change
	addedChange := &Change{
		Type:       ChangeTypeAdded,
		Document:   doc,
		Content:    "Document content here",
		ExternalID: "ext-123",
	}

	if addedChange.Type != ChangeTypeAdded {
		t.Errorf("expected Type added, got %s", addedChange.Type)
	}
	if addedChange.Document == nil {
		t.Error("expected Document to be set for added change")
	}
	if addedChange.Content != "Document content here" {
		t.Errorf("expected Content 'Document content here', got %s", addedChange.Content)
	}

	// Deleted change
	deletedChange := &Change{
		Type:       ChangeTypeDeleted,
		DeletedID:  "doc-456",
		ExternalID: "ext-456",
	}

	if deletedChange.Type != ChangeTypeDeleted {
		t.Errorf("expected Type deleted, got %s", deletedChange.Type)
	}
	if deletedChange.DeletedID != "doc-456" {
		t.Errorf("expected DeletedID doc-456, got %s", deletedChange.DeletedID)
	}
}

func TestSyncResult(t *testing.T) {
	result := &SyncResult{
		SourceID: "source-123",
		Success:  true,
		Stats: SyncStats{
			DocumentsAdded:   10,
			DocumentsUpdated: 5,
			DocumentsDeleted: 2,
			ChunksIndexed:    50,
		},
		Error:    "",
		Duration: 5.5,
		Cursor:   "new-cursor",
	}

	if result.SourceID != "source-123" {
		t.Errorf("expected SourceID source-123, got %s", result.SourceID)
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Duration != 5.5 {
		t.Errorf("expected Duration 5.5, got %f", result.Duration)
	}
	if result.Cursor != "new-cursor" {
		t.Errorf("expected Cursor new-cursor, got %s", result.Cursor)
	}

	// Failed result
	failedResult := &SyncResult{
		SourceID: "source-456",
		Success:  false,
		Error:    "connection timeout",
		Duration: 30.0,
	}

	if failedResult.Success {
		t.Error("expected Success to be false")
	}
	if failedResult.Error != "connection timeout" {
		t.Errorf("expected Error 'connection timeout', got %s", failedResult.Error)
	}
}
