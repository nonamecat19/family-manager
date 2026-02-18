package postgres

import (
	"strconv"
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/jackc/pgx/v5/pgtype"
)

// Helper functions to convert between SQLC models and domain entities

func pgtypeToTime(t pgtype.Timestamp) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func timeToPgtype(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: t, Valid: true}
}

func timePtrToPgtype(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}

func pgtypeToTimePtr(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func pgtypeDateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

func timeToPgtypeDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func timePtrToPgtypeDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func pgtypeDateToTimePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	return &d.Time
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil {
		return 0
	}
	return f.Float64
}

func float64ToNumeric(f float64) pgtype.Numeric {
	// Convert float64 to string, then to numeric
	// Use maximum precision to preserve very small decimal values
	// Database schema for wallet balance is NUMERIC(30, 18) which supports up to 18 decimal places
	n := pgtype.Numeric{}
	
	// For very small or very large numbers, use 'e' format (scientific notation) to preserve precision
	// For normal numbers, use 'f' format with enough precision
	// strconv.FormatFloat with 'g' format automatically chooses the most compact representation
	// Using precision of 17 (maximum for float64) to preserve all significant digits
	var str string
	if f == 0 {
		str = "0"
	} else if f < 1e-10 || f > 1e10 {
		// Use scientific notation for very small or very large numbers
		str = strconv.FormatFloat(f, 'e', 17, 64)
	} else {
		// Use fixed point notation with maximum precision for normal numbers
		// Use 17 decimal places to preserve float64's full precision
		str = strconv.FormatFloat(f, 'f', 17, 64)
	}
	
	_ = n.Scan(str)
	return n
}

func pgtypeIntToInt(i pgtype.Int4) int {
	if !i.Valid {
		return 0
	}
	return int(i.Int32)
}

func intToPgtypeInt4(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

func pgtypeTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func stringToPgtypeText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func stringPtrToPgtypeText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func pgtypeTextToStringPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// Convert SQLC User to domain entity
func toDomainUser(u User) *entity.User {
	return &entity.User{
		ID:           int(u.ID),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    pgtypeToTime(u.CreatedAt),
	}
}

// Convert SQLC Wallet to domain entity
func toDomainWallet(w Wallet) *entity.Wallet {
	return &entity.Wallet{
		ID:        int(w.ID),
		UserID:    pgtypeIntToInt(w.UserID),
		Name:      w.Name,
		Type:      entity.WalletType(w.Type),
		Currency:  w.Currency,
		Balance:   numericToFloat64(w.Balance),
		Category:  pgtypeTextToStringPtr(w.Category),
		CreatedAt: pgtypeToTime(w.CreatedAt),
	}
}

// Convert SQLC Wallet Row types to domain entity
func toDomainWalletFromRow(w interface {
	GetID() int32
	GetUserID() pgtype.Int4
	GetName() string
	GetType() string
	GetCurrency() string
	GetBalance() pgtype.Numeric
	GetCategory() pgtype.Text
	GetCreatedAt() pgtype.Timestamp
}) *entity.Wallet {
	return &entity.Wallet{
		ID:        int(w.GetID()),
		UserID:    pgtypeIntToInt(w.GetUserID()),
		Name:      w.GetName(),
		Type:      entity.WalletType(w.GetType()),
		Currency:  w.GetCurrency(),
		Balance:   numericToFloat64(w.GetBalance()),
		Category:  pgtypeTextToStringPtr(w.GetCategory()),
		CreatedAt: pgtypeToTime(w.GetCreatedAt()),
	}
}

// Helper functions for Row types
func createWalletRowToDomain(w CreateWalletRow) *entity.Wallet {
	return &entity.Wallet{
		ID:        int(w.ID),
		UserID:    pgtypeIntToInt(w.UserID),
		Name:      w.Name,
		Type:      entity.WalletType(w.Type),
		Currency:  w.Currency,
		Balance:   numericToFloat64(w.Balance),
		Category:  pgtypeTextToStringPtr(w.Category),
		CreatedAt: pgtypeToTime(w.CreatedAt),
	}
}

func getWalletByIDRowToDomain(w GetWalletByIDRow) *entity.Wallet {
	return &entity.Wallet{
		ID:        int(w.ID),
		UserID:    pgtypeIntToInt(w.UserID),
		Name:      w.Name,
		Type:      entity.WalletType(w.Type),
		Currency:  w.Currency,
		Balance:   numericToFloat64(w.Balance),
		Category:  pgtypeTextToStringPtr(w.Category),
		CreatedAt: pgtypeToTime(w.CreatedAt),
	}
}

func getWalletsByUserIDRowToDomain(w GetWalletsByUserIDRow) *entity.Wallet {
	return &entity.Wallet{
		ID:        int(w.ID),
		UserID:    pgtypeIntToInt(w.UserID),
		Name:      w.Name,
		Type:      entity.WalletType(w.Type),
		Currency:  w.Currency,
		Balance:   numericToFloat64(w.Balance),
		Category:  pgtypeTextToStringPtr(w.Category),
		CreatedAt: pgtypeToTime(w.CreatedAt),
	}
}

// Convert SQLC Transaction to domain entity
func toDomainTransaction(t Transaction) *entity.Transaction {
	return &entity.Transaction{
		ID:          int(t.ID),
		WalletID:    pgtypeIntToInt(t.WalletID),
		Type:        entity.TransactionType(t.Type),
		Amount:      numericToFloat64(t.Amount),
		Currency:    t.Currency,
		Description: pgtypeTextToString(t.Description),
		Category:    pgtypeTextToString(t.Category),
		CreatedAt:   pgtypeToTime(t.CreatedAt),
		ExecutedAt:  pgtypeToTimePtr(t.ExecutedAt),
	}
}

// Convert SQLC Recurring to domain entity
func toDomainRecurring(r Recurring) *entity.Recurring {
	return &entity.Recurring{
		ID:            int(r.ID),
		UserID:        pgtypeIntToInt(r.UserID),
		WalletID:      pgtypeIntToInt(r.WalletID),
		Type:          entity.RecurringType(r.Type),
		Amount:        numericToFloat64(r.Amount),
		Currency:      r.Currency,
		Description:   pgtypeTextToString(r.Description),
		Recurrence:    entity.RecurrencePattern(r.Recurrence),
		StartDate:     pgtypeDateToTime(r.StartDate),
		EndDate:       pgtypeDateToTimePtr(r.EndDate),
		NextExecution: pgtypeToTimePtr(r.NextExecution),
	}
}

// Convert SQLC Investment to domain entity
func toDomainInvestment(i Investment) *entity.Investment {
	return &entity.Investment{
		ID:          int(i.ID),
		UserID:      pgtypeIntToInt(i.UserID),
		WalletID:    pgtypeIntToInt(i.WalletID),
		Type:        entity.InvestmentType(i.Type),
		Coin:        pgtypeTextToString(i.Coin),
		Amount:      numericToFloat64(i.Amount),
		StartDate:   pgtypeDateToTime(i.StartDate),
		EndDate:     pgtypeDateToTimePtr(i.EndDate),
		DailyReward: numericToFloat64(i.DailyReward),
		Status:      entity.InvestmentStatus(i.Status),
	}
}

// Convert SQLC Currency to domain entity
func toDomainCurrency(c Currency) *entity.Currency {
	return &entity.Currency{
		ID:           int(c.ID),
		Code:         c.Code,
		Type:         entity.CurrencyType(c.Type),
		ExchangeRate: numericToFloat64(c.ExchangeRate),
		Name:         c.Name,
	}
}
