package cb

import "context"

// RejectionWriter contains the Redis write operation required by the service.
type RejectionWriter interface {
	Reject(ctx context.Context, userID, itemID int64) error
}

// RejectionStore manages per-user temporary item rejections.
type RejectionStore interface {
	RejectionWriter
	FilterRejected(ctx context.Context, userID int64, candidateItemIDs []int64) ([]int64, error)
	CleanupExpired(ctx context.Context) error
}
