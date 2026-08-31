package store

import (
	"database/sql"
	"fmt"
	"time"
	"watchparty-backend/internal/model"

	_ "github.com/lib/pq"
)

// PostgresStore implements permanent database storage using PostgreSQL
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore establishes a PostgreSQL connection and migrates tables
func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	p := &PostgresStore{db: db}
	if err := p.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return p, nil
}

func (p *PostgresStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(64) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE,
		avatar TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS rooms (
		id VARCHAR(64) PRIMARY KEY,
		code VARCHAR(16) UNIQUE NOT NULL,
		host_id VARCHAR(64) REFERENCES users(id),
		only_host_can_control BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS room_members (
		room_id VARCHAR(64) REFERENCES rooms(id) ON DELETE CASCADE,
		user_id VARCHAR(64) REFERENCES users(id) ON DELETE CASCADE,
		is_host BOOLEAN DEFAULT FALSE,
		is_online BOOLEAN DEFAULT TRUE,
		joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (room_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS messages (
		id VARCHAR(64) PRIMARY KEY,
		room_id VARCHAR(64) REFERENCES rooms(id) ON DELETE CASCADE,
		user_id VARCHAR(64),
		user_name VARCHAR(255),
		avatar TEXT,
		message TEXT NOT NULL,
		is_system BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_room_id ON messages(room_id, created_at ASC);
	`
	_, err := p.db.Exec(schema)
	return err
}

func (p *PostgresStore) CreateUser(u *model.User) error {
	query := `
	INSERT INTO users (id, name, email, avatar, created_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (id) DO UPDATE SET name = $2, avatar = $4;`
	_, err := p.db.Exec(query, u.ID, u.Name, u.Email, u.Avatar, u.CreatedAt)
	return err
}

func (p *PostgresStore) GetUserByID(id string) (*model.User, error) {
	u := &model.User{}
	query := `SELECT id, name, email, avatar, created_at FROM users WHERE id = $1`
	err := p.db.QueryRow(query, id).Scan(&u.ID, &u.Name, &u.Email, &u.Avatar, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return u, err
}

func (p *PostgresStore) GetUserByEmail(email string) (*model.User, error) {
	u := &model.User{}
	query := `SELECT id, name, email, avatar, created_at FROM users WHERE email = $1`
	err := p.db.QueryRow(query, email).Scan(&u.ID, &u.Name, &u.Email, &u.Avatar, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return u, err
}

func (p *PostgresStore) CreateRoom(r *model.Room) error {
	query := `INSERT INTO rooms (id, code, host_id, only_host_can_control, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := p.db.Exec(query, r.ID, r.Code, r.HostID, r.OnlyHostCanControl, r.CreatedAt)
	return err
}

func (p *PostgresStore) GetRoomByCode(code string) (*model.Room, error) {
	r := &model.Room{}
	query := `SELECT id, code, host_id, only_host_can_control, created_at FROM rooms WHERE code = $1`
	err := p.db.QueryRow(query, code).Scan(&r.ID, &r.Code, &r.HostID, &r.OnlyHostCanControl, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return r, err
}

func (p *PostgresStore) GetRoomByID(id string) (*model.Room, error) {
	r := &model.Room{}
	query := `SELECT id, code, host_id, only_host_can_control, created_at FROM rooms WHERE id = $1`
	err := p.db.QueryRow(query, id).Scan(&r.ID, &r.Code, &r.HostID, &r.OnlyHostCanControl, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return r, err
}

func (p *PostgresStore) UpdateRoomSettings(roomID string, onlyHostCanControl bool) error {
	query := `UPDATE rooms SET only_host_can_control = $1 WHERE id = $2`
	_, err := p.db.Exec(query, onlyHostCanControl, roomID)
	return err
}

func (p *PostgresStore) AddMember(m *model.RoomMember) error {
	query := `
	INSERT INTO room_members (room_id, user_id, is_host, is_online, joined_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (room_id, user_id) DO UPDATE SET is_online = $4;`
	_, err := p.db.Exec(query, m.RoomID, m.UserID, m.IsHost, m.IsOnline, m.JoinedAt)
	return err
}

func (p *PostgresStore) RemoveMember(roomID, userID string) error {
	query := `DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`
	_, err := p.db.Exec(query, roomID, userID)
	return err
}

func (p *PostgresStore) GetRoomMembers(roomID string) ([]model.RoomMember, error) {
	query := `
	SELECT rm.room_id, rm.user_id, rm.is_host, rm.is_online, rm.joined_at,
	       u.name, u.email, u.avatar, u.created_at
	FROM room_members rm
	JOIN users u ON rm.user_id = u.id
	WHERE rm.room_id = $1`
	rows, err := p.db.Query(query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.RoomMember
	for rows.Next() {
		var m model.RoomMember
		var u model.User
		if err := rows.Scan(&m.RoomID, &m.UserID, &m.IsHost, &m.IsOnline, &m.JoinedAt,
			&u.Name, &u.Email, &u.Avatar, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.ID = m.UserID
		m.User = u
		members = append(members, m)
	}
	return members, nil
}

func (p *PostgresStore) UpdateMemberOnline(roomID, userID string, isOnline bool) error {
	query := `UPDATE room_members SET is_online = $1 WHERE room_id = $2 AND user_id = $3`
	_, err := p.db.Exec(query, isOnline, roomID, userID)
	return err
}

func (p *PostgresStore) SaveMessage(msg *model.Message) error {
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	query := `INSERT INTO messages (id, room_id, user_id, user_name, avatar, message, is_system, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := p.db.Exec(query, msg.ID, msg.RoomID, msg.UserID, msg.UserName, msg.Avatar, msg.Message, msg.IsSystem, msg.CreatedAt)
	return err
}

func (p *PostgresStore) GetRecentMessages(roomID string, limit int) ([]model.Message, error) {
	query := `
	SELECT id, room_id, user_id, user_name, avatar, message, is_system, created_at
	FROM messages
	WHERE room_id = $1
	ORDER BY created_at DESC
	LIMIT $2`
	rows, err := p.db.Query(query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.RoomID, &m.UserID, &m.UserName, &m.Avatar, &m.Message, &m.IsSystem, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}

	// Reverse to get chronological order (oldest to newest)
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// Fallbacks for VideoState in PostgresStore when Redis is not linked
func (p *PostgresStore) GetVideoState(roomID string) (*model.VideoState, error) {
	return nil, ErrNotFound
}

func (p *PostgresStore) SetVideoState(state *model.VideoState) error {
	return nil
}
