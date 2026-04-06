import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Represents the current filter state for the expense history.
class FilterState {
  /// Creates a [FilterState].
  const FilterState({
    this.dateFrom,
    this.dateTo,
    this.categoryId,
    this.presetLabel,
  });

  /// Start of date range filter.
  final DateTime? dateFrom;

  /// End of date range filter.
  final DateTime? dateTo;

  /// Category ID filter.
  final String? categoryId;

  /// Label for preset date filters (e.g. "Today", "This Month").
  final String? presetLabel;

  /// Whether a date filter is active.
  bool get hasDateFilter => dateFrom != null;

  /// Whether a category filter is active.
  bool get hasCategoryFilter => categoryId != null;

  /// Whether any filter is active.
  bool get hasActiveFilters => hasDateFilter || hasCategoryFilter;

  /// Display label for the date filter chip.
  String get dateChipLabel {
    if (!hasDateFilter) return 'All Dates';
    if (presetLabel != null) return presetLabel!;
    // Custom range: format as "Mar 1 -- Mar 15"
    return '${_shortDate(dateFrom!)} -- ${_shortDate(dateTo!)}';
  }

  static String _shortDate(DateTime d) {
    const months = [
      'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', //
      'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
    ];
    final now = DateTime.now();
    if (d.year != now.year) {
      return '${months[d.month - 1]} ${d.day}, ${d.year}';
    }
    return '${months[d.month - 1]} ${d.day}';
  }
}

/// Manages the [FilterState] for expense history filtering.
class FilterNotifier extends StateNotifier<FilterState> {
  /// Creates a [FilterNotifier] with empty filters.
  FilterNotifier() : super(const FilterState());

  /// Sets a preset date filter (e.g. "Today", "This Month").
  void setDatePreset(String label, DateTime from, DateTime to) {
    state = FilterState(
      dateFrom: from,
      dateTo: to,
      categoryId: state.categoryId,
      presetLabel: label,
    );
  }

  /// Sets a custom date range filter.
  void setCustomDateRange(DateTime from, DateTime to) {
    state = FilterState(
      dateFrom: from,
      dateTo: to,
      categoryId: state.categoryId,
    );
  }

  /// Sets the category filter.
  void setCategory(String? categoryId) {
    state = FilterState(
      dateFrom: state.dateFrom,
      dateTo: state.dateTo,
      categoryId: categoryId,
      presetLabel: state.presetLabel,
    );
  }

  /// Clears the date filter.
  void clearDate() {
    state = FilterState(categoryId: state.categoryId);
  }

  /// Clears the category filter.
  void clearCategory() {
    state = FilterState(
      dateFrom: state.dateFrom,
      dateTo: state.dateTo,
      presetLabel: state.presetLabel,
    );
  }
}

/// Provides the filter state and notifier.
final filterStateProvider =
    StateNotifierProvider<FilterNotifier, FilterState>(
  (ref) => FilterNotifier(),
);

/// Calculates date range for a named preset.
({DateTime from, DateTime to}) calculateDatePreset(String preset) {
  final now = DateTime.now();
  return switch (preset) {
    'Today' => (
      from: DateTime(now.year, now.month, now.day),
      to: DateTime(now.year, now.month, now.day),
    ),
    'This Week' => (
      from: DateTime(now.year, now.month, now.day - (now.weekday - 1)),
      to: DateTime(now.year, now.month, now.day + (7 - now.weekday)),
    ),
    'This Month' => (
      from: DateTime(now.year, now.month),
      to: DateTime(now.year, now.month + 1, 0),
    ),
    'Last Month' => (
      from: DateTime(now.year, now.month - 1),
      to: DateTime(now.year, now.month, 0),
    ),
    _ => throw ArgumentError('Unknown preset: $preset'),
  };
}
