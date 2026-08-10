package barter

import (
	"context"
	"errors"
	"testing"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

func TestServiceValidatesInput(t *testing.T) {
	service := NewService(&fakeRepository{})

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "select method chain id",
			call: func() error {
				_, err := service.SelectTransferMethod(context.Background(), SelectTransferMethodInput{
					BarterID: 1,
					UserID:   1,
					Method:   exchange.TransferMethodMeetup,
				})
				return err
			},
		},
		{
			name: "mark given barter id",
			call: func() error {
				_, err := service.MarkGiven(context.Background(), OperationInput{ChainID: 1, UserID: 1})
				return err
			},
		},
		{
			name: "confirm returned user id",
			call: func() error {
				_, err := service.ConfirmReturned(context.Background(), OperationInput{ChainID: 1, BarterID: 1})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validationError *ValidationError
			if err := tt.call(); !errors.As(err, &validationError) {
				t.Fatalf("error = %v, want ValidationError", err)
			}
		})
	}
}

func TestServiceRejectsInvalidTransferMethod(t *testing.T) {
	service := NewService(&fakeRepository{})

	_, err := service.SelectTransferMethod(context.Background(), SelectTransferMethodInput{
		ChainID:  1,
		BarterID: 1,
		UserID:   1,
		Method:   exchange.TransferMethod("pickup"),
	})

	if !errors.Is(err, ErrInvalidMethod) {
		t.Fatalf("error = %v, want ErrInvalidMethod", err)
	}
}

func TestServiceTrimsRejectReason(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	_, err := service.Reject(context.Background(), RejectInput{
		ChainID:  1,
		BarterID: 1,
		UserID:   1,
		Reason:   "  damaged  ",
	})
	if err != nil {
		t.Fatalf("reject: %v", err)
	}

	if repository.rejectInput.Reason != "damaged" {
		t.Fatalf("reason = %q, want trimmed reason", repository.rejectInput.Reason)
	}
}

type fakeRepository struct {
	rejectInput RejectInput
}

func (r fakeRepository) SelectTransferMethod(context.Context, SelectTransferMethodInput) (OperationResult, error) {
	return OperationResult{}, nil
}

func (r fakeRepository) MarkGiven(context.Context, OperationInput) (OperationResult, error) {
	return OperationResult{}, nil
}

func (r fakeRepository) ConfirmReceived(context.Context, OperationInput) (OperationResult, error) {
	return OperationResult{}, nil
}

func (r *fakeRepository) Reject(_ context.Context, input RejectInput) (OperationResult, error) {
	r.rejectInput = input
	return OperationResult{}, nil
}

func (r fakeRepository) MarkReturnStarted(context.Context, OperationInput) (OperationResult, error) {
	return OperationResult{}, nil
}

func (r fakeRepository) ConfirmReturned(context.Context, OperationInput) (OperationResult, error) {
	return OperationResult{}, nil
}
