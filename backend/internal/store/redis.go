package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"watchparty-backend/internal/model"

	"github.com/redis/go-redis/v9"
)

// RedisStore handles high-speed video state and presence caching
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore connects to Redis server
func NewRedisStore(addr, password string, db int) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisStore{client: rdb}, nil
}

func (r *RedisStore) SetVideoState(ctx context.Context, state *model.VideoState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("room:%s:video", state.RoomID)
	return r.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (r *RedisStore) GetVideoState(ctx context.Context, roomID string) (*model.VideoState, error) {
	key := fmt.Sprintf("room:%s:video", roomID)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var state model.VideoState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *RedisStore) SetPresence(ctx context.Context, roomID, userID string, isOnline bool) error {
	key := fmt.Sprintf("room:%s:presence", roomID)
	status := "offline"
	if isOnline {
		status = "online"
	}
	return r.client.HSet(ctx, key, userID, status).Err()
}

func (r *RedisStore) GetPresence(ctx context.Context, roomID string) (map[string]string, error) {
	key := fmt.Sprintf("room:%s:presence", roomID)
	return r.client.HGetAll(ctx, key).Result()
}
