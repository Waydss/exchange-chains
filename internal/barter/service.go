package barter

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

const maxRejectionReasonLength = 500

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) SelectTransferMethod(
	ctx context.Context,
	input SelectTransferMethodInput,
) (OperationResult, error) {
	if err := validateOperation(input.ChainID, input.BarterID, input.UserID); err != nil {
		return OperationResult{}, err
	}
	if !validTransferMethod(input.Method) {
		return OperationResult{}, ErrInvalidMethod
	}

	result, err := s.repository.SelectTransferMethod(ctx, input)
	if err != nil {
		return OperationResult{}, fmt.Errorf("select barter transfer method: %w", err)
	}

	return result, nil
}

func (s *Service) MarkGiven(ctx context.Context, input OperationInput) (OperationResult, error) {
	if err := validateOperation(input.ChainID, input.BarterID, input.UserID); err != nil {
		return OperationResult{}, err
	}

	result, err := s.repository.MarkGiven(ctx, input)
	if err != nil {
		return OperationResult{}, fmt.Errorf("mark barter given: %w", err)
	}

	return result, nil
}

func (s *Service) ConfirmReceived(ctx context.Context, input OperationInput) (OperationResult, error) {
	if err := validateOperation(input.ChainID, input.BarterID, input.UserID); err != nil {
		return OperationResult{}, err
	}

	result, err := s.repository.ConfirmReceived(ctx, input)
	if err != nil {
		return OperationResult{}, fmt.Errorf("confirm barter received: %w", err)
	}

	return result, nil
}

func (s *Service) Reject(ctx context.Context, input RejectInput) (OperationResult, error) {
	if err := validateOperation(input.ChainID, input.BarterID, input.UserID); err != nil {
		return OperationResult{}, err
	}

	input.Reason = strings.TrimSpace(input.Reason)
	if utf8.RuneCountInString(input.Reason) > maxRejectionReasonLength {
		return OperationResult{}, &ValidationError{Message: "reason must not exceed 500 characters"}
	}

	result, err := s.repository.Reject(ctx, input)
	if err != nil {
		return OperationResult{}, fmt.Errorf("reject barter: %w", err)
	}

	return result, nil
}

func (s *Service) MarkReturnStarted(ctx context.Context, input OperationInput) (OperationResult, error) {
	if err := validateOperation(input.ChainID, input.BarterID, input.UserID); err != nil {
		return OperationResult{}, err
	}

	result, err := s.repository.MarkReturnStarted(ctx, input)
	if err != nil {
		return OperationResult{}, fmt.Errorf("mark barter return started: %w", err)
	}

	return result, nil
}

func (s *Service) ConfirmReturned(ctx context.Context, input OperationInput) (OperationResult, error) {
	if err := validateOperation(input.ChainID, input.BarterID, input.UserID); err != nil {
		return OperationResult{}, err
	}

	result, err := s.repository.ConfirmReturned(ctx, input)
	if err != nil {
		return OperationResult{}, fmt.Errorf("confirm barter returned: %w", err)
	}

	return result, nil
}

func validateOperation(chainID, barterID, userID int64) error {
	if err := validatePositive("chain_id", chainID); err != nil {
		return err
	}
	if err := validatePositive("barter_id", barterID); err != nil {
		return err
	}
	if err := validatePositive("user_id", userID); err != nil {
		return err
	}

	return nil
}

func validTransferMethod(method exchange.TransferMethod) bool {
	switch method {
	case exchange.TransferMethodMeetup, exchange.TransferMethodDelivery:
		return true
	default:
		return false
	}
}
