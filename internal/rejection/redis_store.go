package rejection

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const rejectedItemsKeyPrefix = "rejected_items:"

type sortedSetBackend interface {
	reject(ctx context.Context, key, member string, score float64, ttl time.Duration) error
	scores(ctx context.Context, key string, members []string) ([]float64, error)
	remove(ctx context.Context, key string, members ...string) error
	scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error)
	removeExpired(ctx context.Context, key, maximumScore string) error
}

// RedisStore persists temporary item rejections in per-user Redis sorted sets.
type RedisStore struct {
	backend sortedSetBackend
	ttl     time.Duration
	now     func() time.Time
}

// NewRedisStore creates a Redis-backed rejection store.
func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{
		backend: &redisBackend{client: client},
		ttl:     ttl,
		now:     time.Now,
	}
}

func newRedisStore(backend sortedSetBackend, ttl time.Duration, now func() time.Time) *RedisStore {
	return &RedisStore{backend: backend, ttl: ttl, now: now}
}

// Reject adds or refreshes one rejected item for a user.
func (s *RedisStore) Reject(ctx context.Context, userID, itemID int64) error {
	if err := validatePositive("user_id", userID); err != nil {
		return err
	}
	if err := validatePositive("item_id", itemID); err != nil {
		return err
	}

	key := rejectedItemsKey(userID)
	member := strconv.FormatInt(itemID, 10)
	expiresAt := s.now().Add(s.ttl).Unix()
	if err := s.backend.reject(ctx, key, member, float64(expiresAt), s.ttl); err != nil {
		return fmt.Errorf("reject item %d for user %d: %w", itemID, userID, err)
	}

	return nil
}

// FilterRejected preserves candidate order and removes active user rejections.
func (s *RedisStore) FilterRejected(ctx context.Context, userID int64, candidateItemIDs []int64) ([]int64, error) {
	if err := validatePositive("user_id", userID); err != nil {
		return nil, err
	}
	if len(candidateItemIDs) == 0 {
		return []int64{}, nil
	}

	members := make([]string, len(candidateItemIDs))
	for index, itemID := range candidateItemIDs {
		if err := validatePositive("candidate_item_id", itemID); err != nil {
			return nil, err
		}
		members[index] = strconv.FormatInt(itemID, 10)
	}

	scores, err := s.backend.scores(ctx, rejectedItemsKey(userID), members)
	if err != nil {
		return nil, fmt.Errorf("get rejected candidates for user %d: %w", userID, err)
	}
	if len(scores) != len(candidateItemIDs) {
		return nil, fmt.Errorf("get rejected candidates for user %d: unexpected score count", userID)
	}

	nowUnix := float64(s.now().Unix())
	allowed := make([]int64, 0, len(candidateItemIDs))
	expiredMembers := make([]string, 0)
	for index, score := range scores {
		if score > nowUnix {
			continue
		}
		allowed = append(allowed, candidateItemIDs[index])
		if score > 0 {
			expiredMembers = append(expiredMembers, members[index])
		}
	}

	if len(expiredMembers) > 0 {
		if err := s.backend.remove(ctx, rejectedItemsKey(userID), expiredMembers...); err != nil {
			return nil, fmt.Errorf("remove expired candidates for user %d: %w", userID, err)
		}
	}

	return allowed, nil
}

func rejectedItemsKey(userID int64) string {
	return rejectedItemsKeyPrefix + strconv.FormatInt(userID, 10)
}

type redisBackend struct {
	client *redis.Client
}

func (b *redisBackend) reject(ctx context.Context, key, member string, score float64, ttl time.Duration) error {
	_, err := b.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
		pipe.Expire(ctx, key, ttl)
		return nil
	})
	return err
}

func (b *redisBackend) scores(ctx context.Context, key string, members []string) ([]float64, error) {
	return b.client.ZMScore(ctx, key, members...).Result()
}

func (b *redisBackend) remove(ctx context.Context, key string, members ...string) error {
	values := make([]any, len(members))
	for index, member := range members {
		values[index] = member
	}
	return b.client.ZRem(ctx, key, values...).Err()
}

func (b *redisBackend) scan(ctx context.Context, cursor uint64, pattern string, count int64) ([]string, uint64, error) {
	return b.client.Scan(ctx, cursor, pattern, count).Result()
}

func (b *redisBackend) removeExpired(ctx context.Context, key, maximumScore string) error {
	return b.client.ZRemRangeByScore(ctx, key, "-inf", maximumScore).Err()
}
