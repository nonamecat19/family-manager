package topics

import "testing"

func TestValidAcceptsKeysAndDomains(t *testing.T) {
	for _, s := range []string{"finance", "family", "recipes", FinanceBudgetExceeded, FamilyMemberJoined} {
		if !Valid(s) {
			t.Errorf("Valid(%q) = false", s)
		}
	}
	for _, s := range []string{"", "notes", "finance.budget", "finance.budget.created", "FINANCE"} {
		if Valid(s) {
			t.Errorf("Valid(%q) = true", s)
		}
	}
}

func TestMutedByDomainOrKey(t *testing.T) {
	if !Muted(map[string]bool{"finance": true}, FinanceBudgetExceeded) {
		t.Error("a muted domain did not mute its topic")
	}
	if !Muted(map[string]bool{FamilyMemberJoined: true}, FamilyMemberJoined) {
		t.Error("a muted topic was delivered")
	}
	if Muted(map[string]bool{FamilyMemberJoined: true}, FamilyMemberRemoved) {
		t.Error("muting one family topic muted another")
	}
}
