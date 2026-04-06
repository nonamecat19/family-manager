package postgres

import (
	"context"
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type recurringRepository struct {
	queries *Queries
}

func NewRecurringRepository(queries *Queries) repository.RecurringRepository {
	return &recurringRepository{queries: queries}
}

func (r *recurringRepository) Create(recurring *entity.Recurring) error {
	ctx := context.Background()
	
	var descPtr *string
	if recurring.Description != "" {
		descPtr = &recurring.Description
	}
	
	params := CreateRecurringParams{
		UserID:        intToPgtypeInt4(recurring.UserID),
		WalletID:      intToPgtypeInt4(recurring.WalletID),
		Type:          string(recurring.Type),
		Amount:        float64ToNumeric(recurring.Amount),
		Currency:      recurring.Currency,
		Description:   stringPtrToPgtypeText(descPtr),
		Recurrence:    string(recurring.Recurrence),
		StartDate:     timeToPgtypeDate(recurring.StartDate),
		EndDate:       timePtrToPgtypeDate(recurring.EndDate),
		NextExecution: timePtrToPgtype(recurring.NextExecution),
	}
	
	dbRecurring, err := r.queries.CreateRecurring(ctx, params)
	if err != nil {
		return err
	}
	
	recurring.ID = int(dbRecurring.ID)
	return nil
}

func (r *recurringRepository) GetByID(id int) (*entity.Recurring, error) {
	ctx := context.Background()
	
	dbRecurring, err := r.queries.GetRecurringByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	
	return toDomainRecurring(dbRecurring), nil
}

func (r *recurringRepository) GetByUserID(userID int) ([]*entity.Recurring, error) {
	ctx := context.Background()
	
	dbRecurring, err := r.queries.GetRecurringByUserID(ctx, intToPgtypeInt4(userID))
	if err != nil {
		return nil, err
	}
	
	recurring := make([]*entity.Recurring, len(dbRecurring))
	for i, rec := range dbRecurring {
		recurring[i] = toDomainRecurring(rec)
	}
	
	return recurring, nil
}

func (r *recurringRepository) GetDueRecurring() ([]*entity.Recurring, error) {
	ctx := context.Background()
	
	dbRecurring, err := r.queries.GetDueRecurring(ctx)
	if err != nil {
		return nil, err
	}
	
	recurring := make([]*entity.Recurring, len(dbRecurring))
	for i, rec := range dbRecurring {
		recurring[i] = toDomainRecurring(rec)
	}
	
	return recurring, nil
}

func (r *recurringRepository) Update(recurring *entity.Recurring) error {
	ctx := context.Background()
	
	var descPtr *string
	if recurring.Description != "" {
		descPtr = &recurring.Description
	}
	
	params := UpdateRecurringParams{
		ID:            int32(recurring.ID),
		WalletID:      intToPgtypeInt4(recurring.WalletID),
		Type:          string(recurring.Type),
		Amount:        float64ToNumeric(recurring.Amount),
		Currency:      recurring.Currency,
		Description:   stringPtrToPgtypeText(descPtr),
		Recurrence:    string(recurring.Recurrence),
		StartDate:     timeToPgtypeDate(recurring.StartDate),
		EndDate:       timePtrToPgtypeDate(recurring.EndDate),
		NextExecution: timePtrToPgtype(recurring.NextExecution),
	}
	
	_, err := r.queries.UpdateRecurring(ctx, params)
	return err
}

func (r *recurringRepository) UpdateNextExecution(id int, nextExecution *time.Time) error {
	ctx := context.Background()
	
	params := UpdateRecurringNextExecutionParams{
		ID:            int32(id),
		NextExecution: timePtrToPgtype(nextExecution),
	}
	
	_, err := r.queries.UpdateRecurringNextExecution(ctx, params)
	return err
}

func (r *recurringRepository) Delete(id int) error {
	ctx := context.Background()
	return r.queries.DeleteRecurring(ctx, int32(id))
}

