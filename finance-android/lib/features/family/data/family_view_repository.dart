import 'package:dio/dio.dart';
import 'package:finance_tracker/core/network/api_client.dart';
import 'package:finance_tracker/features/family/data/models/family_expense.dart';
import 'package:finance_tracker/features/family/data/models/family_summary.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Repository for family expense view API calls.
///
/// Consumes the family expense feed and summary endpoints
/// built in Phase 9 Plan 1.
class FamilyViewRepository {
  /// Creates a [FamilyViewRepository] with the given [Dio] client.
  const FamilyViewRepository(this._dio);

  final Dio _dio;

  /// Fetches paginated family expenses.
  ///
  /// Returns a list of [FamilyExpense] from all family members.
  Future<List<FamilyExpense>> getFamilyExpenses({
    int limit = 50,
    int offset = 0,
  }) async {
    final response = await _dio.get<List<dynamic>>(
      '/families/me/expenses',
      queryParameters: {'limit': limit, 'offset': offset},
    );
    return (response.data ?? [])
        .cast<Map<String, dynamic>>()
        .map(FamilyExpense.fromJson)
        .toList();
  }

  /// Fetches the family spending summary for a given [month].
  ///
  /// [month] should be in "YYYY-MM" format.
  /// Returns a [FamilySummary] with per-person and per-category breakdowns.
  Future<FamilySummary> getFamilySummary({required String month}) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/families/me/summary',
      queryParameters: {'month': month},
    );
    return FamilySummary.fromJson(response.data!);
  }
}

/// Provides a [FamilyViewRepository] using the app's Dio client.
final familyViewRepositoryProvider = Provider<FamilyViewRepository>((ref) {
  return FamilyViewRepository(ref.read(dioProvider));
});
