import 'package:finance_tracker/core/database/local_category_source.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

void main() {
  late Database db;
  late LocalCategorySource source;

  setUpAll(() {
    sqfliteFfiInit();
    databaseFactory = databaseFactoryFfi;
  });

  setUp(() async {
    db = await openDatabase(
      inMemoryDatabasePath,
      version: 1,
      onCreate: (db, version) async {
        await db.execute('''
          CREATE TABLE cached_categories (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            icon TEXT NOT NULL,
            color TEXT NOT NULL,
            sort_order INTEGER NOT NULL DEFAULT 0
          )
        ''');
      },
    );
    source = LocalCategorySource(db);
  });

  tearDown(() async {
    await db.close();
  });

  group('LocalCategorySource', () {
    test('cacheCategories stores categories in the database', () async {
      final categories = [
        const Category(
          id: 'c1',
          name: 'Food',
          icon: 'restaurant',
          color: '#FF5722',
          sortOrder: 0,
        ),
        const Category(
          id: 'c2',
          name: 'Transport',
          icon: 'directions_car',
          color: '#2196F3',
          sortOrder: 1,
        ),
      ];

      await source.cacheCategories(categories);

      final rows = await db.query('cached_categories');
      expect(rows, hasLength(2));
    });

    test('getCachedCategories returns sorted by sort_order', () async {
      final categories = [
        const Category(
          id: 'c2',
          name: 'Transport',
          icon: 'directions_car',
          color: '#2196F3',
          sortOrder: 2,
        ),
        const Category(
          id: 'c1',
          name: 'Food',
          icon: 'restaurant',
          color: '#FF5722',
          sortOrder: 1,
        ),
      ];

      await source.cacheCategories(categories);
      final result = await source.getCachedCategories();

      expect(result, hasLength(2));
      expect(result[0].name, 'Food');
      expect(result[1].name, 'Transport');
    });

    test('cacheCategories replaces existing data', () async {
      final original = [
        const Category(
          id: 'c1',
          name: 'Food',
          icon: 'restaurant',
          color: '#FF5722',
          sortOrder: 0,
        ),
      ];
      await source.cacheCategories(original);

      final replacement = [
        const Category(
          id: 'c3',
          name: 'Health',
          icon: 'health_and_safety',
          color: '#4CAF50',
          sortOrder: 0,
        ),
      ];
      await source.cacheCategories(replacement);

      final result = await source.getCachedCategories();
      expect(result, hasLength(1));
      expect(result[0].name, 'Health');
    });
  });
}
