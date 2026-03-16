import 'package:dio/dio.dart';
import 'package:finance_tracker/features/family/data/family_view_repository.dart';
import 'package:finance_tracker/features/family/data/models/family_expense.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Sealed state for the family expense feed.
sealed class FamilyFeedState {
  /// Base constructor.
  const FamilyFeedState();
}

/// Initial state before feed data has been loaded.
class FamilyFeedInitial extends FamilyFeedState {
  /// Creates a [FamilyFeedInitial].
  const FamilyFeedInitial();
}

/// Loading state while fetching family expenses.
class FamilyFeedLoading extends FamilyFeedState {
  /// Creates a [FamilyFeedLoading].
  const FamilyFeedLoading();
}

/// Family expenses successfully loaded.
class FamilyFeedLoaded extends FamilyFeedState {
  /// Creates a [FamilyFeedLoaded] with the given [expenses].
  const FamilyFeedLoaded(this.expenses);

  /// The list of family expenses for the selected month.
  final List<FamilyExpense> expenses;
}

/// An error occurred while loading family expenses.
class FamilyFeedError extends FamilyFeedState {
  /// Creates a [FamilyFeedError] with the given [message].
  const FamilyFeedError(this.message);

  /// Human-readable error description.
  final String message;
}

/// Manages the family expense feed state.
///
/// Loads expenses via [FamilyViewRepository] and filters by month
/// client-side (server endpoint is paginated without month filter).
class FamilyFeedNotifier extends StateNotifier<FamilyFeedState> {
  /// Creates a [FamilyFeedNotifier].
  FamilyFeedNotifier(this._repository) : super(const FamilyFeedInitial());

  final FamilyViewRepository _repository;

  /// Loads family expenses for the given [month] (format: "YYYY-MM").
  ///
  /// Fetches a large page and filters client-side by month.
  Future<void> loadExpenses({required String month}) async {
    state = const FamilyFeedLoading();
    try {
      final expenses = await _repository.getFamilyExpenses(limit: 200);
      final monthDate = DateTime.parse('$month-01');
      final filtered = expenses
          .where(
            (e) =>
                e.expenseDate.year == monthDate.year &&
                e.expenseDate.month == monthDate.month,
          )
          .toList();
      state = FamilyFeedLoaded(filtered);
    } on DioException catch (e) {
      state = FamilyFeedError(e.message ?? 'Failed to load family expenses');
    } on Exception catch (e) {
      state = FamilyFeedError(e.toString());
    }
  }
}

/// Provides the family feed state and notifier.
final familyFeedStateProvider =
    StateNotifierProvider<FamilyFeedNotifier, FamilyFeedState>((ref) {
  return FamilyFeedNotifier(ref.read(familyViewRepositoryProvider));
});

/// A fake [FamilyFeedNotifier] for widget test mocking.
///
/// All methods are no-ops. Override [state] in tests to control UI.
class FakeFamilyFeedNotifier extends StateNotifier<FamilyFeedState>
    implements FamilyFeedNotifier {
  /// Creates a [FakeFamilyFeedNotifier] with the given initial [state].
  FakeFamilyFeedNotifier([super.initial = const FamilyFeedInitial()]);

  @override
  // ignore: unused_field
  FamilyViewRepository get _repository => throw UnimplementedError();

  @override
  Future<void> loadExpenses({required String month}) async {}
}
