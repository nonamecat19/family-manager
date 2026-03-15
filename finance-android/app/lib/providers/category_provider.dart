import 'dart:async';

import 'package:finance_tracker/features/categories/data/category_repository.dart';
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Default starter categories offered on first visit.
const List<Map<String, String>> starterCategories = [
  {'name': 'Food', 'icon': 'restaurant', 'color': '#FF7043'},
  {'name': 'Transport', 'icon': 'directions_car', 'color': '#42A5F5'},
  {'name': 'Housing', 'icon': 'home', 'color': '#66BB6A'},
  {'name': 'Shopping', 'icon': 'shopping_cart', 'color': '#AB47BC'},
  {'name': 'Coffee', 'icon': 'local_cafe', 'color': '#8D6E63'},
  {'name': 'Entertainment', 'icon': 'movie', 'color': '#FFCA28'},
];

/// Manages category state for the application.
///
/// Handles CRUD operations, reorder (with debounced API call),
/// and bulk creation of starter categories via [CategoryRepository].
class CategoryNotifier extends StateNotifier<CategoryState> {
  /// Creates a [CategoryNotifier].
  CategoryNotifier(this._repository) : super(const CategoryInitial());

  final CategoryRepository _repository;
  Timer? _reorderTimer;

  /// Loads all categories for the authenticated user.
  Future<void> loadCategories() async {
    state = const CategoryLoading();
    try {
      final categories = await _repository.getCategories();
      state = CategoryLoaded(categories);
    } on Exception catch (e) {
      state = CategoryError(e.toString());
    }
  }

  /// Creates a new category and refreshes the list.
  Future<void> createCategory({
    required String name,
    required String icon,
    required String color,
  }) async {
    try {
      await _repository.createCategory(name: name, icon: icon, color: color);
      await loadCategories();
    } on Exception catch (e) {
      state = CategoryError(e.toString());
    }
  }

  /// Updates an existing category and refreshes the list.
  Future<void> updateCategory({
    required String id,
    required String name,
    required String icon,
    required String color,
  }) async {
    try {
      await _repository.updateCategory(
        id: id,
        name: name,
        icon: icon,
        color: color,
      );
      await loadCategories();
    } on Exception catch (e) {
      state = CategoryError(e.toString());
    }
  }

  /// Deletes a category and refreshes the list.
  Future<void> deleteCategory(String id) async {
    try {
      await _repository.deleteCategory(id);
      await loadCategories();
    } on Exception catch (e) {
      state = CategoryError(e.toString());
    }
  }

  /// Optimistically reorders categories and debounces the API call.
  ///
  /// Updates local state immediately for responsive UX, then sends
  /// the reorder request to the server after 500ms of inactivity.
  void reorderCategories(int oldIndex, int newIndex) {
    final currentState = state;
    if (currentState is! CategoryLoaded) return;

    final items = List.of(currentState.categories);
    // ReorderableListView adjusts newIndex when moving downward.
    final adjustedNew = oldIndex < newIndex ? newIndex - 1 : newIndex;
    final item = items.removeAt(oldIndex);
    items.insert(adjustedNew, item);

    // Update sort orders to match new positions.
    final reordered = [
      for (var i = 0; i < items.length; i++) items[i].copyWith(sortOrder: i),
    ];
    state = CategoryLoaded(reordered);

    // Debounce the API call.
    _reorderTimer?.cancel();
    _reorderTimer = Timer(const Duration(milliseconds: 500), () {
      _repository.reorderCategories([
        for (final cat in reordered)
          {'id': cat.id, 'sort_order': cat.sortOrder},
      ]);
    });
  }

  /// Bulk creates the 6 starter categories.
  Future<void> bulkCreateStarters() async {
    try {
      await _repository.bulkCreateCategories(starterCategories);
      await loadCategories();
    } on Exception catch (e) {
      state = CategoryError(e.toString());
    }
  }

  @override
  void dispose() {
    _reorderTimer?.cancel();
    super.dispose();
  }
}

/// Provides the category state and notifier.
final categoryStateProvider =
    StateNotifierProvider<CategoryNotifier, CategoryState>((ref) {
  return CategoryNotifier(ref.read(categoryRepositoryProvider));
});
