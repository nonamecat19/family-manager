import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/family/data/models/family_expense.dart';
import 'package:finance_tracker/features/family/domain/family_feed_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

/// Screen displaying the combined family expense feed.
///
/// Shows all family members' expenses for the selected month,
/// with each tile displaying the spender's email address.
/// Supports month navigation and pull-to-refresh.
class FamilyFeedScreen extends ConsumerStatefulWidget {
  /// Creates a [FamilyFeedScreen].
  const FamilyFeedScreen({super.key});

  @override
  ConsumerState<FamilyFeedScreen> createState() => _FamilyFeedScreenState();
}

class _FamilyFeedScreenState extends ConsumerState<FamilyFeedScreen> {
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
        .read(familyFeedStateProvider.notifier)
        .loadExpenses(month: formatted);
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
    final feedState = ref.watch(familyFeedStateProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Family Expenses')),
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
          // Feed content.
          Expanded(
            child: switch (feedState) {
              FamilyFeedInitial() ||
              FamilyFeedLoading() =>
                const Center(child: CircularProgressIndicator()),
              FamilyFeedError() => RefreshIndicator(
                  onRefresh: _loadData,
                  child: SingleChildScrollView(
                    physics: const AlwaysScrollableScrollPhysics(),
                    child: SizedBox(
                      height: MediaQuery.of(context).size.height * 0.5,
                      child: Center(
                        child: Text(
                          "Couldn't load family expenses. "
                          'Pull down to try again.',
                          style: theme.textTheme.bodyMedium,
                          textAlign: TextAlign.center,
                        ),
                      ),
                    ),
                  ),
                ),
              FamilyFeedLoaded(:final expenses) => expenses.isEmpty
                  ? Center(
                      child: Padding(
                        padding:
                            const EdgeInsets.symmetric(horizontal: 32),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(
                              Icons.receipt_long,
                              size: 48,
                              color: colorScheme.onSurfaceVariant,
                            ),
                            const SizedBox(height: 16),
                            Text(
                              'No family expenses',
                              style: theme.textTheme.titleLarge,
                            ),
                            const SizedBox(height: 8),
                            Text(
                              'No expenses logged this month. '
                              "Family members' spending will "
                              'appear here.',
                              style: theme.textTheme.bodyLarge,
                              textAlign: TextAlign.center,
                            ),
                          ],
                        ),
                      ),
                    )
                  : RefreshIndicator(
                      onRefresh: _loadData,
                      child: ListView.builder(
                        itemCount: expenses.length,
                        itemBuilder: (context, index) =>
                            _FamilyExpenseTile(
                          expense: expenses[index],
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

/// A single expense row in the family feed.
class _FamilyExpenseTile extends StatelessWidget {
  const _FamilyExpenseTile({required this.expense});

  final FamilyExpense expense;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final color = parseHexColor(expense.categoryColor);
    final iconData =
        categoryIcons[expense.categoryIcon] ?? Icons.category;

    return ListTile(
      leading: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: color,
          borderRadius: BorderRadius.circular(18),
        ),
        child: Icon(iconData, size: 18, color: Colors.white),
      ),
      title: Text(
        Expense.formatCents(expense.amountCents),
        style: theme.textTheme.titleMedium,
      ),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            expense.userEmail,
            style: theme.textTheme.bodyLarge?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
          if (expense.note.isNotEmpty)
            Text(
              expense.note,
              style: theme.textTheme.bodyLarge,
            ),
        ],
      ),
      trailing: Text(
        DateFormat.yMMMd().format(expense.expenseDate),
        style: theme.textTheme.bodySmall,
      ),
    );
  }
}
