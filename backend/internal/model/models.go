package model

import "time"

// User represents an authenticated or guest user
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"createdAt"`
}

// Room represents a watch party room
type Room struct {
	ID                 string    `json:"id"`
	Code               string    `json:"code"`
	HostID             string    `json:"hostId"`
	OnlyHostCanControl bool      `json:"onlyHostCanControl"`
	CreatedAt          time.Time `json:"createdAt"`
}

// RoomMember represents a user joined in a room
type RoomMember struct {
	RoomID   string    `json:"roomId"`
	UserID   string    `json:"userId"`
	User     User      `json:"user"`
	IsHost   bool      `json:"isHost"`
	JoinedAt time.Time `json:"joinedAt"`
	IsOnline bool      `json:"isOnline"`
}

// Message represents a chat message in a room
type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"roomId"`
	UserID    string    `json:"userId"`
	UserName  string    `json:"userName"`
	Avatar    string    `json:"avatar"`
	Message   string    `json:"message"`
	IsSystem  bool      `json:"isSystem"`
	CreatedAt time.Time `json:"createdAt"`
}

// VideoState represents the authoritative playback state of a room
type VideoState struct {
	RoomID    string  `json:"roomId"`
	Playing   bool    `json:"playing"`
	Position  float64 `json:"position"`  // seconds
	Rate      float64 `json:"rate"`      // playback rate (e.g. 1.0, 1.25, 1.5)
	UpdatedAt int64   `json:"updatedAt"` // Unix timestamp in milliseconds
	ChangedBy string  `json:"changedBy"` // User name or ID who initiated change
}

// Event Types for WebSocket
const (
	EventPlay           = "VIDEO_PLAY"
	EventPause          = "VIDEO_PAUSE"
	EventSeek           = "VIDEO_SEEK"
	EventRate           = "VIDEO_RATE"
	EventSyncRequest    = "SYNC_REQUEST"
	EventSyncCorrection = "SYNC_CORRECTION"
	EventChatMessage    = "CHAT_MESSAGE"
	EventUserJoin       = "USER_JOIN"
	EventUserLeave      = "USER_LEAVE"
	EventPresence       = "PRESENCE_UPDATE"
	EventTyping         = "TYPING"
	EventRoomState      = "ROOM_STATE"
	EventUpdateSettings     = "UPDATE_SETTINGS"
	EventAudioChangeRequest = "AUDIO_CHANGE_REQUEST"
	EventError              = "ERROR"
)

// WSMessage represents the standard WebSocket envelope
type WSMessage struct {
	Type      string      `json:"type"`
	RoomID    string      `json:"roomId,omitempty"`
	UserID    string      `json:"userId,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// VideoActionPayload represents client actions for PLAY, PAUSE, SEEK, RATE
type VideoActionPayload struct {
	Position float64 `json:"position"`
	Rate     float64 `json:"rate,omitempty"`
}

// SyncRequestPayload is sent periodically by clients to check drift
type SyncRequestPayload struct {
	ClientPosition  float64 `json:"clientPosition"`
	ClientTimestamp int64   `json:"clientTimestamp"`
}

// SyncCorrectionPayload is sent by server with authoritative timing
type SyncCorrectionPayload struct {
	Playing         bool    `json:"playing"`
	ServerPosition  float64 `json:"serverPosition"`
	Rate            float64 `json:"rate"`
	Drift           float64 `json:"drift"`           // ServerPosition - ClientPosition
	ServerTimestamp int64   `json:"serverTimestamp"`
	Action          string  `json:"action"`          // "NONE", "RATE_ADJUST", "HARD_SEEK"
	TargetRate      float64 `json:"targetRate"`      // suggested temporary playback rate
}

// ChatPayload represents a chat message payload
type ChatPayload struct {
	Message string `json:"message"`
}

// TypingPayload represents a user typing status
type TypingPayload struct {
	IsTyping bool   `json:"isTyping"`
	UserName string `json:"userName"`
}

// UpdateSettingsPayload for room control mode
type UpdateSettingsPayload struct {
	OnlyHostCanControl bool `json:"onlyHostCanControl"`
}
