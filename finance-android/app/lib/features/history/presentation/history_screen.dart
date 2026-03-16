import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:finance_tracker/features/history/domain/filter_state.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:finance_tracker/providers/expense_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

/// Screen displaying the user's expense history.
///
/// Shows a list of expenses with formatted amounts, dates, and
/// category chips. Supports tap-to-edit, swipe-to-delete, and
/// filtering by date range and category.
class HistoryScreen extends ConsumerStatefulWidget {
  /// Creates a [HistoryScreen].
  const HistoryScreen({super.key});

  @override
  ConsumerState<HistoryScreen> createState() => _HistoryScreenState();
}

class _HistoryScreenState extends ConsumerState<HistoryScreen> {
  @override
  void initState() {
    super.initState();
    final state = ref.read(expenseStateProvider);
    if (state is ExpenseInitial) {
      Future.microtask(
        () => ref.read(expenseStateProvider.notifier).loadExpenses(),
      );
    }

    final catState = ref.read(categoryStateProvider);
    if (catState is CategoryInitial) {
      Future.microtask(
        () => ref.read(categoryStateProvider.notifier).loadCategories(),
      );
    }
  }

  Category? _findCategory(
    List<Category> categories,
    String categoryId,
  ) {
    for (final cat in categories) {
      if (cat.id == categoryId) return cat;
    }
    return null;
  }

  Future<bool> _confirmDelete(
    BuildContext context,
    Expense expense,
  ) async {
    final result = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(
          'Delete this '
          '${Expense.formatCents(expense.amountCents)} expense?',
        ),
        content: const Text(
          'This expense will be permanently deleted.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Keep Expense'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Delete Expense'),
          ),
        ],
      ),
    );
    return result ?? false;
  }

  void _reloadWithFilters(FilterState filterState) {
    final notifier = ref.read(expenseStateProvider.notifier);
    notifier.loadExpenses(
      dateFrom: filterState.dateFrom
          ?.toIso8601String()
          .substring(0, 10),
      dateTo: filterState.dateTo
          ?.toIso8601String()
          .substring(0, 10),
      categoryId: filterState.categoryId,
    );
  }

  @override
  Widget build(BuildContext context) {
    final expenseState = ref.watch(expenseStateProvider);
    final categoryState = ref.watch(categoryStateProvider);
    final filterState = ref.watch(filterStateProvider);
    final categories = categoryState is CategoryLoaded
        ? categoryState.categories
        : <Category>[];

    ref.listen<FilterState>(filterStateProvider, (prev, next) {
      _reloadWithFilters(next);
    });

    return Column(
      children: [
        _FilterBar(
          filterState: filterState,
          categories: categories,
        ),
        Expanded(
          child: switch (expenseState) {
            ExpenseInitial() || ExpenseLoading() => const Center(
                child: CircularProgressIndicator(),
              ),
            ExpenseError(:final message) => Center(
                child: Text(message),
              ),
            ExpenseLoaded(:final expenses) => expenses.isEmpty
                ? _buildEmptyState(context, filterState)
                : ListView.builder(
                    itemCount: expenses.length,
                    itemBuilder: (context, index) {
                      final expense = expenses[index];
                      return Dismissible(
                        key: ValueKey(expense.id),
                        direction: DismissDirection.endToStart,
                        background: Container(
                          alignment: Alignment.centerRight,
                          padding: const EdgeInsets.only(right: 16),
                          color: Colors.red,
                          child: const Icon(
                            Icons.delete,
                            color: Colors.white,
                          ),
                        ),
                        confirmDismiss: (_) =>
                            _confirmDelete(context, expense),
                        onDismissed: (_) {
                          ref
                              .read(expenseStateProvider.notifier)
                              .deleteExpense(expense.id);
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(
                              content: Text('Expense deleted'),
                            ),
                          );
                        },
                        child: _ExpenseTile(
                          expense: expense,
                          category: _findCategory(
                            categories,
                            expense.categoryId,
                          ),
                          onTap: () => context.push(
                            '/expenses/edit',
                            extra: expense,
                          ),
                        ),
                      );
                    },
                  ),
          },
        ),
      ],
    );
  }

  Widget _buildEmptyState(BuildContext context, FilterState filterState) {
    if (filterState.hasActiveFilters) {
      final theme = Theme.of(context);
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.filter_list_off,
              size: 48,
              color: theme.colorScheme.onSurfaceVariant,
            ),
            const SizedBox(height: 8),
            const Text('No expenses match these filters'),
            const SizedBox(height: 4),
            Text(
              'Try adjusting your date range or category.',
              style: theme.textTheme.bodySmall,
            ),
          ],
        ),
      );
    }
    return const Center(child: Text('No expenses yet'));
  }
}

/// Filter bar with date and category chips.
class _FilterBar extends ConsumerWidget {
  const _FilterBar({
    required this.filterState,
    required this.categories,
  });

