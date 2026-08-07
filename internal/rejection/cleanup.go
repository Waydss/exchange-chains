package rejection

import (
	"context"
	"fmt"
	"strconv"
)

const cleanupScanCount int64 = 100

// CleanupExpired removes all rejection members whose score has expired.
func (s *RedisStore) CleanupExpired(ctx context.Context) error {
	maximumScore := strconv.FormatInt(s.now().Unix(), 10)
	pattern := rejectedItemsKeyPrefix + "*"
	var cursor uint64

	for {
		keys, nextCursor, err := s.backend.scan(ctx, cursor, pattern, cleanupScanCount)
		if err != nil {
			return fmt.Errorf("scan rejected item keys: %w", err)
		}

		for _, key := range keys {
			if err := s.backend.removeExpired(ctx, key, maximumScore); err != nil {
				return fmt.Errorf("cleanup rejected item key %q: %w", key, err)
			}
		}

		if nextCursor == 0 {
			return nil
		}
		cursor = nextCursor
	}
}
