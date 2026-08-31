package handler

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// share is spent/total as a ratio, guarding the empty period. Unclamped on purpose: a budget
// share above 1 is what the overspend colour is for.
func share(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

// memberSpending is the per-member split behind the household cards, the member screen and the
// group drill-down. It reads through the same visibility predicate as everything else, so a
// member's private spend appears on their own device and nowhere else.
func (h *Handler) memberSpending(
	ctx context.Context, c caller, hh household, window dayRange, kind *string, accounts []pgtype.UUID,
) ([]*financev1.MemberSpending, error) {
	if accounts == nil {
		accounts = []pgtype.UUID{}
	}
	rows, err := h.q.SumByMember(ctx, db.SumByMemberParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: hh.currency(), Kind: kind, AccountIds: accounts,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum by member")
	}
	totals := map[string]int64{}
	counts := map[string]int32{}
	var grand int64
	for _, r := range rows {
		id := pgconv.UUIDString(r.MemberID)
		totals[id] += r.TotalMinor
		counts[id] += r.TransactionCount
		grand += r.TotalMinor
	}

	members, err := h.q.ListMembers(ctx, db.ListMembersParams{FamilyID: c.familyID})
	if err != nil {
		return nil, h.internal(ctx, err, "list members")
	}
	out := make([]*financev1.MemberSpending, 0, len(members))
	for _, m := range members {
		id := pgconv.UUIDString(m.UserID)
		out = append(out, &financev1.MemberSpending{
			Member:           toProtoMember(m),
			Spent:            money(totals[id], hh.currency()),
			Share:            share(totals[id], grand),
			TransactionCount: counts[id],
		})
	}
	return out, nil
}

// GetHomeSummary is the home screen in a single call. Headline, donut, group rows with their
// budget bars, member chips and template chips all move together when the scope or the period
// changes, so fetching them apart is what lets two of them disagree on screen.
func (h *Handler) GetHomeSummary(
	ctx context.Context, req *connect.Request[financev1.GetHomeSummaryRequest],
) (*connect.Response[financev1.GetHomeSummaryResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	window, err := h.window(msg.GetPeriod(), hh)
	if err != nil {
		return nil, err
	}
	filters, err := h.scopeFilters(msg.GetScope(), nil, nil)
	if err != nil {
		return nil, err
	}
	kind := kindFilter(msg.GetKind())

	// The headline is the household's shared money regardless of scope: switching the donut to
	// one member does not change what the family has, and a headline that moved with the
	// filter would read as "Олена's balance".
	balances, err := h.q.SumFamilyBalances(ctx, db.SumFamilyBalancesParams{
		FamilyID: c.familyID, CurrencyCode: hh.currency(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum family balances")
	}

	groupSums, err := h.q.SumByGroup(ctx, db.SumByGroupParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: hh.currency(), Kind: kind,
		MemberIds: filters.members, AccountIds: filters.accounts,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum by group")
	}
	// A transaction with no category still spent money, so SumByGroup returns it under a NULL
	// group. It is counted in the period total AND given a row of its own below: a total the
	// rows do not add up to is the disagreement this bucket exists to prevent.
	sums := map[string]int64{}
	var periodTotal, uncategorised int64
	for _, r := range groupSums {
		if !r.GroupID.Valid {
			uncategorised = r.TotalMinor
			periodTotal += r.TotalMinor
			continue
		}
		sums[pgconv.UUIDString(r.GroupID)] = r.TotalMinor
		periodTotal += r.TotalMinor
	}

	groups, err := h.q.ListCategoryGroups(ctx, db.ListCategoryGroupsParams{
		FamilyID: c.familyID, Kind: kindFilter(msg.GetKind()),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list category groups")
	}
	categories, err := h.q.ListCategories(ctx, db.ListCategoriesParams{FamilyID: c.familyID})
	if err != nil {
		return nil, h.internal(ctx, err, "list categories")
	}
	categoryCount := map[string]int32{}
	for _, cat := range categories {
		categoryCount[pgconv.UUIDString(cat.GroupID)]++
	}

	budgets, err := h.q.ListBudgets(ctx, db.ListBudgetsParams{
		FamilyID: c.familyID, TargetKind: strPtr(targetGroup),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list budgets")
	}
	budgetByGroup := map[string]db.Budget{}
	for _, b := range budgets {
		budgetByGroup[pgconv.UUIDString(b.GroupID)] = b
	}

	contributors, err := h.contributorsByGroup(ctx, c, hh, window, kind, filters)
	if err != nil {
		return nil, err
	}

	out := &financev1.GetHomeSummaryResponse{
		HeadlineBalance: money(balances.SharedBalanceMinor, hh.currency()),
		PeriodTotal:     money(periodTotal, hh.currency()),
		GroupCount:      int32(len(groups)),
		Window:          window.proto(),
	}
	for _, g := range groups {
		id := pgconv.UUIDString(g.ID)
		amount := sums[id]
		row := &financev1.GroupRow{
			GroupId: id, Name: g.Name, Icon: g.Icon, ColorStep: g.ColorStep,
			Amount: money(amount, hh.currency()), Share: share(amount, periodTotal),
			CategoryCount:        categoryCount[id],
			ContributorMemberIds: contributors[id],
		}
		if b, ok := budgetByGroup[id]; ok {
			status, err := h.budgetStatus(ctx, c, hh, b, h.today(hh))
			if err != nil {
				return nil, err
			}
			row.Budget = status
		}
		out.Groups = append(out.Groups, row)
		// A group with nothing spent in it is a row but not a slice: a zero-width wedge is
		// noise in a donut and a legend entry the user cannot tap.
		if amount > 0 {
			out.Slices = append(out.Slices, &financev1.DonutSlice{
				GroupId: id, Name: g.Name, Icon: g.Icon, ColorStep: g.ColorStep,
				Amount: money(amount, hh.currency()), Share: share(amount, periodTotal),
			})
		}
	}

	// The uncategorised bucket is last, and only when there is something in it. Its group id is
	// empty, which is what tells the app to name it "Без категорії" — the server has no name to
	// give a group that does not exist.
	if uncategorised > 0 {
		out.Groups = append(out.Groups, &financev1.GroupRow{
			Amount: money(uncategorised, hh.currency()),
			Share:  share(uncategorised, periodTotal),
		})
		out.Slices = append(out.Slices, &financev1.DonutSlice{
			Amount: money(uncategorised, hh.currency()),
			Share:  share(uncategorised, periodTotal),
		})
	}

	templates, err := h.q.ListTemplates(ctx, db.ListTemplatesParams{
		FamilyID: c.familyID, OwnerUserID: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list templates")
	}
	for _, t := range templates {
		out.Templates = append(out.Templates, toProtoTemplate(t))
	}

	members, err := h.q.ListMembers(ctx, db.ListMembersParams{FamilyID: c.familyID})
	if err != nil {
		return nil, h.internal(ctx, err, "list members")
	}
	for _, m := range members {
		out.Members = append(out.Members, &financev1.MemberChip{
			MemberId:        pgconv.UUIDString(m.UserID),
			DisplayName:     m.DisplayName,
			Initial:         m.Initial,
			AvatarColorStep: m.AvatarColorStep,
		})
	}
	return connect.NewResponse(out), nil
}

// contributorsByGroup answers the "· Сергій" caption: who spent in this group this period. It
// is not an ownership claim on the group — nothing in the model attributes a group to a member.
func (h *Handler) contributorsByGroup(
	ctx context.Context, c caller, hh household, window dayRange, kind *string, filters scopeFilter,
) (map[string][]string, error) {
	rows, err := h.q.SumByMember(ctx, db.SumByMemberParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: hh.currency(), Kind: kind, AccountIds: filters.accounts,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum by member")
	}
	seen := map[string]map[string]bool{}
	for _, r := range rows {
		if r.TotalMinor == 0 || !r.GroupID.Valid {
			continue
		}
		group := pgconv.UUIDString(r.GroupID)
		if seen[group] == nil {
			seen[group] = map[string]bool{}
		}
		seen[group][pgconv.UUIDString(r.MemberID)] = true
	}
	out := make(map[string][]string, len(seen))
	for group, members := range seen {
		ids := make([]string, 0, len(members))
		for id := range members {
			ids = append(ids, id)
		}
		// Sorted so the caption is stable between reads; map iteration order is not.
		sort.Strings(ids)
		out[group] = ids
	}
	return out, nil
}

func (h *Handler) GetGroupBreakdown(
	ctx context.Context, req *connect.Request[financev1.GetGroupBreakdownRequest],
) (*connect.Response[financev1.GetGroupBreakdownResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	groupID, err := requireUUID("group_id", msg.GetGroupId())
	if err != nil {
		return nil, err
	}
	window, err := h.window(msg.GetPeriod(), hh)
	if err != nil {
		return nil, err
	}
	filters, err := h.scopeFilters(msg.GetScope(), nil, nil)
	if err != nil {
		return nil, err
	}
	kind := kindFilter(msg.GetKind())

	group, err := h.q.GetCategoryGroup(ctx, db.GetCategoryGroupParams{
		ID: groupID, FamilyID: c.familyID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("category group")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get category group")
	}

	sums, err := h.q.SumByCategory(ctx, db.SumByCategoryParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: hh.currency(), Kind: kind, GroupID: groupID,
		MemberIds: filters.members,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum by category")
	}
	amounts := map[string]int64{}
	var total int64
	for _, r := range sums {
		amounts[pgconv.UUIDString(r.CategoryID)] = r.TotalMinor
		total += r.TotalMinor
	}

	categories, err := h.q.ListCategories(ctx, db.ListCategoriesParams{
		FamilyID: c.familyID, GroupID: groupID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list categories")
	}

	out := &financev1.GetGroupBreakdownResponse{
		Group: toProtoGroup(group),
		Total: money(total, hh.currency()),
	}
	for _, cat := range categories {
		id := pgconv.UUIDString(cat.ID)
		out.Categories = append(out.Categories, &financev1.CategorySlice{
			CategoryId: id, Name: cat.Name, Icon: cat.Icon,
			Amount: money(amounts[id], hh.currency()),
			Share:  share(amounts[id], total),
		})
	}

	budgets, err := h.q.ListBudgets(ctx, db.ListBudgetsParams{
		FamilyID: c.familyID, TargetKind: strPtr(targetGroup),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list budgets")
	}
	for _, b := range budgets {
		if pgconv.UUIDString(b.GroupID) != pgconv.UUIDString(groupID) {
			continue
		}
		status, err := h.budgetStatus(ctx, c, hh, b, h.today(hh))
		if err != nil {
			return nil, err
		}
		out.Budget = status
		break
	}

	members, err := h.memberSpending(ctx, c, hh, window, kind, filters.accounts)
	if err != nil {
		return nil, err
	}
	out.Members = members
	return connect.NewResponse(out), nil
}

func (h *Handler) GetMemberBreakdown(
	ctx context.Context, req *connect.Request[financev1.GetMemberBreakdownRequest],
) (*connect.Response[financev1.GetMemberBreakdownResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	window, err := h.window(req.Msg.GetPeriod(), hh)
	if err != nil {
		return nil, err
	}
	kind := kindFilter(req.Msg.GetKind())

	rows, err := h.q.SumByMember(ctx, db.SumByMemberParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: hh.currency(), Kind: kind, AccountIds: []pgtype.UUID{},
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum by member")
	}

	var total int64
	perGroup := map[string]map[string]int64{}
	groupTotals := map[string]int64{}
	for _, r := range rows {
		total += r.TotalMinor
		if !r.GroupID.Valid {
			continue
		}
		g := pgconv.UUIDString(r.GroupID)
		if perGroup[g] == nil {
			perGroup[g] = map[string]int64{}
		}
		perGroup[g][pgconv.UUIDString(r.MemberID)] += r.TotalMinor
		groupTotals[g] += r.TotalMinor
	}

	members, err := h.memberSpending(ctx, c, hh, window, kind, nil)
	if err != nil {
		return nil, err
	}

	groups, err := h.q.ListCategoryGroups(ctx, db.ListCategoryGroupsParams{FamilyID: c.familyID})
	if err != nil {
		return nil, h.internal(ctx, err, "list category groups")
	}
	out := &financev1.GetMemberBreakdownResponse{
		Total:   money(total, hh.currency()),
		Members: members,
	}
	for _, g := range groups {
		id := pgconv.UUIDString(g.ID)
		if groupTotals[id] == 0 {
			continue
		}
		split := &financev1.GroupMemberSplit{
			GroupId: id, Name: g.Name, Icon: g.Icon, ColorStep: g.ColorStep,
			Total: money(groupTotals[id], hh.currency()),
		}
		memberIDs := make([]string, 0, len(perGroup[id]))
		for memberID := range perGroup[id] {
			memberIDs = append(memberIDs, memberID)
		}
		sort.Strings(memberIDs)
		for _, memberID := range memberIDs {
			split.Members = append(split.Members, &financev1.MemberAmount{
				MemberId: memberID,
				Amount:   money(perGroup[id][memberID], hh.currency()),
			})
		}
		out.Groups = append(out.Groups, split)
	}

	insights, err := h.insights(ctx, c, hh, window, kind, 3)
	if err != nil {
		return nil, err
	}
	out.Insights = insights
	return connect.NewResponse(out), nil
}

// GetSpendingSeries buckets in Go from one daily-totals query rather than issuing one query
// per bucket: seven round trips to draw seven bars is the shape that makes the charts screen
// feel slow on a phone.
func (h *Handler) GetSpendingSeries(
	ctx context.Context, req *connect.Request[financev1.GetSpendingSeriesRequest],
) (*connect.Response[financev1.GetSpendingSeriesResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	if msg.GetGranularity() == financev1.PeriodGranularity_PERIOD_GRANULARITY_CUSTOM {
		return nil, invalid("a series needs a fixed bucket width, not a custom period")
	}
	count := int(msg.GetBucketCount())
	if count > maxSeriesBuckets {
		return nil, invalid("bucket_count must be at most %d", maxSeriesBuckets)
	}
	buckets := seriesBuckets(msg.GetGranularity(), count, h.now(), hh.loc, hh.settings.WeekStartsOn)
	if len(buckets) == 0 {
		return connect.NewResponse(&financev1.GetSpendingSeriesResponse{}), nil
	}
	filters, err := h.scopeFilters(msg.GetScope(), nil, nil)
	if err != nil {
		return nil, err
	}

	rows, err := h.q.SumDailyTotals(ctx, db.SumDailyTotalsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(buckets[0].from), ToDate: pgDate(buckets[len(buckets)-1].to),
		CurrencyCode: hh.currency(), Kind: kindFilter(msg.GetKind()),
		MemberIds: filters.members, AccountIds: filters.accounts,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum daily totals")
	}

	labels, steps, err := h.segmentLabels(ctx, c, msg.GetStackedBy())
	if err != nil {
		return nil, err
	}

	totals := make([]int64, len(buckets))
	segments := make([]map[string]int64, len(buckets))
	for i := range segments {
		segments[i] = map[string]int64{}
	}
	for _, r := range rows {
		day := r.OccurredOn.Time
		for i, b := range buckets {
			if !b.contains(day) {
				continue
			}
			totals[i] += r.TotalMinor
			switch msg.GetStackedBy() {
			case financev1.SeriesStacking_SERIES_STACKING_MEMBER:
				segments[i][pgconv.UUIDString(r.MemberID)] += r.TotalMinor
			case financev1.SeriesStacking_SERIES_STACKING_GROUP:
				if r.GroupID.Valid {
					segments[i][pgconv.UUIDString(r.GroupID)] += r.TotalMinor
				}
			}
			break
		}
	}

	out := &financev1.GetSpendingSeriesResponse{
		// The current bucket is always the last: seriesBuckets counts back from today, so the
		// chart can outline it without a second request.
		CurrentBucketIndex: int32(len(buckets) - 1),
	}
	for i, b := range buckets {
		bucket := &financev1.SeriesBucket{
			Label: bucketLabel(msg.GetGranularity(), b),
			Start: b.fromString(), End: b.toString(),
			Total: money(totals[i], hh.currency()),
		}
		keys := make([]string, 0, len(segments[i]))
		for key := range segments[i] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			bucket.Segments = append(bucket.Segments, &financev1.SeriesSegment{
				Key: key, Label: labels[key], ColorStep: steps[key],
				Amount: money(segments[i][key], hh.currency()),
			})
		}
		out.Buckets = append(out.Buckets, bucket)
	}
	return connect.NewResponse(out), nil
}

// maxSeriesBuckets bounds the chart's x-axis. The design draws 7; anything past a couple of
// years of daily buckets is a scan of the whole table dressed as a chart.
const maxSeriesBuckets = 60

// segmentLabels resolves a stacked series' keys to the names and ramp steps the legend draws,
// so the client does not have to hold the member list and the taxonomy to render a chart.
func (h *Handler) segmentLabels(
	ctx context.Context, c caller, stacking financev1.SeriesStacking,
) (map[string]string, map[string]int32, error) {
	labels := map[string]string{}
	steps := map[string]int32{}
	switch stacking {
	case financev1.SeriesStacking_SERIES_STACKING_MEMBER:
		members, err := h.q.ListMembers(ctx, db.ListMembersParams{FamilyID: c.familyID})
		if err != nil {
			return nil, nil, h.internal(ctx, err, "list members")
		}
		for _, m := range members {
			id := pgconv.UUIDString(m.UserID)
			labels[id] = m.DisplayName
			steps[id] = m.AvatarColorStep
		}
	case financev1.SeriesStacking_SERIES_STACKING_GROUP:
		groups, err := h.q.ListCategoryGroups(ctx, db.ListCategoryGroupsParams{
			FamilyID: c.familyID, IncludeArchived: true,
		})
		if err != nil {
			return nil, nil, h.internal(ctx, err, "list category groups")
		}
		for _, g := range groups {
			id := pgconv.UUIDString(g.ID)
			labels[id] = g.Name
			steps[id] = g.ColorStep
		}
	}
	return labels, steps, nil
}

func (h *Handler) ListInsights(
	ctx context.Context, req *connect.Request[financev1.ListInsightsRequest],
) (*connect.Response[financev1.ListInsightsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	window, err := h.window(req.Msg.GetPeriod(), hh)
	if err != nil {
		return nil, err
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 || limit > maxInsights {
		limit = maxInsights
	}
	insights, err := h.insights(ctx, c, hh, window,
		kindFilter(financev1.TransactionKind_TRANSACTION_KIND_EXPENSE), limit)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.ListInsightsResponse{Insights: insights}), nil
}

const (
	maxInsights = 5
	// spikeRatio and spikeFloor are the thresholds an insight has to clear. The floor exists
	// because a doubling from ₴20 to ₴40 is arithmetically identical to one from ₴2,000 to
	// ₴4,000 and worth telling nobody. Both are guesses that the design never specified.
	spikeRatio = 1.5
	spikeFloor = 50_000
)

// insights compares this window's per-member, per-group spend with the same-length window
// immediately before it, and composes the sentence server-side so app, widget and any future
// notification say the same thing rather than three nearly identical ones.
func (h *Handler) insights(
	ctx context.Context, c caller, hh household, window dayRange, kind *string, limit int,
) ([]*financev1.Insight, error) {
	days := int(window.to.Sub(window.from).Hours()/24) + 1
	previous := dayRange{
		from: window.from.AddDate(0, 0, -days),
		to:   window.from.AddDate(0, 0, -1),
	}

	current, err := h.q.SumByMember(ctx, db.SumByMemberParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: hh.currency(), Kind: kind, AccountIds: []pgtype.UUID{},
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum by member")
	}
	prior, err := h.q.SumByMember(ctx, db.SumByMemberParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(previous.from), ToDate: pgDate(previous.to),
		CurrencyCode: hh.currency(), Kind: kind, AccountIds: []pgtype.UUID{},
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum by member")
	}

	key := func(member, group pgtype.UUID) string {
		return pgconv.UUIDString(member) + "|" + pgconv.UUIDString(group)
	}
	before := map[string]int64{}
	for _, r := range prior {
		before[key(r.MemberID, r.GroupID)] = r.TotalMinor
	}

	names, err := h.memberNames(ctx, c)
	if err != nil {
		return nil, err
	}
	groups, err := h.q.ListCategoryGroups(ctx, db.ListCategoryGroupsParams{
		FamilyID: c.familyID, IncludeArchived: true,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list category groups")
	}
	groupNames := map[string]string{}
	for _, g := range groups {
		groupNames[pgconv.UUIDString(g.ID)] = g.Name
	}

	var out []*financev1.Insight
	// Sorted by key so the same window always produces the same list: an insight card that
	// reshuffles between two identical reads reads as a bug.
	sort.Slice(current, func(i, j int) bool {
		return key(current[i].MemberID, current[i].GroupID) < key(current[j].MemberID, current[j].GroupID)
	})
	for _, r := range current {
		if !r.GroupID.Valid || r.TotalMinor < spikeFloor {
			continue
		}
		k := key(r.MemberID, r.GroupID)
		was := before[k]
		if was <= 0 || float64(r.TotalMinor) < float64(was)*spikeRatio {
			continue
		}
		memberID := pgconv.UUIDString(r.MemberID)
		groupID := pgconv.UUIDString(r.GroupID)
		out = append(out, &financev1.Insight{
			Id:              k,
			Kind:            financev1.InsightKind_INSIGHT_KIND_SPEND_SPIKE,
			Icon:            "trend-up",
			Title:           fmt.Sprintf("%s — %s", names[memberID], groupNames[groupID]),
			Body:            fmt.Sprintf("%d vs %d", r.TotalMinor, was),
			SubjectMemberId: memberID,
			SubjectGroupId:  groupID,
			Current:         money(r.TotalMinor, hh.currency()),
			Previous:        money(was, hh.currency()),
			Ratio:           share(r.TotalMinor, was),
		})
		if len(out) == limit {
			break
		}
	}
	return out, nil
}