  final FilterState filterState;
  final List<Category> categories;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final categoryName = _selectedCategoryName();

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          FilterChip(
            label: Text(filterState.dateChipLabel),
            selected: filterState.hasDateFilter,
            onSelected: (_) =>
                _showDatePresetSheet(context, ref),
            onDeleted: filterState.hasDateFilter
                ? () =>
                    ref.read(filterStateProvider.notifier).clearDate()
                : null,
          ),
          const SizedBox(width: 8),
          FilterChip(
            label: Text(
              filterState.hasCategoryFilter
                  ? categoryName
                  : 'All Categories',
            ),
            selected: filterState.hasCategoryFilter,
            onSelected: (_) =>
                _showCategorySheet(context, ref),
            onDeleted: filterState.hasCategoryFilter
                ? () => ref
                    .read(filterStateProvider.notifier)
                    .clearCategory()
                : null,
          ),
        ],
      ),
    );
  }

  String _selectedCategoryName() {
    if (!filterState.hasCategoryFilter) return 'All Categories';
    for (final cat in categories) {
      if (cat.id == filterState.categoryId) return cat.name;
    }
    return 'Unknown';
  }

  void _showDatePresetSheet(BuildContext context, WidgetRef ref) {
    final colorScheme = Theme.of(context).colorScheme;
    showModalBottomSheet<void>(
      context: context,
      builder: (context) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(
              'Filter by Date',
              style: Theme.of(context).textTheme.titleMedium,
            ),
          ),
          if (filterState.hasDateFilter)
            ListTile(
              leading: const Icon(Icons.close),
              title: const Text('Clear'),
              onTap: () {
                ref.read(filterStateProvider.notifier).clearDate();
                Navigator.pop(context);
              },
            ),
          for (final preset in [
            'Today',
            'This Week',
            'This Month',
            'Last Month',
          ])
            ListTile(
              leading: filterState.presetLabel == preset
                  ? Icon(Icons.check, color: colorScheme.primary)
                  : const SizedBox(width: 24),
              title: Text(preset),
              onTap: () {
                final dates = calculateDatePreset(preset);
                ref
                    .read(filterStateProvider.notifier)
                    .setDatePreset(preset, dates.from, dates.to);
                Navigator.pop(context);
              },
            ),
          ListTile(
            leading: const Icon(Icons.date_range),
            title: const Text('Custom Range'),
            onTap: () async {
              Navigator.pop(context);
              final range = await showDateRangePicker(
                context: context,
                firstDate: DateTime(2020),
                lastDate: DateTime.now(),
              );
              if (range == null) return;
              ref
                  .read(filterStateProvider.notifier)
                  .setCustomDateRange(range.start, range.end);
            },
          ),
        ],
      ),
    );
  }

  void _showCategorySheet(BuildContext context, WidgetRef ref) {
    final colorScheme = Theme.of(context).colorScheme;
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (context) => ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(context).size.height * 0.5,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                'Filter by Category',
                style: Theme.of(context).textTheme.titleMedium,
              ),
            ),
            if (filterState.hasCategoryFilter)
              ListTile(
                leading: const Icon(Icons.close),
                title: const Text('Clear'),
                onTap: () {
                  ref
                      .read(filterStateProvider.notifier)
                      .clearCategory();
                  Navigator.pop(context);
                },
              ),
            Flexible(
              child: ListView(
                shrinkWrap: true,
                children: [
                  for (final category in categories)
                    ListTile(
                      leading: _CategoryChipSmall(category: category),
                      title: Text(category.name),
                      trailing:
                          filterState.categoryId == category.id
                              ? Icon(
                                  Icons.check,
                                  color: colorScheme.primary,
                                )
                              : null,
                      onTap: () {
                        ref
                            .read(filterStateProvider.notifier)
                            .setCategory(category.id);
                        Navigator.pop(context);
                      },
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A single expense row in the history list.
class _ExpenseTile extends StatelessWidget {
  const _ExpenseTile({
    required this.expense,
    this.category,
    this.onTap,
  });

  final Expense expense;
  final Category? category;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final dateStr = DateFormat.yMMMd().format(expense.expenseDate);

    return ListTile(
      leading: category != null
          ? _CategoryChipSmall(category: category!)
          : null,
      title: Text(
        Expense.formatCents(expense.amountCents),
        style: theme.textTheme.titleMedium,
      ),
      subtitle: Text(
        expense.note.isNotEmpty ? expense.note : dateStr,
        style: theme.textTheme.bodyMedium,
      ),
      trailing: Text(
        dateStr,
        style: theme.textTheme.bodySmall,
      ),
      onTap: onTap,
    );
  }
}

/// Compact circular category icon for ListTile leading position.
class _CategoryChipSmall extends StatelessWidget {
  const _CategoryChipSmall({required this.category});

  final Category category;

  @override
  Widget build(BuildContext context) {
    final color = parseHexColor(category.color);
    final iconData = categoryIcons[category.icon] ?? Icons.category;

    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: color.withAlpha(38),
        borderRadius: BorderRadius.circular(18),
      ),
      child: Icon(iconData, size: 18, color: color),
    );
  }
}
