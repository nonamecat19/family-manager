import 'package:finance_tracker/features/expenses/data/models/expense.dart';

/// Represents the state of expense management.
sealed class ExpenseState {
  const ExpenseState();
}

/// Initial state before expenses have been loaded.
class ExpenseInitial extends ExpenseState {
  const ExpenseInitial();
}

/// Loading state while fetching expenses from the API.
class ExpenseLoading extends ExpenseState {
  const ExpenseLoading();
}

/// Expenses successfully loaded.
class ExpenseLoaded extends ExpenseState {
  /// Creates an [ExpenseLoaded] state with the given [expenses].
  const ExpenseLoaded(this.expenses);

  /// The user's expenses ordered by most recent first.
  final List<Expense> expenses;
}

/// An error occurred during an expense operation.
class ExpenseError extends ExpenseState {
  /// Creates an [ExpenseError] with the given [message].
  const ExpenseError(this.message);

  /// Human-readable error description.
  final String message;
}
