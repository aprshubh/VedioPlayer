package store

import (
	"errors"
	"sync"
	"watchparty-backend/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
)

// MemoryStore provides a full in-memory, thread-safe implementation of Store
type MemoryStore struct {
	mu          sync.RWMutex
	users       map[string]*model.User          // userID -> User
	usersByMail map[string]*model.User          // email -> User
	rooms       map[string]*model.Room          // roomID -> Room
	roomsByCode map[string]*model.Room          // code -> Room
	members     map[string]map[string]*model.RoomMember // roomID -> userID -> RoomMember
	messages    map[string][]*model.Message     // roomID -> []Message
	videoStates map[string]*model.VideoState    // roomID -> VideoState
}

// NewMemoryStore creates a new thread-safe in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:       make(map[string]*model.User),
		usersByMail: make(map[string]*model.User),
		rooms:       make(map[string]*model.Room),
		roomsByCode: make(map[string]*model.Room),
		members:     make(map[string]map[string]*model.RoomMember),
		messages:    make(map[string][]*model.Message),
		videoStates: make(map[string]*model.VideoState),
	}
}

func (s *MemoryStore) CreateUser(user *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = user
	if user.Email != "" {
		s.usersByMail[user.Email] = user
	}
	return nil
}

func (s *MemoryStore) GetUserByID(id string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *MemoryStore) GetUserByEmail(email string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.usersByMail[email]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *MemoryStore) CreateRoom(room *model.Room) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.ID] = room
	s.roomsByCode[room.Code] = room
	if s.members[room.ID] == nil {
		s.members[room.ID] = make(map[string]*model.RoomMember)
	}
	return nil
}

func (s *MemoryStore) GetRoomByCode(code string) (*model.Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.roomsByCode[code]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) GetRoomByID(id string) (*model.Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rooms[id]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) UpdateRoomSettings(roomID string, onlyHostCanControl bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	r.OnlyHostCanControl = onlyHostCanControl
	return nil
}

func (s *MemoryStore) AddMember(member *model.RoomMember) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.members[member.RoomID] == nil {
		s.members[member.RoomID] = make(map[string]*model.RoomMember)
	}
	s.members[member.RoomID][member.UserID] = member
	return nil
}

func (s *MemoryStore) RemoveMember(roomID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if roomMembers, ok := s.members[roomID]; ok {
		delete(roomMembers, userID)
	}
	return nil
}

func (s *MemoryStore) GetRoomMembers(roomID string) ([]model.RoomMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []model.RoomMember
	if roomMembers, ok := s.members[roomID]; ok {
		for _, m := range roomMembers {
			list = append(list, *m)
		}
	}
	return list, nil
}

func (s *MemoryStore) UpdateMemberOnline(roomID, userID string, isOnline bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if roomMembers, ok := s.members[roomID]; ok {
		if m, found := roomMembers[userID]; found {
			m.IsOnline = isOnline
			return nil
		}
	}
	return ErrNotFound
}

func (s *MemoryStore) SaveMessage(msg *model.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[msg.RoomID] = append(s.messages[msg.RoomID], msg)
	return nil
}

func (s *MemoryStore) GetRecentMessages(roomID string, limit int) ([]model.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := s.messages[roomID]
	if len(all) == 0 {
		return []model.Message{}, nil
	}
	start := 0
	if len(all) > limit {
		start = len(all) - limit
	}
	var res []model.Message
	for _, m := range all[start:] {
		res = append(res, *m)
	}
	return res, nil
}

func (s *MemoryStore) GetVideoState(roomID string) (*model.VideoState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.videoStates[roomID]
	if !ok {
		return nil, ErrNotFound
	}
	return state, nil
}

func (s *MemoryStore) SetVideoState(state *model.VideoState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.videoStates[state.RoomID] = state
	return nil
}
