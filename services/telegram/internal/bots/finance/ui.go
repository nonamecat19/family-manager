package finance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

const (
	stateAwaitAmount = "finance:amount"
	topGroups        = 6
	pageSize         = 8
)

func (c *client) home(ctx context.Context, cc *bot.Context) (string, bot.Keyboard, error) {
	res, err := c.summary(ctx, cc, monthAnchor(0))
	if err != nil {
		return "", nil, err
	}

	text := bot.Lines(
		bot.Bold(cc.T(i18n.FinanceTitle)),
		"",
		cc.T(i18n.FinanceBalance)+"   "+bot.Bold(money(res.GetHeadlineBalance())),
		cc.T(i18n.FinanceThisMonth)+"   "+bot.Bold(money(res.GetPeriodTotal())),
		"",
		bot.Italic(cc.T(i18n.FinancePick)),
	)

	keyboard := bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.FinanceLogSpend), "spend"), bot.Data(cc.T(i18n.FinanceAccounts), "bal")),
		bot.Row(bot.Data("📊 "+cc.T(i18n.FinanceThisMonth), "month:0"), bot.Data(cc.T(i18n.FinanceRecent), "last:0")),
		bot.Row(bot.Data(cc.T(i18n.FinanceCategories), "cats")),
	}
	return text, keyboard, nil
}

func (c *client) showBalance(ctx context.Context, cc *bot.Context) error {
	accounts, err := c.accounts(ctx, cc)
	if err != nil {
		return err
	}

	if len(accounts) == 0 {
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.FinanceAccounts)), "",
			bot.Esc(cc.T(i18n.FinanceNoAccounts))), backOnly(cc))
	}

	lines := []string{bot.Bold(cc.T(i18n.FinanceAccounts)), ""}
	for _, a := range accounts {
		lines = append(lines, fmt.Sprintf("%s  %s\n      %s",
			icon(a.GetKind()), bot.Bold(bot.Esc(a.GetName())), money(a.GetBalance())))
	}

	return cc.Show(ctx, strings.Join(lines, "\n"), bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.FinanceLogSpend), "spend")),
		bot.Row(bot.Data(cc.T(i18n.Back), "home")),
	})
}

func (c *client) showCategories(ctx context.Context, cc *bot.Context) error {
	groups, err := c.categoryTree(ctx, cc)
	if err != nil {
		return err
	}

	lines := []string{bot.Bold(cc.T(i18n.FinanceCategories)), ""}
	for _, g := range groups {
		names := make([]string, 0, len(g.GetCategories()))
		for _, cat := range g.GetCategories() {
			names = append(names, bot.Esc(cat.GetName()))
		}
		if len(names) == 0 {
			continue
		}
		lines = append(lines, bot.Bold(bot.Esc(g.GetGroup().GetName()))+"\n      "+strings.Join(names, " · "))
	}
	if len(lines) == 2 {
		lines = append(lines, bot.Italic(cc.T(i18n.FinanceNoCategories)))
	}

	return cc.Show(ctx, strings.Join(lines, "\n"), backOnly(cc))
}

func (c *client) startSpend(ctx context.Context, cc *bot.Context) error {
	groups, err := c.categoryTree(ctx, cc)
	if err != nil {
		return err
	}

	var buttons []bot.Button
	for _, g := range groups {
		for _, cat := range g.GetCategories() {
			buttons = append(buttons, bot.Data(bot.Esc(cat.GetName()), "pick:"+cat.GetId()))
		}
	}
	if len(buttons) == 0 {
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.FinanceLogSpend)), "",
			bot.Esc(cc.T(i18n.FinanceNoCategories))), backOnly(cc))
	}

	keyboard := bot.Keyboard{}.Grid(buttons, 2)
	keyboard = append(keyboard, bot.Row(bot.Data(cc.T(i18n.Back), "home")))

	return cc.Show(ctx, bot.Lines(
		bot.Bold(cc.T(i18n.FinanceLogSpend)),
		"",
		bot.Esc(cc.T(i18n.FinanceWhichCategory)),
	), keyboard)
}

