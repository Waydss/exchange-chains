package rejection

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisStoreIntegration(t *testing.T) {
	redisAddress := os.Getenv("REDIS_TEST_ADDR")
	if redisAddress == "" {
		t.Skip("REDIS_TEST_ADDR is not configured")
	}

	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: redisAddress})
	defer func() { _ = client.Close() }()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping integration Redis: %v", err)
	}

	userID := time.Now().UnixNano()%1_000_000_000 + 1_000_000_000
	otherUserID := userID + 1
	keys := []string{rejectedItemsKey(userID), rejectedItemsKey(otherUserID)}
	defer func() { _ = client.Del(ctx, keys...).Err() }()

	now := time.Now().Truncate(time.Second)
	store := NewRedisStore(client, 14*24*time.Hour)
	store.now = func() time.Time { return now }

	for _, itemID := range []int64{101, 205} {
		if err := store.Reject(ctx, userID, itemID); err != nil {
			t.Fatalf("reject item %d: %v", itemID, err)
		}
	}
	if err := store.Reject(ctx, otherUserID, 101); err != nil {
		t.Fatalf("reject item for other user: %v", err)
	}
	if cardinality := client.ZCard(ctx, keys[0]).Val(); cardinality != 2 {
		t.Fatalf("user sorted set cardinality = %d, want 2", cardinality)
	}
	if exists := client.Exists(ctx, keys[1]).Val(); exists != 1 {
		t.Fatal("other user did not receive a separate Redis key")
	}

	oldScore := client.ZScore(ctx, keys[0], "101").Val()
	now = now.Add(time.Hour)
	if err := store.Reject(ctx, userID, 101); err != nil {
		t.Fatalf("refresh item rejection: %v", err)
	}
	if newScore := client.ZScore(ctx, keys[0], "101").Val(); newScore <= oldScore {
		t.Fatalf("refreshed score = %v, want greater than %v", newScore, oldScore)
	}
	if cardinality := client.ZCard(ctx, keys[0]).Val(); cardinality != 2 {
		t.Fatal("refresh created a duplicate sorted set member")
	}

	filtered, err := store.FilterRejected(ctx, userID, []int64{101, 300, 205})
	if err != nil {
		t.Fatalf("filter rejected: %v", err)
	}
	if !reflect.DeepEqual(filtered, []int64{300}) {
		t.Fatalf("filtered candidates = %v, want [300]", filtered)
	}

	expiredItemID := int64(318)
	if err := client.ZAdd(ctx, keys[0], redis.Z{
		Score:  float64(now.Add(-time.Second).Unix()),
		Member: fmt.Sprintf("%d", expiredItemID),
	}).Err(); err != nil {
		t.Fatalf("add expired item: %v", err)
	}
	if err := store.CleanupExpired(ctx); err != nil {
		t.Fatalf("cleanup expired items: %v", err)
	}
	if _, err := client.ZScore(ctx, keys[0], fmt.Sprintf("%d", expiredItemID)).Result(); err != redis.Nil {
		t.Fatalf("expired item lookup error = %v, want redis.Nil", err)
	}
}
