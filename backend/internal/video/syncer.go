package video

import (
	"errors"
	"math"
	"time"
	"watchparty-backend/internal/model"
	"watchparty-backend/internal/store"
)

var (
	ErrUnauthorized = errors.New("only host can control video in this room")
)

type VideoSyncer struct {
	store store.Store
}

func NewVideoSyncer(s store.Store) *VideoSyncer {
	return &VideoSyncer{store: s}
}

// CalculateCurrentPosition calculates the exact authoritative current position
// based on elapsed time since the last update if the video is currently playing.
func (vs *VideoSyncer) CalculateCurrentPosition(state *model.VideoState) float64 {
	if !state.Playing {
		return state.Position
	}

	nowMs := time.Now().UnixMilli()
	elapsedSeconds := float64(nowMs-state.UpdatedAt) / 1000.0

	rate := state.Rate
	if rate <= 0 {
		rate = 1.0
	}

	return state.Position + (elapsedSeconds * rate)
}

// HandleAction processes a PLAY, PAUSE, SEEK, or RATE action from a user
func (vs *VideoSyncer) HandleAction(actionType string, roomID string, userID string, isHost bool, position float64, rate float64) (*model.VideoState, error) {
	room, err := vs.store.GetRoomByID(roomID)
	if err != nil {
		return nil, err
	}

	// Verify permissions
	if room.OnlyHostCanControl && !isHost {
		return nil, ErrUnauthorized
	}

	// Retrieve current state or create default
	currentState, err := vs.store.GetVideoState(roomID)
	if err != nil {
		currentState = &model.VideoState{
			RoomID:    roomID,
			Playing:   false,
			Position:  0.0,
			Rate:      1.0,
			UpdatedAt: time.Now().UnixMilli(),
		}
	}

	nowMs := time.Now().UnixMilli()
	if rate <= 0 {
		rate = 1.0
	}

	switch actionType {
	case model.EventPlay:
		currentState.Playing = true
		currentState.Position = position
		currentState.Rate = rate
		currentState.UpdatedAt = nowMs
		currentState.ChangedBy = userID

	case model.EventPause:
		currentState.Playing = false
		currentState.Position = position
		currentState.Rate = rate
		currentState.UpdatedAt = nowMs
		currentState.ChangedBy = userID

	case model.EventSeek:
		currentState.Position = position
		currentState.UpdatedAt = nowMs
		currentState.ChangedBy = userID

	case model.EventRate:
		// If video was playing, update position up to this instant
		if currentState.Playing {
			currentState.Position = vs.CalculateCurrentPosition(currentState)
		}
		currentState.Rate = rate
		currentState.UpdatedAt = nowMs
		currentState.ChangedBy = userID
	}

	if err := vs.store.SetVideoState(currentState); err != nil {
		return nil, err
	}

	return currentState, nil
}

// CheckSync calculates drift between client and authoritative server position
func (vs *VideoSyncer) CheckSync(roomID string, clientPos float64) (*model.SyncCorrectionPayload, error) {
	state, err := vs.store.GetVideoState(roomID)
	if err != nil {
		return nil, err
	}

	serverPos := vs.CalculateCurrentPosition(state)
	drift := serverPos - clientPos // positive = client is behind; negative = client is ahead
	absDrift := math.Abs(drift)

	payload := &model.SyncCorrectionPayload{
		Playing:         state.Playing,
		ServerPosition:  serverPos,
		Rate:            state.Rate,
		Drift:           drift,
		ServerTimestamp: time.Now().UnixMilli(),
		Action:          "NONE",
		TargetRate:      state.Rate,
	}

	if !state.Playing {
		if absDrift > 0.3 {
			payload.Action = "HARD_SEEK"
		}
		return payload, nil
	}

	// Small drift threshold: 0.1s to 0.7s (100ms - 700ms)
	// Smoothly nudge playback rate slightly without seeking stutter
	if absDrift >= 0.10 && absDrift <= 0.70 {
		payload.Action = "RATE_ADJUST"
		if drift > 0 {
			// Client is slightly behind -> speed up slightly to catch up
			payload.TargetRate = state.Rate * 1.03
		} else {
			// Client is slightly ahead -> slow down slightly
			payload.TargetRate = state.Rate * 0.97
		}
	} else if absDrift > 0.70 {
		// Large drift (>700ms) -> snap currentTime directly
		payload.Action = "HARD_SEEK"
		payload.TargetRate = state.Rate
	}

	return payload, nil
}
