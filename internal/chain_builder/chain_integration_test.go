package cb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/simulation"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestBuildChainOnSeededData seeds a dense dataset and runs a real BFS chain
// build against it, proving the seed produces matchable data end-to-end.
func TestBuildChainOnSeededData(t *testing.T) {
	dsn := os.Getenv("SIMULATION_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SIMULATION_TEST_DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	seedRepo := simulation.NewPostgresRepository(pool)
	if err := seedRepo.ClearData(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}

	_, err = simulation.SeedData(ctx, seedRepo, simulation.SeedOptions{
		UserCount:  100_000,
		MinGives:   1,
		MaxGives:   5,
		MinWishes:  1,
		MaxWishes:  5,
		RandomSeed: 7,
		BatchSize:  5_000,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	var seededUsers, seededItems, seededWishes int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&seededUsers); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM items`).Scan(&seededItems); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM wishes`).Scan(&seededWishes); err != nil {
		t.Fatalf("count wishes: %v", err)
	}
	if seededUsers != 100_000 {
		t.Fatalf("users: want 100000, got %d", seededUsers)
	}
	t.Logf("seeded users=%d items=%d wishes=%d", seededUsers, seededItems, seededWishes)

	cbRepo := NewPostgresRepository(pool)

	// Probe a handful of roots; the first one that produces a chain wins.
	rootsChecked := 0
	var totalChains int
	var candidateCount int

	for id := int64(1); id <= 500; id++ {
		account, err := cbRepo.GetAccount(ctx, id)
		if err != nil {
			t.Fatalf("get account %d: %v", id, err)
		}
		if len(account.Wishes) == 0 || len(account.Gives) == 0 {
			continue
		}

		give, ok := account.CurrentGive()
		if !ok {
			continue
		}
		if _, err := cbRepo.FindCandidateAccounts(ctx, give, candidate_limit); err != nil {
			t.Fatalf("find candidates for give %d: %v", give.ID, err)
		}
		candidateCount++

		svc := NewService(cbRepo)
		for i := range account.Wishes {
			chains := svc.buildClosedChains(ctx, &account, i)
			totalChains += len(chains)
			rootsChecked++
		}

		if totalChains > 0 {
			t.Logf("found chains after probing %d roots (candidate sets fetched: %d): total %d chains", rootsChecked, candidateCount, totalChains)
			break
		}
	}

	if totalChains == 0 {
		t.Fatalf("no chains found across %d probed roots", rootsChecked)
	}

	var dbChains int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM chains`).Scan(&dbChains); err != nil {
		t.Fatalf("count chains: %v", err)
	}
	t.Logf("chains persisted in DB: %d", dbChains)
	if dbChains == 0 {
		t.Fatalf("no chains written to DB")
	}

	// Read-only listing must reflect persisted data.
	listedChains, err := cbRepo.ListChains(ctx)
	if err != nil {
		t.Fatalf("list chains: %v", err)
	}
	if len(listedChains) != dbChains {
		t.Fatalf("listed chains %d != db %d", len(listedChains), dbChains)
	}
	for _, chain := range listedChains {
		if len(chain.Barters) == 0 {
			t.Fatalf("chain %d has no barters", chain.ID)
		}
	}
}
