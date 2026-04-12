package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestRepositoryCreateSpikeEvent(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := NewDatabase(dbPath, 7)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	// Create a test spike event
	event := &SpikeEvent{
		ID:                   "test-namespace/test-pod-123",
		Timestamp:            time.Now(),
		PodName:              "test-pod-123",
		Namespace:            "test-namespace",
		CPUUsagePercent:      85.5,
		CPULimitPercent:      100.0,
		RAMUsagePercent:      70.0,
		RAMLimitPercent:      100.0,
		ThresholdPercent:     50.0,
		MovingAveragePercent: 50.0,
		AlertSent:            false,
		CreatedAt:            time.Now(),
	}

	ctx := context.Background()
	err = repo.CreateSpikeEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to create spike event: %v", err)
	}

	// Verify it was created
	retrieved, err := repo.GetSpikeEvent(ctx, event.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve spike event: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved event is nil")
	}

	if retrieved.ID != event.ID {
		t.Errorf("ID mismatch: got %v, want %v", retrieved.ID, event.ID)
	}

	if retrieved.PodName != event.PodName {
		t.Errorf("PodName mismatch: got %v, want %v", retrieved.PodName, event.PodName)
	}

	if retrieved.Namespace != event.Namespace {
		t.Errorf("Namespace mismatch: got %v, want %v", retrieved.Namespace, event.Namespace)
	}
}

func TestRepositoryGetSpikeEventNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := NewDatabase(dbPath, 7)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	ctx := context.Background()
	event, err := repo.GetSpikeEvent(ctx, "nonexistent-id")
	if err != nil {
		t.Fatalf("Failed to get spike event: %v", err)
	}

	if event != nil {
		t.Errorf("Expected nil event for nonexistent ID, got %v", event)
	}
}

func TestRepositoryListSpikeEvents(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := NewDatabase(dbPath, 7)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	ctx := context.Background()

	// Create multiple test events
	for i := 0; i < 5; i++ {
		event := &SpikeEvent{
			ID:                   "default/test-pod-" + string(rune('0'+i)),
			Timestamp:            time.Now().Add(time.Duration(-i) * time.Hour),
			PodName:              "test-pod-" + string(rune('0'+i)),
			Namespace:            "default",
			CPUUsagePercent:      80.0 + float64(i)*10,
			RAMUsagePercent:      60.0,
			ThresholdPercent:     50.0,
			MovingAveragePercent: 50.0,
			CreatedAt:            time.Now(),
		}

		err := repo.CreateSpikeEvent(ctx, event)
		if err != nil {
			t.Fatalf("Failed to create spike event: %v", err)
		}
	}

	// List all events
	events, err := repo.ListSpikeEvents(ctx, 10, 0, "", "", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("Failed to list spike events: %v", err)
	}

	if len(events) != 5 {
		t.Errorf("Expected 5 events, got %d", len(events))
	}
}

func TestRepositoryListSpikeEventsFiltered(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := NewDatabase(dbPath, 7)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	ctx := context.Background()

	// Create events in different namespaces
	events := []*SpikeEvent{
		{
			ID: "default/pod-1", PodName: "pod-1", Namespace: "default",
			CPUUsagePercent: 80.0, ThresholdPercent: 50.0, MovingAveragePercent: 50.0, CreatedAt: time.Now(),
		},
		{
			ID: "production/pod-2", PodName: "pod-2", Namespace: "production",
			CPUUsagePercent: 85.0, ThresholdPercent: 50.0, MovingAveragePercent: 50.0, CreatedAt: time.Now(),
		},
		{
			ID: "production/pod-3", PodName: "pod-3", Namespace: "production",
			CPUUsagePercent: 90.0, ThresholdPercent: 50.0, MovingAveragePercent: 50.0, CreatedAt: time.Now(),
		},
	}

	for _, e := range events {
		if err := repo.CreateSpikeEvent(ctx, e); err != nil {
			t.Fatalf("Failed to create spike event: %v", err)
		}
	}

	// Filter by namespace
	productionEvents, err := repo.ListSpikeEvents(ctx, 10, 0, "production", "", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("Failed to list spike events: %v", err)
	}

	if len(productionEvents) != 2 {
		t.Errorf("Expected 2 production events, got %d", len(productionEvents))
	}
}

