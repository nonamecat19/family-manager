import 'package:dio/dio.dart';
import 'package:finance_tracker/core/network/api_client.dart';
import 'package:finance_tracker/features/charts/domain/chart_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Repository for chart summary API calls.
class ChartRepository {
  /// Creates a [ChartRepository] with the given [Dio] client.
  const ChartRepository(this._dio);

  final Dio _dio;

  /// Fetches the monthly spending summary.
  ///
  /// [month] format: "YYYY-MM" (e.g. "2026-03").
  Future<ChartData> getSummary({required String month}) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/expenses/summary',
      queryParameters: {'month': month},
    );
    return ChartData.fromJson(response.data!);
  }
}

/// Provides a [ChartRepository] using the app's Dio client.
final chartRepositoryProvider = Provider<ChartRepository>((ref) {
  return ChartRepository(ref.read(dioProvider));
});
