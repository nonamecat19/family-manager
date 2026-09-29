package recipes

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	recipesv1 "github.com/nnc/family-manager/sdk/go/recipes/v1"
	"github.com/nnc/family-manager/sdk/go/recipes/v1/recipesv1connect"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

const (
	intro       = string(i18n.RecipesIntro)
	pageSize    = 6
	planWindow  = 7 * 24 * time.Hour
	stateSearch = "recipes:search"
)

type client struct {
	rpc recipesv1connect.RecipesServiceClient
}

func Bot(httpClient *http.Client, addr string) bot.Options {
	c := &client{rpc: recipesv1connect.NewRecipesServiceClient(httpClient, addr)}

	return bot.Options{
		Intro: intro,
		Home:  c.home,
		Commands: []bot.Command{
			{Name: "recipes", Args: "[search]", Help: string(i18n.RecipesBrowse), Run: c.browse},
			{Name: "recipe", Args: "<title>", Help: string(i18n.RecipesTitle), Run: c.find},
			{Name: "favorites", Help: string(i18n.RecipesFavorites), Run: c.showFavorites},
			{Name: "plan", Help: string(i18n.RecipesWeek), Run: c.showPlan},
		},
		Callbacks: []bot.Callback{
			{Prefix: "list", Run: c.showList},
			{Prefix: "fav", Run: c.showFavorites},
			{Prefix: "plan", Run: c.showPlan},
			{Prefix: "open", Run: c.open},
			{Prefix: "star", Run: c.toggleFavorite},
			{Prefix: "search", Run: c.askSearch},
		},
		OnText: c.onText,
	}
}

func (c *client) home(ctx context.Context, cc *bot.Context) (string, bot.Keyboard, error) {
	all, err := c.search(ctx, cc, "")
	if err != nil {
		return "", nil, err
	}

	text := bot.Lines(
		bot.Bold(cc.T(i18n.RecipesTitle)),
		"",
		bot.Esc(cc.T(i18n.RecipesShelf, len(all))),
		"",
		bot.Italic(cc.T(i18n.RecipesHint)),
	)

	keyboard := bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.RecipesBrowse), "list:0"), bot.Data(cc.T(i18n.RecipesSearch), "search")),
		bot.Row(bot.Data(cc.T(i18n.RecipesFavorites), "fav"), bot.Data(cc.T(i18n.RecipesWeek), "plan")),
	}
	return text, keyboard, nil
}

func (c *client) browse(ctx context.Context, cc *bot.Context) error {
	query := strings.TrimSpace(cc.Args)
	if query == "" {
		return c.showList(ctx, cc)
	}
	return c.results(ctx, cc, query)
}

func (c *client) showList(ctx context.Context, cc *bot.Context) error {
	page := parsePage(cc.Payload())

	all, err := c.search(ctx, cc, "")
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.RecipesBrowse)), "",
			bot.Italic(cc.T(i18n.RecipesEmpty))), backOnly(cc))
	}

	start := page * pageSize
	if start >= len(all) {
		return cc.Toast(ctx, cc.T(i18n.RecipesEnd))
	}
	end := min(start+pageSize, len(all))

	var buttons []bot.Button
	for _, r := range all[start:end] {
		buttons = append(buttons, bot.Data(bot.Esc(r.GetTitle()), "open:"+r.GetId()))
	}

	keyboard := bot.Keyboard{}.Grid(buttons, 1)
	nav := bot.Row()
	if page > 0 {
		nav = append(nav, bot.Data(cc.T(i18n.FinanceNewer), fmt.Sprintf("list:%d", page-1)))
	}
	if end < len(all) {
		nav = append(nav, bot.Data(cc.T(i18n.FinanceOlder), fmt.Sprintf("list:%d", page+1)))
	}
	if len(nav) > 0 {
		keyboard = append(keyboard, nav)
	}
	keyboard = append(keyboard, bot.Row(bot.Data(cc.T(i18n.Menu), "home")))

	text := bot.Lines(
		bot.Bold(cc.T(i18n.RecipesBrowse)),
		"",
		bot.Esc(cc.T(i18n.RecipesShowing, start+1, end, len(all))),
	)
	return cc.Show(ctx, text, keyboard)
}

func (c *client) find(ctx context.Context, cc *bot.Context) error {
	query := strings.TrimSpace(cc.Args)
	if query == "" {
		return c.askSearch(ctx, cc)
	}
	return c.results(ctx, cc, query)
}

func (c *client) askSearch(ctx context.Context, cc *bot.Context) error {
	if err := cc.SetState(ctx, stateSearch, nil); err != nil {
		return err
	}
	return cc.Show(ctx, bot.Lines(
		bot.Bold(cc.T(i18n.RecipesSearch)),
		"",
		bot.Esc(cc.T(i18n.RecipesAskSearch)),
	), bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Cancel), "home"))})
}