func TestRepositoryUpdateSpikeEvent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := NewDatabase(dbPath, 7)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	ctx := context.Background()

	// Create an event
	event := &SpikeEvent{
		ID:                   "default/test-pod",
		Timestamp:            time.Now(),
		PodName:              "test-pod",
		Namespace:            "default",
		CPUUsagePercent:      80.0,
		ThresholdPercent:     50.0,
		MovingAveragePercent: 50.0,
		CreatedAt:            time.Now(),
	}

	err = repo.CreateSpikeEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to create spike event: %v", err)
	}

	// Update it
	routeName := "/api/users"
	event.RouteName = &routeName
	traceID := "abc-123"
	event.TraceID = &traceID

	err = repo.UpdateSpikeEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to update spike event: %v", err)
	}

	// Verify update
	updated, err := repo.GetSpikeEvent(ctx, event.ID)
	if err != nil {
		t.Fatalf("Failed to get spike event: %v", err)
	}

	if updated.RouteName == nil || *updated.RouteName != routeName {
		t.Errorf("RouteName not updated: got %v, want %v", updated.RouteName, routeName)
	}

	if updated.TraceID == nil || *updated.TraceID != traceID {
		t.Errorf("TraceID not updated: got %v, want %v", updated.TraceID, traceID)
	}
}

func TestRepositoryDeleteSpikeEvent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := NewDatabase(dbPath, 7)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	ctx := context.Background()

	// Create an event
	event := &SpikeEvent{
		ID:                   "default/test-pod-delete",
		Timestamp:            time.Now(),
		PodName:              "test-pod-delete",
		Namespace:            "default",
		CPUUsagePercent:      80.0,
		ThresholdPercent:     50.0,
		MovingAveragePercent: 50.0,
		CreatedAt:            time.Now(),
	}

	err = repo.CreateSpikeEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to create spike event: %v", err)
	}

	// Delete it
	err = repo.DeleteSpikeEvent(ctx, event.ID)
	if err != nil {
		t.Fatalf("Failed to delete spike event: %v", err)
	}

	// Verify deletion
	deleted, err := repo.GetSpikeEvent(ctx, event.ID)
	if err != nil {
		t.Fatalf("Failed to get spike event: %v", err)
	}

	if deleted != nil {
		t.Error("Event was not deleted")
	}
}

func TestNullStringHelpers(t *testing.T) {
	// Test nullString with nil pointer
	var nilPtr *string
	result := nullString(nilPtr)
	if result.Valid {
		t.Error("nullString should return invalid for nil pointer")
	}

	str := "test"
	result = nullString(&str)
	if !result.Valid || result.String != str {
		t.Errorf("nullString incorrect: got %v", result)
	}

	// Test nullStringPtr
	ptr := nullStringPtr(result)
	if ptr == nil || *ptr != str {
		t.Errorf("nullStringPtr incorrect: got %v", ptr)
	}

	// Test with invalid NullString
	invalid := sql.NullString{}
	ptr = nullStringPtr(invalid)
	if ptr != nil {
		t.Errorf("nullStringPtr should return nil for invalid: got %v", ptr)
	}
}

func TestNullTimeHelpers(t *testing.T) {
	// Test nullTime with nil
	var nilTime *time.Time
	result := nullTime(nilTime)
	if result.Valid {
		t.Error("nullTime should return invalid for nil pointer")
	}

	now := time.Now()
	result = nullTime(&now)
	if !result.Valid {
		t.Errorf("nullTime incorrect: got %v", result)
	}

	// Test nullTimePtr
	ptr := nullTimePtr(result)
	if ptr == nil {
		t.Error("nullTimePtr should not return nil for valid time")
	}

	// Test with invalid NullString
	invalid := sql.NullString{}
	ptr = nullTimePtr(invalid)
	if ptr != nil {
		t.Errorf("nullTimePtr should return nil for invalid: got %v", ptr)
	}
}
