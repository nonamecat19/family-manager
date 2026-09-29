package finance

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/sdk/go/finance/v1/financev1connect"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

const intro = string(i18n.FinanceIntro)

type client struct {
	rpc financev1connect.FinanceServiceClient
}

func Bot(httpClient *http.Client, addr string) bot.Options {
	c := &client{rpc: financev1connect.NewFinanceServiceClient(httpClient, addr)}

	return bot.Options{
		Intro: intro,
		Home:  c.home,
		Commands: []bot.Command{
			{Name: "balance", Help: string(i18n.FinanceAccounts), Run: c.showBalance},
			{Name: "spend", Args: "[amount] [category]", Help: string(i18n.FinanceLogSpend), Run: c.spend},
			{Name: "month", Help: string(i18n.FinanceThisMonth), Run: c.showMonth},
			{Name: "last", Help: string(i18n.FinanceRecent), Run: c.showLast},
			{Name: "categories", Help: string(i18n.FinanceCategories), Run: c.showCategories},
		},
		Callbacks: []bot.Callback{
			{Prefix: "bal", Run: c.showBalance},
			{Prefix: "cats", Run: c.showCategories},
			{Prefix: "spend", Run: c.startSpend},
			{Prefix: "pick", Run: c.pickCategory},
			{Prefix: "month", Run: c.showMonth},
			{Prefix: "last", Run: c.showLast},
		},
		OnText: c.onText,
	}
}

func (c *client) spend(ctx context.Context, cc *bot.Context) error {
	fields := strings.Fields(cc.Args)
	if len(fields) < 2 {
		return c.startSpend(ctx, cc)
	}

	minor, err := parseAmount(cc, fields[0])
	if err != nil {
		return err
	}

	category, err := c.findCategory(ctx, cc, fields[1])
	if err != nil {
		return err
	}

	note := strings.TrimSpace(strings.Join(fields[2:], " "))
	return c.logSpend(ctx, cc, category.GetId(), category.GetName(), minor, note)
}

func (c *client) accounts(ctx context.Context, cc *bot.Context) ([]*financev1.Account, error) {
	req := connect.NewRequest(&financev1.ListAccountsRequest{})
	cc.Authorize(req)

	res, err := c.rpc.ListAccounts(ctx, req)
	if err != nil {
		return nil, err
	}
	return append(append([]*financev1.Account{}, res.Msg.GetShared()...), res.Msg.GetPrivateOwn()...), nil
}

func (c *client) categoryTree(ctx context.Context, cc *bot.Context) ([]*financev1.GroupNode, error) {
	req := connect.NewRequest(&financev1.ListCategoryTreeRequest{
		Kind: financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
	})
	cc.Authorize(req)

	res, err := c.rpc.ListCategoryTree(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Msg.GetGroups(), nil
}

func (c *client) summary(
	ctx context.Context, cc *bot.Context, period *financev1.Period,
) (*financev1.GetHomeSummaryResponse, error) {
	req := connect.NewRequest(&financev1.GetHomeSummaryRequest{
		Period: period,
		Kind:   financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
	})
	cc.Authorize(req)

	res, err := c.rpc.GetHomeSummary(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

func (c *client) categoryByID(
	ctx context.Context, cc *bot.Context, id string,
) (*financev1.Category, error) {
	groups, err := c.categoryTree(ctx, cc)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		for _, cat := range g.GetCategories() {
			if cat.GetId() == id {
				return cat, nil
			}
		}
	}
	return nil, bot.Invalid("%s", cc.T(i18n.FinanceGoneCategory))
}

func (c *client) findCategory(
	ctx context.Context, cc *bot.Context, needle string,
) (*financev1.Category, error) {
	groups, err := c.categoryTree(ctx, cc)
	if err != nil {
		return nil, err
	}

	needle = strings.ToLower(needle)
	var partial *financev1.Category
	for _, g := range groups {
		for _, cat := range g.GetCategories() {
			name := strings.ToLower(cat.GetName())
			if name == needle {
				return cat, nil
			}
			if partial == nil && strings.Contains(name, needle) {
				partial = cat
			}
		}
	}
	if partial != nil {
		return partial, nil
	}
	return nil, bot.Invalid("%s", cc.T(i18n.FinanceNoMatch, needle))
}

func parseAmount(cc *bot.Context, raw string) (int64, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || value <= 0 {
		return 0, bot.Invalid("%s", cc.T(i18n.FinanceBadAmount, raw, "12.50"))
	}
	return int64(value*100 + 0.5), nil
}

func monthLabel(cc *bot.Context, t time.Time) string {
	return fmt.Sprintf("%s %d", cc.Locale().Month(t.Month()), t.Year())
}

func shortMonth(cc *bot.Context, back int) string {
	return cc.Locale().ShortMonth(monthTime(back).Month())
}

func parseOffset(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return min(value, 24)
}

func money(m *financev1.Money) string {
	if m == nil {
		return "0.00"
	}
	amount := float64(m.GetAmountMinor()) / 100
	if code := m.GetCurrencyCode(); code != "" {
		return fmt.Sprintf("%.2f %s", amount, code)
	}
	return fmt.Sprintf("%.2f", amount)
}

func monthTime(back int) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -back, 0)
}

func monthAnchor(back int) *financev1.Period {
	return &financev1.Period{
		Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_MONTH,
		Anchor:      monthTime(back).Format(time.DateOnly),
	}
}