func (c *client) onText(ctx context.Context, cc *bot.Context) error {
	state, ok := cc.State(ctx)
	if !ok || state.Kind != stateSearch {
		return cc.Send(ctx, bot.Italic(cc.T(i18n.OnlyButtons)), nil)
	}
	cc.ClearState(ctx)
	return c.results(ctx, cc, strings.TrimSpace(cc.Args))
}

func (c *client) results(ctx context.Context, cc *bot.Context, query string) error {
	found, err := c.search(ctx, cc, query)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		return cc.Send(ctx, bot.Lines(
			bot.Bold(cc.T(i18n.RecipesSearch)),
			"",
			cc.T(i18n.RecipesNoMatch, bot.Code(query)),
		), bot.Keyboard{
			bot.Row(bot.Data(cc.T(i18n.RecipesBrowse), "list:0")),
			bot.Row(bot.Data(cc.T(i18n.Menu), "home")),
		})
	}

	if len(found) == 1 {
		return c.detail(ctx, cc, found[0].GetId())
	}

	var buttons []bot.Button
	for i, r := range found {
		if i == pageSize {
			break
		}
		buttons = append(buttons, bot.Data(bot.Esc(r.GetTitle()), "open:"+r.GetId()))
	}
	keyboard := bot.Keyboard{}.Grid(buttons, 1)
	keyboard = append(keyboard, bot.Row(bot.Data(cc.T(i18n.Menu), "home")))

	return cc.Send(ctx, bot.Lines(
		bot.Bold(cc.T(i18n.RecipesMatches, bot.Esc(query))),
		"",
		bot.Esc(cc.T(i18n.RecipesFound, len(found))),
	), keyboard)
}

func (c *client) open(ctx context.Context, cc *bot.Context) error {
	return c.detail(ctx, cc, cc.Payload())
}

func (c *client) detail(ctx context.Context, cc *bot.Context, id string) error {
	req := connect.NewRequest(&recipesv1.GetRecipeRequest{RecipeId: id})
	cc.Authorize(req)

	res, err := c.rpc.GetRecipe(ctx, req)
	if err != nil {
		return err
	}
	r := res.Msg.GetRecipe()
	if r == nil {
		return bot.Invalid("%s", cc.T(i18n.RecipesGone))
	}

	lines := []string{"🍽 " + bot.Bold(bot.Esc(r.GetTitle()))}
	if d := strings.TrimSpace(r.GetDescription()); d != "" {
		lines = append(lines, bot.Italic(bot.Esc(d)))
	}

	var facts []string
	if total := r.GetPrepSeconds() + r.GetCookSeconds(); total > 0 {
		facts = append(facts, "⏱ "+duration(total))
	}
	if r.GetServings() > 0 {
		facts = append(facts, cc.T(i18n.RecipesServings, r.GetServings()))
	}
	if r.GetRating() > 0 {
		facts = append(facts, strings.Repeat("★", int(r.GetRating())))
	}
	if len(facts) > 0 {
		lines = append(lines, "", strings.Join(facts, "   "))
	}

	if len(r.GetIngredients()) > 0 {
		lines = append(lines, "", bot.Bold(cc.T(i18n.RecipesIngredients)))
		for _, in := range r.GetIngredients() {
			lines = append(lines, "• "+bot.Esc(strings.TrimSpace(
				strings.Join([]string{in.GetAmount(), in.GetUnit(), in.GetName()}, " "))))
		}
	}
	if len(r.GetSteps()) > 0 {
		lines = append(lines, "", bot.Bold(cc.T(i18n.RecipesSteps)))
		for i, s := range r.GetSteps() {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, bot.Esc(s.GetInstruction())))
		}
	}

	return cc.Show(ctx, strings.Join(lines, "\n"), bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.RecipesFavorite), "star:"+r.GetId()),
			bot.Data(cc.T(i18n.RecipesBrowse), "list:0")),
		bot.Row(bot.Data(cc.T(i18n.Menu), "home")),
	})
}

func (c *client) toggleFavorite(ctx context.Context, cc *bot.Context) error {
	req := connect.NewRequest(&recipesv1.ToggleFavoriteRequest{RecipeId: cc.Payload()})
	cc.Authorize(req)

	res, err := c.rpc.ToggleFavorite(ctx, req)
	if err != nil {
		return err
	}
	if res.Msg.GetIsFavorite() {
		return cc.Toast(ctx, cc.T(i18n.RecipesFavorited))
	}
	return cc.Toast(ctx, cc.T(i18n.RecipesUnfavorited))
}

