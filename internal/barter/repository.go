package barter

import (
	"context"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

type Repository interface {
	SelectTransferMethod(ctx context.Context, input SelectTransferMethodInput) (OperationResult, error)
	MarkGiven(ctx context.Context, input OperationInput) (OperationResult, error)
	ConfirmReceived(ctx context.Context, input OperationInput) (OperationResult, error)
	Reject(ctx context.Context, input RejectInput) (OperationResult, error)
	MarkReturnStarted(ctx context.Context, input OperationInput) (OperationResult, error)
	ConfirmReturned(ctx context.Context, input OperationInput) (OperationResult, error)
}

type SelectTransferMethodInput struct {
	ChainID  int64
	BarterID int64
	UserID   int64
	Method   exchange.TransferMethod
}

type OperationInput struct {
	ChainID  int64
	BarterID int64
	UserID   int64
}

type RejectInput struct {
	ChainID  int64
	BarterID int64
	UserID   int64
	Reason   string
}

type OperationResult struct {
	ChainID        int64
	BarterID       int64
	BarterStatus   exchange.BarterStatus
	ChainStatus    exchange.Status
	TransferMethod *exchange.TransferMethod
}
