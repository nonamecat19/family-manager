import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:sqflite/sqflite.dart';

/// Local cache for categories stored in sqflite.
///
/// Categories are cached when fetched from the server so the expense form
/// works offline.
class LocalCategorySource {
  /// Creates a [LocalCategorySource] with the given [Database].
  const LocalCategorySource(this._db);

  final Database _db;

  /// Replaces all cached categories with the provided list.
  ///
  /// Deletes existing rows first, then inserts each category in a batch.
  Future<void> cacheCategories(List<Category> categories) async {
    final batch = _db.batch();
    batch.delete('cached_categories');
    for (final cat in categories) {
      batch.insert(
        'cached_categories',
        {
          'id': cat.id,
          'name': cat.name,
          'icon': cat.icon,
          'color': cat.color,
          'sort_order': cat.sortOrder,
        },
        conflictAlgorithm: ConflictAlgorithm.replace,
      );
    }
    await batch.commit(noResult: true);
  }

  /// Returns all cached categories ordered by sort_order ascending.
  Future<List<Category>> getCachedCategories() async {
    final rows = await _db.query(
      'cached_categories',
      orderBy: 'sort_order ASC',
    );
    return rows
        .map(
          (row) => Category(
            id: row['id']! as String,
            name: row['name']! as String,
            icon: row['icon']! as String,
            color: row['color']! as String,
            sortOrder: row['sort_order']! as int,
          ),
        )
        .toList();
  }
}
