package rejection

import (
	"context"
	"path"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestRedisStoreRejectUsesPerUserSortedSetAndRefreshesScore(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	ttl := 14 * 24 * time.Hour
	backend := newMemoryBackend()
	store := newRedisStore(backend, ttl, func() time.Time { return now })

	for _, itemID := range []int64{101, 205} {
		if err := store.Reject(context.Background(), 42, itemID); err != nil {
			t.Fatalf("reject item %d: %v", itemID, err)
		}
	}
	if err := store.Reject(context.Background(), 7, 101); err != nil {
		t.Fatalf("reject item for second user: %v", err)
	}

	userKey := "rejected_items:42"
	if len(backend.sets[userKey]) != 2 {
		t.Fatalf("members in %q = %d, want 2", userKey, len(backend.sets[userKey]))
	}
	wantExpiration := float64(now.Add(ttl).Unix())
	if backend.sets[userKey]["101"] != wantExpiration || backend.sets[userKey]["205"] != wantExpiration {
		t.Fatalf("scores in %q = %#v, want %v", userKey, backend.sets[userKey], wantExpiration)
	}
	if len(backend.sets["rejected_items:7"]) != 1 {
		t.Fatal("second user must have a separate Redis key")
	}
	if backend.ttls[userKey] != ttl {
		t.Fatalf("key TTL = %s, want %s", backend.ttls[userKey], ttl)
	}

	now = now.Add(time.Hour)
	if err := store.Reject(context.Background(), 42, 101); err != nil {
		t.Fatalf("refresh rejection: %v", err)
	}
	if len(backend.sets[userKey]) != 2 {
		t.Fatalf("refresh created duplicate; members = %d", len(backend.sets[userKey]))
	}
	if got := backend.sets[userKey]["101"]; got != float64(now.Add(ttl).Unix()) {
		t.Fatalf("refreshed score = %v, want %d", got, now.Add(ttl).Unix())
	}
}

func TestRedisStoreFilterRejectedPreservesOrder(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	backend := newMemoryBackend()
	backend.sets["rejected_items:42"] = map[string]float64{
		"101": float64(now.Add(time.Hour).Unix()),
		"205": float64(now.Add(2 * time.Hour).Unix()),
		"318": float64(now.Add(-time.Minute).Unix()),
	}
	store := newRedisStore(backend, time.Hour, func() time.Time { return now })

	got, err := store.FilterRejected(context.Background(), 42, []int64{101, 300, 318, 205, 400})
	if err != nil {
		t.Fatalf("filter rejected: %v", err)
	}
	if want := []int64{300, 318, 400}; !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered candidates = %v, want %v", got, want)
	}
	if backend.scoreCalls != 1 {
		t.Fatalf("mass score calls = %d, want 1", backend.scoreCalls)
	}
	if _, exists := backend.sets["rejected_items:42"]["318"]; exists {
		t.Fatal("expired candidate was not removed")
	}

	got, err = store.FilterRejected(context.Background(), 42, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty candidates = %v, error = %v", got, err)
	}
	if backend.scoreCalls != 1 {
		t.Fatal("empty candidates must not access Redis")
	}
}

func TestRedisStoreCleanupExpired(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	backend := newMemoryBackend()
	backend.sets["rejected_items:42"] = map[string]float64{
		"101": float64(now.Add(-time.Second).Unix()),
		"205": float64(now.Add(time.Hour).Unix()),
	}
	backend.sets["rejected_items:7"] = map[string]float64{
		"318": float64(now.Unix()),
	}
	backend.sets["unrelated:key"] = map[string]float64{"1": 0}
	store := newRedisStore(backend, time.Hour, func() time.Time { return now })

	if err := store.CleanupExpired(context.Background()); err != nil {
		t.Fatalf("cleanup expired: %v", err)
	}
	if _, exists := backend.sets["rejected_items:42"]["101"]; exists {
		t.Fatal("expired item remains after cleanup")
	}
	if _, exists := backend.sets["rejected_items:42"]["205"]; !exists {
		t.Fatal("active item was removed by cleanup")
	}
	if _, exists := backend.sets["rejected_items:7"]; exists {
		t.Fatal("empty sorted set key should disappear")
	}
	if _, exists := backend.sets["unrelated:key"]; !exists {
		t.Fatal("cleanup touched an unrelated key")
	}
}

type memoryBackend struct {
	sets       map[string]map[string]float64
	ttls       map[string]time.Duration
	scoreCalls int
}

func newMemoryBackend() *memoryBackend {
	return &memoryBackend{
		sets: make(map[string]map[string]float64),
		ttls: make(map[string]time.Duration),
	}
}

func (b *memoryBackend) reject(_ context.Context, key, member string, score float64, ttl time.Duration) error {
	if b.sets[key] == nil {
		b.sets[key] = make(map[string]float64)
	}
	b.sets[key][member] = score
	b.ttls[key] = ttl
	return nil
}

func (b *memoryBackend) scores(_ context.Context, key string, members []string) ([]float64, error) {
	b.scoreCalls++
	result := make([]float64, len(members))
	for index, member := range members {
		result[index] = b.sets[key][member]
	}
	return result, nil
}

func (b *memoryBackend) remove(_ context.Context, key string, members ...string) error {
	for _, member := range members {
		delete(b.sets[key], member)
	}
	if len(b.sets[key]) == 0 {
		delete(b.sets, key)
	}
	return nil
}

func (b *memoryBackend) scan(_ context.Context, _ uint64, pattern string, _ int64) ([]string, uint64, error) {
	keys := make([]string, 0)
	for key := range b.sets {
		if matches, _ := path.Match(pattern, key); matches {
			keys = append(keys, key)
		}
	}
	return keys, 0, nil
}

func (b *memoryBackend) removeExpired(_ context.Context, key, maximumScore string) error {
	maximum, err := strconv.ParseFloat(maximumScore, 64)
	if err != nil {
		return err
	}
	for member, score := range b.sets[key] {
		if score <= maximum {
			delete(b.sets[key], member)
		}
	}
	if len(b.sets[key]) == 0 {
		delete(b.sets, key)
	}
	return nil
}
