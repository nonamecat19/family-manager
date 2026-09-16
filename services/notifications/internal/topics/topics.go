package topics

import "strings"

type Topic struct {
	Key    string
	Domain string
}

const (
	FamilyMemberJoined    = "family.member.joined"
	FamilyMemberRemoved   = "family.member.removed"
	FinanceBudgetExceeded = "finance.budget.exceeded"
	RecipesRecipeCreated  = "recipes.recipe.created"
	TasksTaskAssigned     = "tasks.task.assigned"
	TasksTaskDue          = "tasks.task.due"
	TasksBirthdayUpcoming = "tasks.birthday.upcoming"
)

var All = []Topic{
	{Key: FamilyMemberJoined, Domain: "family"},
	{Key: FamilyMemberRemoved, Domain: "family"},
	{Key: FinanceBudgetExceeded, Domain: "finance"},
	{Key: RecipesRecipeCreated, Domain: "recipes"},
	{Key: TasksTaskAssigned, Domain: "tasks"},
	{Key: TasksTaskDue, Domain: "tasks"},
	{Key: TasksBirthdayUpcoming, Domain: "tasks"},
}

func Valid(s string) bool {
	for _, t := range All {
		if s == t.Key || s == t.Domain {
			return true
		}
	}
	return false
}

func Domain(key string) string {
	domain, _, _ := strings.Cut(key, ".")
	return domain
}

func Muted(mutes map[string]bool, key string) bool {
	return mutes[key] || mutes[Domain(key)]
}
