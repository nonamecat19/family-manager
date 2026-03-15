import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/features/expenses/domain/expense_state.dart';
import 'package:finance_tracker/providers/expense_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

/// Screen displaying the user's expense history.
///
/// Shows a list of expenses with formatted amounts and dates.
/// Triggers expense loading on first render.
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
  }

  @override
  Widget build(BuildContext context) {
    final expenseState = ref.watch(expenseStateProvider);

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
                return _ExpenseTile(expense: expense);
              },
            ),
    };
  }
}

/// A single expense row in the history list.
class _ExpenseTile extends StatelessWidget {
  const _ExpenseTile({required this.expense});

  final Expense expense;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final dateStr = DateFormat.yMMMd().format(expense.expenseDate);

    return ListTile(
      title: Text(
        Expense.formatCents(expense.amountCents),
        style: theme.textTheme.titleMedium,
      ),
      subtitle: Text(
        expense.note.isNotEmpty ? expense.note : dateStr,
      ),
      trailing: Text(
        dateStr,
        style: theme.textTheme.bodySmall,
      ),
    );
  }
}
