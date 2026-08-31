package websocket

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"sync"
	"time"
	"watchparty-backend/internal/model"
	"watchparty-backend/internal/store"
	"watchparty-backend/internal/video"

	"github.com/google/uuid"
)

// Hub maintains the set of active clients and broadcasts messages to rooms
type Hub struct {
	store      store.Store
	syncer     *video.VideoSyncer
	rooms      map[string]map[*Client]bool // roomID -> set of clients
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

func NewHub(s store.Store, v *video.VideoSyncer) *Hub {
	return &Hub{
		store:      s,
		syncer:     v,
		rooms:      make(map[string]map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	if h.rooms[client.RoomID] == nil {
		h.rooms[client.RoomID] = make(map[*Client]bool)
	}
	h.rooms[client.RoomID][client] = true
	h.mu.Unlock()

	// Mark user online in DB
	_ = h.store.UpdateMemberOnline(client.RoomID, client.UserID, true)

	// Broadcast user join event
	h.BroadcastToRoom(client.RoomID, &model.WSMessage{
		Type:      model.EventUserJoin,
		RoomID:    client.RoomID,
		UserID:    client.UserID,
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"userId":   client.UserID,
			"userName": client.UserName,
			"avatar":   client.Avatar,
			"isHost":   client.IsHost,
			"isOnline": true,
		},
	})

	// Send initial state to the newly connected user
	h.sendInitialRoomState(client)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	if roomClients, ok := h.rooms[client.RoomID]; ok {
		if _, exists := roomClients[client]; exists {
			delete(roomClients, client)
			close(client.Send)
			if len(roomClients) == 0 {
				delete(h.rooms, client.RoomID)
			}
		}
	}
	h.mu.Unlock()

	// Update DB presence
	_ = h.store.UpdateMemberOnline(client.RoomID, client.UserID, false)

	// Broadcast user leave/offline event
	h.BroadcastToRoom(client.RoomID, &model.WSMessage{
		Type:      model.EventUserLeave,
		RoomID:    client.RoomID,
		UserID:    client.UserID,
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"userId":   client.UserID,
			"userName": client.UserName,
			"isOnline": false,
		},
	})
}

// sendInitialRoomState provides all needed data when joining or reconnecting
func (h *Hub) sendInitialRoomState(client *Client) {
	room, err := h.store.GetRoomByID(client.RoomID)
	if err != nil {
		return
	}

	members, _ := h.store.GetRoomMembers(client.RoomID)
	messages, _ := h.store.GetRecentMessages(client.RoomID, 50)
	videoState, err := h.store.GetVideoState(client.RoomID)
	if err != nil {
		videoState = &model.VideoState{
			RoomID:    client.RoomID,
			Playing:   false,
			Position:  0.0,
			Rate:      1.0,
			UpdatedAt: time.Now().UnixMilli(),
		}
	}

	// Calculate up-to-date position
	calculatedPos := h.syncer.CalculateCurrentPosition(videoState)
	stateCopy := *videoState
	stateCopy.Position = calculatedPos

	client.SendJSON(&model.WSMessage{
		Type:      model.EventRoomState,
		RoomID:    client.RoomID,
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"room":       room,
			"members":    members,
			"messages":   messages,
			"videoState": stateCopy,
		},
	})
}

