package redis

import (
	"context"
	"fmt"
	"time"

	"user-service/internal/domain"
)

type UserCache struct {
	client *RedisClient
	ttl    time.Duration
}

func NewUserCache(client *RedisClient, ttl time.Duration) *UserCache {
	return &UserCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *UserCache) Set(ctx context.Context, user *domain.User) error {
	key := c.userKey(user.ID)
	return c.client.Set(ctx, key, user, c.ttl)
}

func (c *UserCache) Get(ctx context.Context, userID uint) (*domain.User, error) {
	key := c.userKey(userID)
	var user domain.User

	err := c.client.Get(ctx, key, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *UserCache) Delete(ctx context.Context, userID uint) error {
	key := c.userKey(userID)
	return c.client.Delete(ctx, key)
}

func (c *UserCache) SetByEmail(ctx context.Context, email string, user *domain.User) error {
	key := c.emailKey(email)
	return c.client.Set(ctx, key, user, c.ttl)
}

func (c *UserCache) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	key := c.emailKey(email)
	var user domain.User

	err := c.client.Get(ctx, key, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *UserCache) DeleteByEmail(ctx context.Context, email string) error {
	key := c.emailKey(email)
	return c.client.Delete(ctx, key)
}

func (c *UserCache) userKey(userID uint) string {
	return fmt.Sprintf("user:id:%d", userID)
}

func (c *UserCache) emailKey(email string) string {
	return fmt.Sprintf("user:email:%s", email)
}
