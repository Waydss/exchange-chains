// Package matching defines the future exchange cycle matching contract.
package matching

// Cycle is a future matching result represented by ordered item and user IDs.
type Cycle struct {
	ItemIDs []int64
	UserIDs []int64
}