func (c *client) pickCategory(ctx context.Context, cc *bot.Context) error {
	categoryID := cc.Payload()
	category, err := c.categoryByID(ctx, cc, categoryID)
	if err != nil {
		return err
	}

	if err := cc.SetState(ctx, stateAwaitAmount, map[string]string{
		"category_id": categoryID,
		"category":    category.GetName(),
	}); err != nil {
		return err
	}

	return cc.Show(ctx, bot.Lines(
		bot.Bold("➕ "+bot.Esc(category.GetName())),
		"",
		cc.T(i18n.FinanceSendAmount, bot.Code("250"), bot.Code("12.50")),
	), bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Cancel), "home"))})
}

func (c *client) onText(ctx context.Context, cc *bot.Context) error {
	state, ok := cc.State(ctx)
	if !ok || state.Kind != stateAwaitAmount {
		return cc.Send(ctx, bot.Italic(cc.T(i18n.OnlyButtons)), nil)
	}

	fields := strings.Fields(cc.Args)
	if len(fields) == 0 {
		return bot.Invalid("%s", cc.T(i18n.FinanceBadAmount, "", "250"))
	}
	minor, err := parseAmount(cc, fields[0])
	if err != nil {
		return err
	}

	note := strings.TrimSpace(strings.Join(fields[1:], " "))
	cc.ClearState(ctx)

	return c.logSpend(ctx, cc, state.Payload["category_id"], state.Payload["category"], minor, note)
}

func (c *client) logSpend(
	ctx context.Context, cc *bot.Context, categoryID, categoryName string, minor int64, note string,
) error {
	accounts, err := c.accounts(ctx, cc)
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return bot.Invalid("%s", cc.T(i18n.FinanceNoAccounts))
	}
	account := accounts[0]

	req := connect.NewRequest(&financev1.CreateTransactionRequest{
		Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		AccountId:  account.GetId(),
		CategoryId: categoryID,
		Amount: &financev1.Money{
			AmountMinor:  minor,
			CurrencyCode: account.GetCurrencyCode(),
		},
		OccurredOn: time.Now().Format(time.DateOnly),
		Note:       note,
	})
	cc.Authorize(req)

	res, err := c.rpc.CreateTransaction(ctx, req)
	if err != nil {
		return err
	}

	lines := []string{
		bot.Bold(cc.T(i18n.FinanceLogged)),
		"",
		money(res.Msg.GetTransaction().GetAmount()) + " · " + bot.Esc(categoryName),
		bot.Italic(cc.T(i18n.FinanceFrom, bot.Esc(account.GetName()))),
	}
	if note != "" {
		lines = append(lines, bot.Italic(bot.Esc(note)))
	}
	for _, b := range res.Msg.GetAffectedBudgets() {
		if b.GetExceeded() {
			lines = append(lines, "",
				cc.T(i18n.FinanceOverBudget, money(b.GetBudget().GetLimit()), money(b.GetSpent())))
			continue
		}
		lines = append(lines, "",
			cc.T(i18n.FinanceLeftBudget, money(b.GetRemaining()), money(b.GetBudget().GetLimit())))
	}

	return cc.Send(ctx, strings.Join(lines, "\n"), bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.FinanceAnother), "spend"),
			bot.Data("📊 "+cc.T(i18n.FinanceThisMonth), "month:0")),
		bot.Row(bot.Data(cc.T(i18n.Menu), "home")),
	})
}

func (c *client) showMonth(ctx context.Context, cc *bot.Context) error {
	back := parseOffset(cc.Payload())
	res, err := c.summary(ctx, cc, monthAnchor(back))
	if err != nil {
		return err
	}

	anchor := monthTime(back)
	lines := []string{
		bot.Bold("📊 " + monthLabel(cc, anchor)),
		"",
		cc.T(i18n.FinanceSpent) + "   " + bot.Bold(money(res.GetPeriodTotal())),
		cc.T(i18n.FinanceBalance) + "   " + bot.Bold(money(res.GetHeadlineBalance())),
	}

	groups := res.GetGroups()
	if len(groups) == 0 {
		lines = append(lines, "", bot.Italic(cc.T(i18n.FinanceNothingLogged)))
	} else {
		lines = append(lines, "", bot.Bold(cc.T(i18n.FinanceWhereItWent)))
		for i, g := range groups {
			if i == topGroups {
				break
			}
			lines = append(lines, fmt.Sprintf("%s  %s\n      %s  %s",
				bar(g.GetShare()), bot.Bold(bot.Esc(g.GetName())),
				money(g.GetAmount()), fmt.Sprintf("%.0f%%", g.GetShare()*100)))
		}
	}

	return cc.Show(ctx, strings.Join(lines, "\n"), bot.Keyboard{
		bot.Row(
			bot.Data("‹ "+shortMonth(cc, back+1), fmt.Sprintf("month:%d", back+1)),
			bot.Data(cc.T(i18n.FinanceRecent), "last:0"),
			nextMonthButton(cc, back),
		),
		bot.Row(bot.Data(cc.T(i18n.Menu), "home")),
	})
}

