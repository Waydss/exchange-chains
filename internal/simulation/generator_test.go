package simulation

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/category"
)

func TestGenerateBatchIDsAreSequentialAndLinked(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	categories := []category.Category{
		{ID: 1, Name: "Гитары", CreatedAt: time.Now().UTC()},
		{ID: 2, Name: "Пианино", CreatedAt: time.Now().UTC()},
		{ID: 3, Name: "Палатки", CreatedAt: time.Now().UTC()},
	}

	users, items, wishes := generateBatch(rng, categories, "hash", SeedOptions{
		MinGives:  1,
		MaxGives:  4,
		MinWishes: 1,
		MaxWishes: 3,
	}, 10, 100, 1000, 5)

	if len(users) != 5 {
		t.Fatalf("want 5 users, got %d", len(users))
	}
	if len(items) < 5 {
		t.Fatalf("want at least 5 items, got %d", len(items))
	}
	if len(wishes) < 5 {
		t.Fatalf("want at least 5 wishes, got %d", len(wishes))
	}

	for i, u := range users {
		if u.ID != int64(10+i) {
			t.Errorf("user %d: want id %d, got %d", i, 10+i, u.ID)
		}
	}

	userIDs := map[int64]bool{}
	for _, u := range users {
		userIDs[u.ID] = true
	}

	for i, it := range items {
		if !userIDs[it.UserID] {
			t.Errorf("item %d: user_id %d not in batch", i, it.UserID)
		}
		if it.PriceMin <= 0 || it.PriceMax < it.PriceMin {
			t.Errorf("item %d: invalid price range %d..%d", i, it.PriceMin, it.PriceMax)
		}
	}

	for i, w := range wishes {
		if !userIDs[w.UserID] {
			t.Errorf("wish %d: user_id %d not in batch", i, w.UserID)
		}
		if w.PriceMin <= 0 || w.PriceMax < w.PriceMin {
			t.Errorf("wish %d: invalid price range %d..%d", i, w.PriceMin, w.PriceMax)
		}
	}

	if items[0].ID != 101 {
		t.Errorf("first item: want id 101, got %d", items[0].ID)
	}
	if wishes[0].ID != 1001 {
		t.Errorf("first wish: want id 1001, got %d", wishes[0].ID)
	}
}

func TestSeedOptionsNormalized(t *testing.T) {
	opts := SeedOptions{}.normalized()
	if opts.UserCount != 100_000 {
		t.Errorf("UserCount: want 100000, got %d", opts.UserCount)
	}
	if opts.MinGives != 1 || opts.MaxGives != 5 {
		t.Errorf("gives bounds: want 1..5, got %d..%d", opts.MinGives, opts.MaxGives)
	}
	if opts.BatchSize != 5_000 {
		t.Errorf("BatchSize: want 5000, got %d", opts.BatchSize)
	}

	opts = SeedOptions{MinGives: 3, MaxGives: 2}.normalized()
	if opts.MinGives != 3 || opts.MaxGives != 3 {
		t.Errorf("max below min: want 3..3, got %d..%d", opts.MinGives, opts.MaxGives)
	}
}

func TestDeterministicWithSameSeed(t *testing.T) {
	opts := SeedOptions{MinGives: 1, MaxGives: 4, MinWishes: 1, MaxWishes: 3, RandomSeed: 42}
	categories := []category.Category{
		{ID: 1, Name: "Гитары"},
		{ID: 2, Name: "Пианино"},
		{ID: 3, Name: "Палатки"},
	}

	run := func() []string {
		rng := rand.New(rand.NewPCG(opts.RandomSeed, 1))
		_, items, wishes := generateBatch(rng, categories, "hash", opts, 0, 0, 0, 100)
		titles := make([]string, 0, len(items)+len(wishes))
		for _, it := range items {
			titles = append(titles, it.Title)
		}
		for _, w := range wishes {
			titles = append(titles, w.TitleKeywords)
		}
		return titles
	}

	first := run()
	second := run()
	if len(first) != len(second) {
		t.Fatalf("determinism broken: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("determinism broken at %d: %q vs %q", i, first[i], second[i])
		}
	}
}
