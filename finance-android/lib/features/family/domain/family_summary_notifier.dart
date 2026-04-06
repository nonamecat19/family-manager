import 'package:dio/dio.dart';
import 'package:finance_tracker/features/family/data/family_view_repository.dart';
import 'package:finance_tracker/features/family/data/models/family_summary.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Sealed state for the family summary dashboard.
sealed class FamilySummaryState {
  /// Base constructor.
  const FamilySummaryState();
}

/// Initial state before summary data has been loaded.
class FamilySummaryInitial extends FamilySummaryState {
  /// Creates a [FamilySummaryInitial].
  const FamilySummaryInitial();
}

/// Loading state while fetching family summary.
class FamilySummaryLoading extends FamilySummaryState {
  /// Creates a [FamilySummaryLoading].
  const FamilySummaryLoading();
}

/// Family summary successfully loaded.
class FamilySummaryLoaded extends FamilySummaryState {
  /// Creates a [FamilySummaryLoaded] with the given [summary].
  const FamilySummaryLoaded(this.summary);

  /// The aggregated family spending summary.
  final FamilySummary summary;
}

/// An error occurred while loading the family summary.
class FamilySummaryError extends FamilySummaryState {
  /// Creates a [FamilySummaryError] with the given [message].
  const FamilySummaryError(this.message);

  /// Human-readable error description.
  final String message;
}

/// Manages the family summary dashboard state.
///
/// Loads monthly summary via [FamilyViewRepository].
class FamilySummaryNotifier extends StateNotifier<FamilySummaryState> {
  /// Creates a [FamilySummaryNotifier].
  FamilySummaryNotifier(this._repository)
      : super(const FamilySummaryInitial());

  final FamilyViewRepository _repository;

  /// Loads the family summary for the given [month] (format: "YYYY-MM").
  Future<void> loadSummary({required String month}) async {
    state = const FamilySummaryLoading();
    try {
      final summary = await _repository.getFamilySummary(month: month);
      state = FamilySummaryLoaded(summary);
    } on DioException catch (e) {
      state =
          FamilySummaryError(e.message ?? 'Failed to load family summary');
    } on Exception catch (e) {
      state = FamilySummaryError(e.toString());
    }
  }
}

/// Provides the family summary state and notifier.
final familySummaryStateProvider =
    StateNotifierProvider<FamilySummaryNotifier, FamilySummaryState>((ref) {
  return FamilySummaryNotifier(ref.read(familyViewRepositoryProvider));
});

/// A fake [FamilySummaryNotifier] for widget test mocking.
///
/// All methods are no-ops. Override [state] in tests to control UI.
class FakeFamilySummaryNotifier extends StateNotifier<FamilySummaryState>
    implements FamilySummaryNotifier {
  /// Creates a [FakeFamilySummaryNotifier] with the given initial [state].
  FakeFamilySummaryNotifier([super.initial = const FamilySummaryInitial()]);

  @override
  // ignore: unused_field
  FamilyViewRepository get _repository => throw UnimplementedError();

  @override
  Future<void> loadSummary({required String month}) async {}
}
