import 'package:finance_tracker/features/charts/data/chart_repository.dart';
import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Manages chart state for the application.
///
/// Handles loading monthly spending summaries via [ChartRepository].
class ChartNotifier extends StateNotifier<ChartState> {
  /// Creates a [ChartNotifier].
  ChartNotifier(this._repository) : super(const ChartInitial());

  final ChartRepository _repository;

  /// Loads summary data for the given month.
  ///
  /// [month] format: "YYYY-MM" (e.g. "2026-03").
  Future<void> loadSummary({required String month}) async {
    state = const ChartLoading();
    try {
      final data = await _repository.getSummary(month: month);
      state = ChartLoaded(data);
    } on Exception catch (e) {
      state = ChartError(e.toString());
    }
  }
}

/// Provides the chart state and notifier.
final chartStateProvider =
    StateNotifierProvider<ChartNotifier, ChartState>((ref) {
  return ChartNotifier(ref.read(chartRepositoryProvider));
});
