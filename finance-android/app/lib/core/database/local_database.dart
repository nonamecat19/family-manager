import 'package:path/path.dart';
import 'package:sqflite/sqflite.dart';

/// Manages the local SQLite database for offline storage.
///
/// Provides a singleton [Database] instance with tables for local expenses
/// (sync queue) and cached categories.
class LocalDatabase {
  static Database? _db;

  /// Returns the database instance, initializing on first access.
  static Future<Database> get database async {
    _db ??= await _initDb();
    return _db!;
  }

  /// Injects an in-memory database for unit tests.
  static void initForTest(Database db) {
    _db = db;
  }

  /// Closes the database and resets the singleton.
  static Future<void> close() async {
    await _db?.close();
    _db = null;
  }

  static Future<Database> _initDb() async {
    final dbPath = await getDatabasesPath();
    final path = join(dbPath, 'finance_tracker.db');

    return openDatabase(
      path,
      version: 1,
      onCreate: (db, version) async {
        await db.execute('''
          CREATE TABLE local_expenses (
            id TEXT PRIMARY KEY,
            server_id TEXT,
            category_id TEXT NOT NULL,
            amount_cents INTEGER NOT NULL,
            note TEXT NOT NULL DEFAULT '',
            expense_date TEXT NOT NULL,
            synced INTEGER NOT NULL DEFAULT 0,
            created_at TEXT NOT NULL DEFAULT (datetime('now'))
          )
        ''');
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
  }
}
