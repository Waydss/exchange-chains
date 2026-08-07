// Package rejection records temporarily rejected items and exposes the chain-decline use case.
package rejection

import "context"

// Store manages per-user temporary item rejections.
type Store interface {
	Reject(ctx context.Context, userID, itemID int64) error
	FilterRejected(ctx context.Context, userID int64, candidateItemIDs []int64) ([]int64, error)
	CleanupExpired(ctx context.Context) error
}
