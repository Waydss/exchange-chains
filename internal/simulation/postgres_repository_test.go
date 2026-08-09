package simulation

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPostgresRepositoryRoundTrip exercises the repository against a live
// PostgreSQL. It is skipped unless SIMULATION_TEST_DATABASE_URL is set.
func TestPostgresRepositoryRoundTrip(t *testing.T) {
	dsn := os.Getenv("SIMULATION_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SIMULATION_TEST_DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresRepository(pool)

	if err := repo.ClearData(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}

	categories, err := repo.LoadCategories(ctx)
	if err != nil {
		t.Fatalf("load categories: %v", err)
	}
	if len(categories) != 17 {
		t.Fatalf("want 17 categories, got %d", len(categories))
	}

	if _, _, _, err := repo.LoadLastIDs(ctx); err != nil {
		t.Fatalf("load last ids: %v", err)
	}

	result, err := SeedData(ctx, repo, SeedOptions{
		UserCount:  100,
		MinGives:   1,
		MaxGives:   3,
		MinWishes:  1,
		MaxWishes:  3,
		RandomSeed: 42,
		BatchSize:  50,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if result.Users != 100 {
		t.Fatalf("want 100 users, got %d", result.Users)
	}

	if err := repo.SyncSequences(ctx); err != nil {
		t.Fatalf("sync sequences: %v", err)
	}

	var userCount, itemCount, wishCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM items`).Scan(&itemCount); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM wishes`).Scan(&wishCount); err != nil {
		t.Fatalf("count wishes: %v", err)
	}
	t.Logf("seeded: users=%d items=%d wishes=%d", userCount, itemCount, wishCount)

	if userCount != 100 {
		t.Fatalf("users after seed: want 100, got %d", userCount)
	}

	var newUserID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (login, password_hash, name) VALUES ('seq_check', 'x', 'x') RETURNING id`).Scan(&newUserID); err != nil {
		t.Fatalf("insert after seed (sequence sync check): %v", err)
	}
	if newUserID <= int64(userCount) {
		t.Fatalf("sequence not synced: new user id %d <= %d", newUserID, userCount)
	}

	if err := repo.ClearData(ctx); err != nil {
		t.Fatalf("clear after seed: %v", err)
	}

	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		t.Fatalf("count users after clear: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("users after clear: want 0, got %d", userCount)
	}
}