// BroadcastToRoom sends a message to all clients in a specific room
func (h *Hub) BroadcastToRoom(roomID string, msg *model.WSMessage) {
	h.mu.RLock()
	clients := make([]*Client, 0)
	if roomClients, ok := h.rooms[roomID]; ok {
		for c := range roomClients {
			clients = append(clients, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range clients {
		_ = c.SendJSON(msg)
	}
}

// BroadcastExcept sends to all room clients except the specified one
func (h *Hub) BroadcastExcept(roomID string, exceptClientID string, msg *model.WSMessage) {
	h.mu.RLock()
	clients := make([]*Client, 0)
	if roomClients, ok := h.rooms[roomID]; ok {
		for c := range roomClients {
			if c.UserID != exceptClientID {
				clients = append(clients, c)
			}
		}
	}
	h.mu.RUnlock()

	for _, c := range clients {
		_ = c.SendJSON(msg)
	}
}

// HandleIncomingMessage parses and routes incoming client events
func (h *Hub) HandleIncomingMessage(client *Client, msg *model.WSMessage) {
	nowMs := time.Now().UnixMilli()

	switch msg.Type {
	case model.EventPlay, model.EventPause, model.EventSeek, model.EventRate:
		var action model.VideoActionPayload
		raw, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(raw, &action); err != nil {
			log.Printf("invalid video payload: %v", err)
			return
		}

		newState, err := h.syncer.HandleAction(msg.Type, client.RoomID, client.UserID, client.IsHost, action.Position, action.Rate)
		if err != nil {
			_ = client.SendJSON(&model.WSMessage{
				Type:      model.EventError,
				RoomID:    client.RoomID,
				Timestamp: nowMs,
				Payload:   map[string]string{"error": err.Error()},
			})
			return
		}

		// Broadcast state change to all members in the room
		h.BroadcastToRoom(client.RoomID, &model.WSMessage{
			Type:      msg.Type,
			RoomID:    client.RoomID,
			UserID:    client.UserID,
			Timestamp: nowMs,
			Payload:   newState,
		})

		// Optionally emit system message in chat
		var sysText string
		switch msg.Type {
		case model.EventPlay:
			sysText = fmt.Sprintf("%s played the video", client.UserName)
		case model.EventPause:
			sysText = fmt.Sprintf("%s paused the video", client.UserName)
		case model.EventSeek:
			sysText = fmt.Sprintf("%s jumped to %.1fs", client.UserName, action.Position)
		case model.EventRate:
			sysText = fmt.Sprintf("%s changed speed to %.2fx", client.UserName, action.Rate)
		}
		if sysText != "" {
			sysMsg := &model.Message{
				ID:        uuid.NewString(),
				RoomID:    client.RoomID,
				UserID:    "system",
				UserName:  "System",
				Message:   sysText,
				IsSystem:  true,
				CreatedAt: time.Now(),
			}
			_ = h.store.SaveMessage(sysMsg)
			h.BroadcastToRoom(client.RoomID, &model.WSMessage{
				Type:      model.EventChatMessage,
				RoomID:    client.RoomID,
				Timestamp: nowMs,
				Payload:   sysMsg,
			})
		}

	case model.EventSyncRequest:
		var req model.SyncRequestPayload
		raw, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(raw, &req); err != nil {
			return
		}

		correction, err := h.syncer.CheckSync(client.RoomID, req.ClientPosition)
		if err != nil {
			return
		}

		_ = client.SendJSON(&model.WSMessage{
			Type:      model.EventSyncCorrection,
			RoomID:    client.RoomID,
			Timestamp: nowMs,
			Payload:   correction,
		})

	case model.EventChatMessage:
		var chatPayload model.ChatPayload
		raw, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(raw, &chatPayload); err != nil {
			return
		}

		sanitizedText := html.EscapeString(chatPayload.Message)
		if len(sanitizedText) == 0 || len(sanitizedText) > 2000 {
			return
		}

		chatMsg := &model.Message{
			ID:        uuid.NewString(),
			RoomID:    client.RoomID,
			UserID:    client.UserID,
			UserName:  client.UserName,
			Avatar:    client.Avatar,
			Message:   sanitizedText,
			IsSystem:  false,
			CreatedAt: time.Now(),
		}

		_ = h.store.SaveMessage(chatMsg)

		h.BroadcastToRoom(client.RoomID, &model.WSMessage{
			Type:      model.EventChatMessage,
			RoomID:    client.RoomID,
			UserID:    client.UserID,
			Timestamp: nowMs,
			Payload:   chatMsg,
		})

	case model.EventTyping:
		var typingPayload model.TypingPayload
		raw, _ := json.Marshal(msg.Payload)
		_ = json.Unmarshal(raw, &typingPayload)
		typingPayload.UserName = client.UserName

		// Broadcast to everyone else in the room
		h.BroadcastExcept(client.RoomID, client.UserID, &model.WSMessage{
			Type:      model.EventTyping,
			RoomID:    client.RoomID,
			UserID:    client.UserID,
			Timestamp: nowMs,
			Payload:   typingPayload,
		})

	case model.EventUpdateSettings:
		if !client.IsHost {
			_ = client.SendJSON(&model.WSMessage{
				Type:      model.EventError,
				RoomID:    client.RoomID,
				Timestamp: nowMs,
				Payload:   map[string]string{"error": "only the host can change room settings"},
			})
			return
		}

		var settings model.UpdateSettingsPayload
		raw, _ := json.Marshal(msg.Payload)
		_ = json.Unmarshal(raw, &settings)

		_ = h.store.UpdateRoomSettings(client.RoomID, settings.OnlyHostCanControl)

		h.BroadcastToRoom(client.RoomID, &model.WSMessage{
			Type:      model.EventUpdateSettings,
			RoomID:    client.RoomID,
			Timestamp: nowMs,
			Payload:   settings,
		})

	case model.EventAudioChangeRequest:
		// Broadcast audio change recommendation to all other room members
		h.BroadcastExcept(client.RoomID, client.UserID, &model.WSMessage{
			Type:      model.EventAudioChangeRequest,
			RoomID:    client.RoomID,
			UserID:    client.UserID,
			Timestamp: nowMs,
			Payload:   msg.Payload,
		})
	}
}
