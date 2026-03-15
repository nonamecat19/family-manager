import 'package:finance_tracker/features/categories/data/models/category.dart';

/// Represents the state of category management.
sealed class CategoryState {
  const CategoryState();
}

/// Initial state before categories have been loaded.
class CategoryInitial extends CategoryState {
  const CategoryInitial();
}

/// Loading state while fetching categories from the API.
class CategoryLoading extends CategoryState {
  const CategoryLoading();
}

/// Categories successfully loaded.
class CategoryLoaded extends CategoryState {
  /// Creates a [CategoryLoaded] state with the given [categories].
  const CategoryLoaded(this.categories);

  /// The user's categories sorted by [Category.sortOrder].
  final List<Category> categories;
}

/// An error occurred during a category operation.
class CategoryError extends CategoryState {
  /// Creates a [CategoryError] with the given [message].
  const CategoryError(this.message);

  /// Human-readable error description.
  final String message;
}
