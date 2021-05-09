package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/algao1/imgrepo"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type SessionService struct {
	rdb *redis.Client
}

var _ imgrepo.SessionService = (*SessionService)(nil)

func NewSessionService(addr, port, pass string, db int) (*SessionService, error) {
	// Initialize new redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", addr, port),
		Password: pass,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to connect to redis", err)
	}

	return &SessionService{rdb: rdb}, nil
}

func (s *SessionService) NewSession() (string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	uuid := uuid.NewString()
	_, err := s.rdb.Set(ctx, uuid, true, 30*time.Minute).Result()
	if err != nil {
		return "", fmt.Errorf("%q: %w", "unable to set session", err)
	}

	return uuid, nil
}

func (s *SessionService) IsSession(uuid string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
	defer cancel()

	_, err := s.rdb.Get(ctx, uuid).Result()
	if err != nil {
		return fmt.Errorf("%q: %w", "no session found", err)
	}

	return nil
}
