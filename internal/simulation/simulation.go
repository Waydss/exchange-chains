// Package simulation generates and clears large volumes of deterministic
// test data (users, items, wishes) for load-testing the whole application.
package simulation

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/platform/security"
)

// SeedOptions controls the simulated data volume.
type SeedOptions struct {
	// UserCount is the total number of users to create.
	UserCount int
	// MinGives and MaxGives bound the number of items offered per user.
	MinGives int
	MaxGives int
	// MinWishes and MaxWishes bound the number of wishes per user.
	MinWishes int
	MaxWishes int
	// RandomSeed makes the generated data reproducible.
	RandomSeed uint64
	// BatchSize controls how many users are persisted in one COPY batch.
	BatchSize int
}

func (o SeedOptions) normalized() SeedOptions {
	if o.UserCount <= 0 {
		o.UserCount = 100_000
	}
	if o.MinGives <= 0 && o.MaxGives <= 0 {
		o.MinGives, o.MaxGives = 1, 5
	}
	if o.MinGives < 0 {
		o.MinGives = 0
	}
	if o.MaxGives < o.MinGives {
		o.MaxGives = o.MinGives
	}
	if o.MinWishes <= 0 && o.MaxWishes <= 0 {
		o.MinWishes, o.MaxWishes = 1, 5
	}
	if o.MinWishes < 0 {
		o.MinWishes = 0
	}
	if o.MaxWishes < o.MinWishes {
		o.MaxWishes = o.MinWishes
	}
	if o.BatchSize <= 0 {
		o.BatchSize = 5_000
	}
	return o
}

// SeedResult summarizes a completed seeding run.
type SeedResult struct {
	Users   int
	Items   int
	Wishes  int
	Elapsed time.Duration
}

// SeedData fills the database with simulated users, items, and wishes.
func SeedData(ctx context.Context, repo Repository, opts SeedOptions) (SeedResult, error) {
	opts = opts.normalized()

	categories, err := repo.LoadCategories(ctx)
	if err != nil {
		return SeedResult{}, err
	}
	if len(categories) == 0 {
		return SeedResult{}, errors.New("no categories found: run migrations before seeding")
	}

	maxUserID, maxItemID, maxWishID, err := repo.LoadLastIDs(ctx)
	if err != nil {
		return SeedResult{}, err
	}

	passwordHash, err := security.NewPasswordHasher().Hash(simulatedPassword)
	if err != nil {
		return SeedResult{}, fmt.Errorf("hash simulated password: %w", err)
	}

	rng := rand.New(rand.NewPCG(opts.RandomSeed, 1))

	var result SeedResult
	started := time.Now()

	for offset := 0; offset < opts.UserCount; offset += opts.BatchSize {
		batchCount := min(opts.BatchSize, opts.UserCount-offset)

		users, items, wishes := generateBatch(
			rng,
			categories,
			passwordHash,
			opts,
			maxUserID+1+int64(offset),
			maxItemID,
			maxWishID,
			batchCount,
		)

		if err := repo.BulkInsertUsers(ctx, users); err != nil {
			return result, err
		}
		if err := repo.BulkInsertItems(ctx, items); err != nil {
			return result, err
		}
		if err := repo.BulkInsertWishes(ctx, wishes); err != nil {
			return result, err
		}

		maxItemID += int64(len(items))
		maxWishID += int64(len(wishes))
		result.Users += len(users)
		result.Items += len(items)
		result.Wishes += len(wishes)
	}

	if err := repo.SyncSequences(ctx); err != nil {
		return result, err
	}

	result.Elapsed = time.Since(started)
	return result, nil
}

// ClearData removes all simulated users, items, and wishes together with
// every dependent row (chains, barters, participants, photos).
func ClearData(ctx context.Context, repo Repository) error {
	return repo.ClearData(ctx)
}
