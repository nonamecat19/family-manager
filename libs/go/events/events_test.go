package events

import (
	"errors"
	"testing"
)

func TestSubjectValidate(t *testing.T) {
	valid := []Subject{
		SubjectFamilyMemberInvited,
		SubjectFinanceTransactionCreated,
		SubjectFinanceBudgetExceeded,
	}
	for _, s := range valid {
		if err := s.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", s, err)
		}
	}

	invalid := []Subject{"", "finance", "finance.transaction", "finance..created", "a.b.c.d"}
	for _, s := range invalid {
		if err := s.Validate(); !errors.Is(err, ErrBadSubject) {
			t.Errorf("Validate(%q) = %v, want ErrBadSubject", s, err)
		}
	}
}

func TestSubjectDomain(t *testing.T) {
	if got := SubjectFinanceTransactionCreated.Domain(); got != "finance" {
		t.Errorf("Domain() = %q, want finance", got)
	}
	if got := SubjectFamilyMemberJoined.Domain(); got != "family" {
		t.Errorf("Domain() = %q, want family", got)
	}
}
