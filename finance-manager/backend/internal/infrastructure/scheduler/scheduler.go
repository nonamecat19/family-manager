package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron            *cron.Cron
	recurringRepo   repository.RecurringRepository
	transactionRepo repository.TransactionRepository
	walletRepo      repository.WalletRepository
}

func NewScheduler(
	recurringRepo repository.RecurringRepository,
	transactionRepo repository.TransactionRepository,
	walletRepo repository.WalletRepository,
) *Scheduler {
	return &Scheduler{
		cron:            cron.New(),
		recurringRepo:   recurringRepo,
		transactionRepo: transactionRepo,
		walletRepo:      walletRepo,
	}
}

func (s *Scheduler) Start() {
	// Run every hour to process due recurring payments
	_, err := s.cron.AddFunc("0 * * * *", s.processRecurringPayments)
	if err != nil {
		log.Fatalf("Failed to schedule recurring payments job: %v", err)
	}

	// Run daily at midnight to process investment rewards
	_, err = s.cron.AddFunc("0 0 * * *", s.processInvestmentRewards)
	if err != nil {
		log.Fatalf("Failed to schedule investment rewards job: %v", err)
	}

	s.cron.Start()
	log.Println("Scheduler started")
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Scheduler stopped")
}

func (s *Scheduler) processRecurringPayments() {
	ctx := context.Background()
	log.Println("Processing recurring payments...")

	dueRecurring, err := s.recurringRepo.GetDueRecurring()
	if err != nil {
		log.Printf("Error fetching due recurring payments: %v", err)
		return
	}

	for _, recurring := range dueRecurring {
		if err := s.executeRecurringPayment(ctx, recurring); err != nil {
			log.Printf("Error executing recurring payment %d: %v", recurring.ID, err)
			continue
		}
	}
}

func (s *Scheduler) executeRecurringPayment(ctx context.Context, recurring *entity.Recurring) error {
	// Get wallet
	wallet, err := s.walletRepo.GetByID(recurring.WalletID)
	if err != nil {
		return err
	}

	// Create transaction
	now := time.Now()
	transactionType := entity.TransactionTypeExpense
	if recurring.Type == entity.RecurringTypeIncome {
		transactionType = entity.TransactionTypeIncome
	}

	transaction := &entity.Transaction{
		WalletID:    recurring.WalletID,
		Type:        transactionType,
		Amount:      recurring.Amount,
		Currency:    recurring.Currency,
		Description: recurring.Description,
		ExecutedAt:  &now,
	}

	if err := s.transactionRepo.Create(transaction); err != nil {
		return err
	}

	// Update wallet balance
	var newBalance float64
	if transactionType == entity.TransactionTypeIncome {
		newBalance = wallet.Balance + recurring.Amount
	} else {
		newBalance = wallet.Balance - recurring.Amount
	}

	if err := s.walletRepo.UpdateBalance(recurring.WalletID, newBalance); err != nil {
		return err
	}

	// Calculate and update next execution time
	nextExecution := calculateNextExecution(recurring.StartDate, recurring.Recurrence, now)
	return s.recurringRepo.UpdateNextExecution(recurring.ID, &nextExecution)
}

func (s *Scheduler) processInvestmentRewards() {
	// This would process daily rewards for active investments
	// Implementation would depend on investment repository methods
	log.Println("Processing investment rewards...")
	// TODO: Implement investment reward processing
}

// calculateNextExecution calculates the next execution time based on recurrence pattern
func calculateNextExecution(startDate time.Time, pattern entity.RecurrencePattern, currentTime time.Time) time.Time {
	switch pattern {
	case entity.RecurrenceDaily:
		return currentTime.AddDate(0, 0, 1)
	case entity.RecurrenceWeekly:
		return currentTime.AddDate(0, 0, 7)
	case entity.RecurrenceMonthly:
		return currentTime.AddDate(0, 1, 0)
	default:
		return currentTime.AddDate(0, 0, 1)
	}
}