func (c *client) showLast(ctx context.Context, cc *bot.Context) error {
	page := parseOffset(cc.Payload())

	req := connect.NewRequest(&financev1.ListTransactionsRequest{
		Period:   monthAnchor(0),
		PageSize: pageSize * int32(page+1),
	})
	cc.Authorize(req)

	res, err := c.rpc.ListTransactions(ctx, req)
	if err != nil {
		return err
	}

	var flat []*financev1.Transaction
	dates := map[string]string{}
	for _, day := range res.Msg.GetDays() {
		for _, t := range day.GetTransactions() {
			flat = append(flat, t)
			dates[t.GetId()] = day.GetDate()
		}
	}

	start := page * pageSize
	if start >= len(flat) {
		if start > 0 {
			return cc.Toast(ctx, cc.T(i18n.FinanceNothingFurther))
		}
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.FinanceRecent)), "",
			bot.Italic(cc.T(i18n.FinanceNothingLogged))), bot.Keyboard{
			bot.Row(bot.Data(cc.T(i18n.FinanceLogSpend), "spend")),
			bot.Row(bot.Data(cc.T(i18n.Menu), "home")),
		})
	}
	end := min(start+pageSize, len(flat))

	lines := []string{bot.Bold(cc.T(i18n.FinanceRecent)), ""}
	for _, t := range flat[start:end] {
		label := t.GetNote()
		if label == "" {
			label = t.GetMerchant()
		}
		if label == "" {
			label = "—"
		}
		lines = append(lines, fmt.Sprintf("%s   %s\n      %s",
			bot.Bold(money(t.GetAmount())), bot.Esc(label), bot.Italic(dates[t.GetId()])))
	}
	lines = append(lines, "", cc.T(i18n.FinancePeriodTotal)+"   "+bot.Bold(money(res.Msg.GetPeriodTotal())))

	nav := bot.Row()
	if page > 0 {
		nav = append(nav, bot.Data(cc.T(i18n.FinanceNewer), fmt.Sprintf("last:%d", page-1)))
	}
	if end < len(flat) {
		nav = append(nav, bot.Data(cc.T(i18n.FinanceOlder), fmt.Sprintf("last:%d", page+1)))
	}

	keyboard := bot.Keyboard{}
	if len(nav) > 0 {
		keyboard = append(keyboard, nav)
	}
	keyboard = append(keyboard, bot.Row(bot.Data(cc.T(i18n.Menu), "home")))

	return cc.Show(ctx, strings.Join(lines, "\n"), keyboard)
}

func backOnly(cc *bot.Context) bot.Keyboard {
	return bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Menu), "home"))}
}

func nextMonthButton(cc *bot.Context, back int) bot.Button {
	if back == 0 {
		return bot.Data(" ", "noop")
	}
	return bot.Data(shortMonth(cc, back-1)+" ›", fmt.Sprintf("month:%d", back-1))
}

func bar(share float64) string {
	filled := int(share*5 + 0.5)
	return strings.Repeat("▰", max(filled, 1)) + strings.Repeat("▱", max(5-filled, 0))
}

func icon(kind financev1.AccountKind) string {
	switch kind {
	case financev1.AccountKind_ACCOUNT_KIND_CASH:
		return "💵"
	case financev1.AccountKind_ACCOUNT_KIND_CARD:
		return "💳"
	case financev1.AccountKind_ACCOUNT_KIND_BANK:
		return "🏦"
	case financev1.AccountKind_ACCOUNT_KIND_SAVINGS:
		return "🐖"
	case financev1.AccountKind_ACCOUNT_KIND_CRYPTO:
		return "🪙"
	case financev1.AccountKind_ACCOUNT_KIND_DEBT:
		return "📕"
	default:
		return "•"
	}
}
