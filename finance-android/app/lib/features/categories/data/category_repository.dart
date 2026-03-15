import 'package:dio/dio.dart';
import 'package:finance_tracker/core/network/api_client.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Repository for category API calls.
///
/// Communicates with the Go backend category endpoints
/// at `/categories`.
class CategoryRepository {
  /// Creates a [CategoryRepository] with the given [Dio] client.
  const CategoryRepository(this._dio);

  final Dio _dio;

  /// Fetches all categories for the authenticated user.
  Future<List<Category>> getCategories() async {
    final response = await _dio.get<List<dynamic>>('/categories');
    return response.data!
        .cast<Map<String, dynamic>>()
        .map(Category.fromJson)
        .toList();
  }

  /// Creates a new category with [name], [icon], and [color].
  Future<Category> createCategory({
    required String name,
    required String icon,
    required String color,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/categories',
      data: {'name': name, 'icon': icon, 'color': color},
    );
    return Category.fromJson(response.data!);
  }

  /// Updates an existing category identified by [id].
  Future<void> updateCategory({
    required String id,
    required String name,
    required String icon,
    required String color,
  }) async {
    await _dio.put<void>(
      '/categories/$id',
      data: {'name': name, 'icon': icon, 'color': color},
    );
  }

  /// Deletes a category identified by [id].
  Future<void> deleteCategory(String id) async {
    await _dio.delete<void>('/categories/$id');
  }

  /// Reorders categories by sending a list of {id, sort_order} pairs.
  Future<void> reorderCategories(
    List<Map<String, dynamic>> orderedItems,
  ) async {
    await _dio.put<void>(
      '/categories/reorder',
      data: orderedItems,
    );
  }

  /// Bulk creates multiple categories (for starter categories).
  Future<List<Category>> bulkCreateCategories(
    List<Map<String, String>> categories,
  ) async {
    final response = await _dio.post<List<dynamic>>(
      '/categories/bulk',
      data: {'categories': categories},
    );
    return response.data!
        .cast<Map<String, dynamic>>()
        .map(Category.fromJson)
        .toList();
  }
}

/// Provides a [CategoryRepository] using the app's Dio client.
final categoryRepositoryProvider = Provider<CategoryRepository>((ref) {
  return CategoryRepository(ref.read(dioProvider));
});
