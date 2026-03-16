import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/family/domain/family_summary_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

/// Screen displaying the family spending summary dashboard.
///
/// Shows the family total, per-person breakdown, and per-category
/// breakdown for the selected month. Supports month navigation
/// and pull-to-refresh.
class FamilySummaryScreen extends ConsumerStatefulWidget {
  /// Creates a [FamilySummaryScreen].
  const FamilySummaryScreen({super.key});

  @override
  ConsumerState<FamilySummaryScreen> createState() =>
      _FamilySummaryScreenState();
}

class _FamilySummaryScreenState
    extends ConsumerState<FamilySummaryScreen> {
  late DateTime _selectedMonth;

  @override
  void initState() {
    super.initState();
    final now = DateTime.now();
    _selectedMonth = DateTime(now.year, now.month);
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadData());
  }

  Future<void> _loadData() async {
    final formatted = DateFormat('yyyy-MM').format(_selectedMonth);
    await ref
        .read(familySummaryStateProvider.notifier)
        .loadSummary(month: formatted);
  }

  void _previousMonth() {
    setState(() {
      _selectedMonth = DateTime(
        _selectedMonth.year,
        _selectedMonth.month - 1,
      );
    });
    _loadData();
  }

  void _nextMonth() {
    setState(() {
      _selectedMonth = DateTime(
        _selectedMonth.year,
        _selectedMonth.month + 1,
      );
    });
    _loadData();
  }

  bool get _isCurrentMonth {
    final now = DateTime.now();
    return _selectedMonth.year == now.year &&
        _selectedMonth.month == now.month;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final summaryState = ref.watch(familySummaryStateProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Family Summary')),
      body: Column(
        children: [
          // Month selector.
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                IconButton(
                  icon: Icon(
                    Icons.chevron_left,
                    color: colorScheme.primary,
                    size: 24,
                  ),
                  onPressed: _previousMonth,
                ),
                Text(
                  DateFormat.yMMMM().format(_selectedMonth),
                  style: theme.textTheme.titleMedium,
                ),
                IconButton(
                  icon: Icon(
                    Icons.chevron_right,
                    color: _isCurrentMonth
                        ? colorScheme.onSurface.withAlpha(97)
                        : colorScheme.primary,
                    size: 24,
                  ),
                  onPressed: _isCurrentMonth ? null : _nextMonth,
                ),
              ],
            ),
          ),
          // Summary content.
          Expanded(
            child: switch (summaryState) {
              FamilySummaryInitial() ||
              FamilySummaryLoading() =>
                const Center(child: CircularProgressIndicator()),
              FamilySummaryError() => RefreshIndicator(
                  onRefresh: _loadData,
                  child: SingleChildScrollView(
                    physics: const AlwaysScrollableScrollPhysics(),
                    child: SizedBox(
                      height: MediaQuery.of(context).size.height * 0.5,
                      child: Center(
                        child: Text(
                          "Couldn't load family summary. "
                          'Pull down to try again.',
                          style: theme.textTheme.bodyMedium,
                          textAlign: TextAlign.center,
                        ),
                      ),
                    ),
                  ),
                ),
              FamilySummaryLoaded(:final summary) =>
                summary.totalCents == 0 && summary.byPerson.isEmpty
                    ? Center(
                        child: Padding(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 32,
                          ),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(
                                Icons.analytics,
                                size: 48,
                                color: colorScheme.onSurfaceVariant,
                              ),
                              const SizedBox(height: 16),
                              Text(
                                'No spending data',
                                style: theme.textTheme.titleLarge,
                              ),
                              const SizedBox(height: 8),
                              Text(
                                'No family spending this month. '
                                'Totals will appear here as '
                                'members log expenses.',
                                style: theme.textTheme.bodyLarge,
                                textAlign: TextAlign.center,
                              ),
                            ],
                          ),
                        ),
                      )
                    : RefreshIndicator(
                        onRefresh: _loadData,
                        child: SingleChildScrollView(
                          physics:
                              const AlwaysScrollableScrollPhysics(),
                          child: Column(
                            crossAxisAlignment:
                                CrossAxisAlignment.start,
                            children: [
                              // Family total.
                              Padding(
                                padding: const EdgeInsets.all(16),
                                child: Column(
                                  crossAxisAlignment:
                                      CrossAxisAlignment.start,
                                  children: [
                                    Text(
                                      'Family Total',
                                      style:
                                          theme.textTheme.bodyLarge,
                                    ),
                                    Text(
                                      Expense.formatCents(
                                        summary.totalCents,
                                      ),
                                      style: theme
                                          .textTheme.headlineMedium,
                                    ),
                                  ],
                                ),
                              ),
                              // By Person section.
                              Padding(
                                padding:
                                    const EdgeInsets.symmetric(
                                  horizontal: 16,
                                ),
                                child: Text(
                                  'By Person',
                                  style:
                                      theme.textTheme.titleMedium,
                                ),
                              ),
                              const SizedBox(height: 8),
                              ListView.separated(
                                shrinkWrap: true,
                                physics:
                                    const NeverScrollableScrollPhysics(),
                                itemCount:
                                    summary.byPerson.length,
                                separatorBuilder: (_, __) =>
                                    const Divider(indent: 56),
                                itemBuilder: (context, index) {
                                  final member =
                                      summary.byPerson[index];
                                  return ListTile(
                                    leading: CircleAvatar(
                                      child: Text(
                                        member.userEmail[0]
                                            .toUpperCase(),
                                      ),
                                    ),
                                    title: Text(
                                      member.userEmail,
                                      style: theme
                                          .textTheme.bodyLarge,
                                    ),
                                    subtitle: Text(
                                      '${member.expenseCount} '
                                      'expense'
                                      '${member.expenseCount == 1 ? '' : 's'}',
                                      style: theme
                                          .textTheme.bodySmall
                                          ?.copyWith(
                                        color: colorScheme
                                            .onSurfaceVariant,
                                      ),
                                    ),
                                    trailing: Text(
                                      Expense.formatCents(
                                        member.totalCents,
                                      ),
                                      style: theme
                                          .textTheme.titleMedium,
                                    ),
                                  );
                                },
                              ),
                              const SizedBox(height: 24),
                              // By Category section.
                              Padding(
                                padding:
                                    const EdgeInsets.symmetric(
                                  horizontal: 16,
                                ),
                                child: Text(
                                  'By Category',
                                  style:
                                      theme.textTheme.titleMedium,
                                ),
                              ),
                              const SizedBox(height: 8),
                              ListView.separated(
                                shrinkWrap: true,
                                physics:
                                    const NeverScrollableScrollPhysics(),
                                itemCount:
                                    summary.byCategory.length,
                                separatorBuilder: (_, __) =>
                                    const Divider(indent: 56),
                                itemBuilder: (context, index) {
                                  final cat =
                                      summary.byCategory[index];
                                  final color = parseHexColor(
                                    cat.categoryColor,
                                  );
                                  final iconData =
                                      categoryIcons[
                                              cat.categoryIcon] ??
                                          Icons.category;
                                  return ListTile(
                                    leading: Container(
                                      width: 36,
                                      height: 36,
                                      decoration: BoxDecoration(
                                        color: color,
                                        borderRadius:
                                            BorderRadius.circular(
                                          18,
                                        ),
                                      ),
                                      child: Icon(
                                        iconData,
                                        size: 18,
                                        color: Colors.white,
                                      ),
                                    ),
                                    title: Text(
                                      cat.categoryName,
                                    ),
                                    subtitle: Text(
                                      '${cat.expenseCount} '
                                      'expense'
                                      '${cat.expenseCount == 1 ? '' : 's'}',
                                      style: theme
                                          .textTheme.bodySmall
                                          ?.copyWith(
                                        color: colorScheme
                                            .onSurfaceVariant,
                                      ),
                                    ),
                                    trailing: Text(
                                      Expense.formatCents(
                                        cat.totalCents,
                                      ),
                                      style: theme
                                          .textTheme.titleMedium,
                                    ),
                                  );
                                },
                              ),
                              const SizedBox(height: 24),
                            ],
                          ),
                        ),
                      ),
            },
          ),
        ],
      ),
    );
  }
}