func (c *client) showFavorites(ctx context.Context, cc *bot.Context) error {
	req := connect.NewRequest(&recipesv1.ListFavoritesRequest{})
	cc.Authorize(req)

	res, err := c.rpc.ListFavorites(ctx, req)
	if err != nil {
		return err
	}
	if len(res.Msg.GetRecipes()) == 0 {
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.RecipesFavorites)), "",
			bot.Italic(cc.T(i18n.RecipesNoFavorites))), bot.Keyboard{
			bot.Row(bot.Data(cc.T(i18n.RecipesBrowse), "list:0")),
			bot.Row(bot.Data(cc.T(i18n.Menu), "home")),
		})
	}

	var buttons []bot.Button
	for _, r := range res.Msg.GetRecipes() {
		buttons = append(buttons, bot.Data("⭐ "+bot.Esc(r.GetTitle()), "open:"+r.GetId()))
	}
	keyboard := bot.Keyboard{}.Grid(buttons, 1)
	keyboard = append(keyboard, bot.Row(bot.Data(cc.T(i18n.Menu), "home")))

	return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.RecipesFavorites)), "",
		bot.Esc(cc.T(i18n.RecipesFavoritesCount, len(res.Msg.GetRecipes())))), keyboard)
}

func (c *client) showPlan(ctx context.Context, cc *bot.Context) error {
	from := time.Now()
	req := connect.NewRequest(&recipesv1.ListMealPlanRequest{
		FromDate: from.Format(time.DateOnly),
		ToDate:   from.Add(planWindow).Format(time.DateOnly),
	})
	cc.Authorize(req)

	res, err := c.rpc.ListMealPlan(ctx, req)
	if err != nil {
		return err
	}
	if len(res.Msg.GetEntries()) == 0 {
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.RecipesWeek)), "",
			bot.Italic(cc.T(i18n.RecipesNoPlan))), backOnly(cc))
	}

	titles, err := c.titles(ctx, cc)
	if err != nil {
		return err
	}

	lines := []string{bot.Bold(cc.T(i18n.RecipesWeek)), ""}
	for _, e := range res.Msg.GetEntries() {
		title := titles[e.GetRecipeId()]
		if title == "" {
			title = "(removed recipe)"
		}
		lines = append(lines, fmt.Sprintf("%s  %s\n      %s",
			bot.Bold(weekday(cc, e.GetDate())), cc.T(slot(e.GetSlot())), bot.Esc(title)))
	}

	return cc.Show(ctx, strings.Join(lines, "\n"), backOnly(cc))
}

func (c *client) search(ctx context.Context, cc *bot.Context, query string) ([]*recipesv1.Recipe, error) {
	req := connect.NewRequest(&recipesv1.ListRecipesRequest{
		Search: query,
		Sort:   recipesv1.RecipeSort_RECIPE_SORT_NEWEST,
	})
	cc.Authorize(req)

	res, err := c.rpc.ListRecipes(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Msg.GetRecipes(), nil
}

func (c *client) titles(ctx context.Context, cc *bot.Context) (map[string]string, error) {
	all, err := c.search(ctx, cc, "")
	if err != nil {
		return nil, err
	}
	titles := make(map[string]string, len(all))
	for _, r := range all {
		titles[r.GetId()] = r.GetTitle()
	}
	return titles, nil
}

func backOnly(cc *bot.Context) bot.Keyboard {
	return bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Menu), "home"))}
}

func parsePage(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return min(value, 200)
}

func duration(seconds int32) string {
	d := time.Duration(seconds) * time.Second
	if d < time.Hour {
		return fmt.Sprintf("%d min", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
}

func weekday(cc *bot.Context, date string) string {
	parsed, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return date
	}
	locale := cc.Locale()
	return fmt.Sprintf("%s %d %s", locale.Weekday(parsed.Weekday()), parsed.Day(),
		locale.ShortMonth(parsed.Month()))
}

func slot(s recipesv1.MealSlot) i18n.Key {
	switch s {
	case recipesv1.MealSlot_MEAL_SLOT_BREAKFAST:
		return i18n.SlotBreakfast
	case recipesv1.MealSlot_MEAL_SLOT_LUNCH:
		return i18n.SlotLunch
	case recipesv1.MealSlot_MEAL_SLOT_DINNER:
		return i18n.SlotDinner
	case recipesv1.MealSlot_MEAL_SLOT_SNACK:
		return i18n.SlotSnack
	case recipesv1.MealSlot_MEAL_SLOT_DESSERT:
		return i18n.SlotDessert
	default:
		return i18n.SlotMeal
	}
}
