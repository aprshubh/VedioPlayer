package video

import (
	"math"
	"testing"
	"time"
	"watchparty-backend/internal/model"
	"watchparty-backend/internal/store"
)

func TestVideoSyncerDrift(t *testing.T) {
	memStore := store.NewMemoryStore()
	syncer := NewVideoSyncer(memStore)

	room := &model.Room{
		ID:                 "room1",
		Code:               "TEST01",
		HostID:             "user1",
		OnlyHostCanControl: false,
		CreatedAt:          time.Now(),
	}
	if err := memStore.CreateRoom(room); err != nil {
		t.Fatalf("failed to create room: %v", err)
	}

	// Host plays video at position 10.0s
	state, err := syncer.HandleAction(model.EventPlay, "room1", "user1", true, 10.0, 1.0)
	if err != nil {
		t.Fatalf("failed to handle play: %v", err)
	}
	if !state.Playing || state.Position != 10.0 {
		t.Fatalf("unexpected state: %+v", state)
	}

	// Check drift for a client that is at 9.8s (200ms drift)
	correction, err := syncer.CheckSync("room1", 9.8)
	if err != nil {
		t.Fatalf("failed to check sync: %v", err)
	}

	if math.Abs(correction.Drift) < 0.1 {
		t.Errorf("expected drift around 0.2s, got %f", correction.Drift)
	}
	if correction.Action != "RATE_ADJUST" {
		t.Errorf("expected RATE_ADJUST for ~200ms drift, got %s", correction.Action)
	}
}

func TestHostOnlyPermissions(t *testing.T) {
	memStore := store.NewMemoryStore()
	syncer := NewVideoSyncer(memStore)

	room := &model.Room{
		ID:                 "room2",
		Code:               "TEST02",
		HostID:             "host1",
		OnlyHostCanControl: true, // Only host can control
		CreatedAt:          time.Now(),
	}
	_ = memStore.CreateRoom(room)

	// Non-host user tries to play -> should return ErrUnauthorized
	_, err := syncer.HandleAction(model.EventPlay, "room2", "viewer1", false, 0.0, 1.0)
	if err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}

	// Host user plays -> should succeed
	state, err := syncer.HandleAction(model.EventPlay, "room2", "host1", true, 0.0, 1.0)
	if err != nil {
		t.Fatalf("host should be authorized: %v", err)
	}
	if !state.Playing {
		t.Fatalf("expected playing to be true")
	}
}
