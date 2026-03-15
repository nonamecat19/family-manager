import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:finance_tracker/providers/expense_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

/// Screen displaying the user's expense history.
///
/// Shows a list of expenses with formatted amounts, dates, and
/// category chips. Supports tap-to-edit and swipe-to-delete.
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

  @override
  Widget build(BuildContext context) {
    final expenseState = ref.watch(expenseStateProvider);
    final categoryState = ref.watch(categoryStateProvider);
    final categories = categoryState is CategoryLoaded
        ? categoryState.categories
        : <Category>[];

    return switch (expenseState) {
      ExpenseInitial() || ExpenseLoading() => const Center(
          child: CircularProgressIndicator(),
        ),
      ExpenseError(:final message) => Center(
          child: Text(message),
        ),
      ExpenseLoaded(:final expenses) => expenses.isEmpty
          ? const Center(child: Text('No expenses yet'))
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
    };
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
