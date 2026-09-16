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

func TestTasksTopicsAreInTheCatalog(t *testing.T) {
	for _, key := range []string{TasksTaskAssigned, TasksTaskDue, TasksBirthdayUpcoming} {
		if !Valid(key) {
			t.Errorf("%s missing from the catalog", key)
		}
		if Domain(key) != "tasks" {
			t.Errorf("%s domain = %s", key, Domain(key))
		}
	}
	if !Valid("tasks") {
		t.Error("the tasks domain must be mutable as a whole")
	}
}
