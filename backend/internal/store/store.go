package store

import (
	"watchparty-backend/internal/model"
)

// Store defines the interface for persistence (supported by MemoryStore and PostgresStore/Redis)
type Store interface {
	// User operations
	CreateUser(user *model.User) error
	GetUserByID(id string) (*model.User, error)
	GetUserByEmail(email string) (*model.User, error)

	// Room operations
	CreateRoom(room *model.Room) error
	GetRoomByCode(code string) (*model.Room, error)
	GetRoomByID(id string) (*model.Room, error)
	UpdateRoomSettings(roomID string, onlyHostCanControl bool) error

	// Room Member operations
	AddMember(member *model.RoomMember) error
	RemoveMember(roomID, userID string) error
	GetRoomMembers(roomID string) ([]model.RoomMember, error)
	UpdateMemberOnline(roomID, userID string, isOnline bool) error

	// Message operations
	SaveMessage(msg *model.Message) error
	GetRecentMessages(roomID string, limit int) ([]model.Message, error)

	// Video State operations (Redis / Fast In-Memory)
	GetVideoState(roomID string) (*model.VideoState, error)
	SetVideoState(state *model.VideoState) error
}
