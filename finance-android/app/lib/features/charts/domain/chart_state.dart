/// Breakdown of spending for one category.
class CategoryBreakdown {
  /// Creates a [CategoryBreakdown].
  const CategoryBreakdown({
    required this.categoryId,
    required this.categoryName,
    required this.categoryColor,
    required this.categoryIcon,
    required this.totalCents,
    required this.count,
  });

  /// Parses a [CategoryBreakdown] from a JSON map.
  factory CategoryBreakdown.fromJson(Map<String, dynamic> json) {
    return CategoryBreakdown(
      categoryId: json['category_id'] as String,
      categoryName: json['category_name'] as String,
      categoryColor: json['category_color'] as String,
      categoryIcon: json['category_icon'] as String,
      totalCents: json['total_cents'] as int,
      count: json['count'] as int,
    );
  }

  /// Category UUID.
  final String categoryId;

  /// Display name of the category.
  final String categoryName;

  /// Hex color string (e.g. "#FF7043").
  final String categoryColor;

  /// Icon name string (e.g. "restaurant").
  final String categoryIcon;

  /// Total spending in integer cents.
  final int totalCents;

  /// Number of expenses in this category.
  final int count;
}

/// Daily spending total.
class DateBreakdown {
  /// Creates a [DateBreakdown].
  const DateBreakdown({
    required this.date,
    required this.totalCents,
  });

  /// Parses a [DateBreakdown] from a JSON map.
  factory DateBreakdown.fromJson(Map<String, dynamic> json) {
    return DateBreakdown(
      date: json['date'] as String,
      totalCents: json['total_cents'] as int,
    );
  }

  /// Date string in "YYYY-MM-DD" format.
  final String date;

  /// Total spending in integer cents for this date.
  final int totalCents;
}

/// Aggregated chart data for a month.
class ChartData {
  /// Creates a [ChartData].
  const ChartData({
    required this.month,
    required this.totalCents,
    required this.byCategory,
    required this.byDate,
  });

  /// Parses [ChartData] from the summary API JSON response.
  factory ChartData.fromJson(Map<String, dynamic> json) {
    return ChartData(
      month: json['month'] as String,
      totalCents: json['total_cents'] as int,
      byCategory: (json['by_category'] as List<dynamic>)
          .cast<Map<String, dynamic>>()
          .map(CategoryBreakdown.fromJson)
          .toList(),
      byDate: (json['by_date'] as List<dynamic>)
          .cast<Map<String, dynamic>>()
          .map(DateBreakdown.fromJson)
          .toList(),
    );
  }

  /// Month string in "YYYY-MM" format.
  final String month;

  /// Total spending in integer cents for the month.
  final int totalCents;

  /// Spending broken down by category.
  final List<CategoryBreakdown> byCategory;

  /// Spending broken down by date.
  final List<DateBreakdown> byDate;
}

/// Sealed state for chart data management.
sealed class ChartState {
  /// Base constructor.
  const ChartState();
}

/// Initial state before chart data has been loaded.
class ChartInitial extends ChartState {
  /// Creates a [ChartInitial].
  const ChartInitial();
}

/// Loading state while fetching chart data from the API.
class ChartLoading extends ChartState {
  /// Creates a [ChartLoading].
  const ChartLoading();
}

/// Chart data successfully loaded.
class ChartLoaded extends ChartState {
  /// Creates a [ChartLoaded] with the given [data].
  const ChartLoaded(this.data);

  /// The aggregated chart data for the selected month.
  final ChartData data;
}

/// An error occurred during chart data loading.
class ChartError extends ChartState {
  /// Creates a [ChartError] with the given [message].
  const ChartError(this.message);

  /// Human-readable error description.
  final String message;
}
