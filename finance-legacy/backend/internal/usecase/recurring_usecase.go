package usecase

import (
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type RecurringUseCase struct {
	recurringRepo repository.RecurringRepository
	walletRepo    repository.WalletRepository
}

func NewRecurringUseCase(
	recurringRepo repository.RecurringRepository,
	walletRepo repository.WalletRepository,
) *RecurringUseCase {
	return &RecurringUseCase{
		recurringRepo: recurringRepo,
		walletRepo:    walletRepo,
	}
}

func (uc *RecurringUseCase) CreateRecurring(
	userID, walletID int,
	recurringType entity.RecurringType,
	amount float64,
	currency string,
	description string,
	recurrence entity.RecurrencePattern,
	startDate time.Time,
	endDate *time.Time,
) (*entity.Recurring, error) {
	// Verify wallet exists
	_, err := uc.walletRepo.GetByID(walletID)
	if err != nil {
		return nil, err
	}

	// Calculate next execution time
	nextExecution := calculateNextExecution(startDate, recurrence)

	recurring := &entity.Recurring{
		UserID:        userID,
		WalletID:      walletID,
		Type:          recurringType,
		Amount:        amount,
		Currency:      currency,
		Description:   description,
		Recurrence:    recurrence,
		StartDate:     startDate,
		EndDate:       endDate,
		NextExecution: &nextExecution,
	}

	if err := uc.recurringRepo.Create(recurring); err != nil {
		return nil, err
	}

	return recurring, nil
}

func (uc *RecurringUseCase) GetRecurringByUserID(userID int) ([]*entity.Recurring, error) {
	return uc.recurringRepo.GetByUserID(userID)
}

func (uc *RecurringUseCase) GetRecurringByID(id int) (*entity.Recurring, error) {
	return uc.recurringRepo.GetByID(id)
}

func (uc *RecurringUseCase) GetDueRecurring() ([]*entity.Recurring, error) {
	return uc.recurringRepo.GetDueRecurring()
}

func (uc *RecurringUseCase) DeleteRecurring(id int) error {
	return uc.recurringRepo.Delete(id)
}

// calculateNextExecution calculates the next execution time based on recurrence pattern
func calculateNextExecution(startDate time.Time, pattern entity.RecurrencePattern) time.Time {
	now := time.Now()
	if startDate.After(now) {
		return startDate
	}

	switch pattern {
	case entity.RecurrenceDaily:
		next := startDate
		for next.Before(now) || next.Equal(now) {
			next = next.AddDate(0, 0, 1)
		}
		return next
	case entity.RecurrenceWeekly:
		next := startDate
		for next.Before(now) || next.Equal(now) {
			next = next.AddDate(0, 0, 7)
		}
		return next
	case entity.RecurrenceMonthly:
		next := startDate
		for next.Before(now) || next.Equal(now) {
			next = next.AddDate(0, 1, 0)
		}
		return next
	default:
		return now.AddDate(0, 0, 1)
	}
}
