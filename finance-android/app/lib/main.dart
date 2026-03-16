import 'package:finance_tracker/app.dart';
import 'package:finance_tracker/core/database/local_category_source.dart';
import 'package:finance_tracker/core/database/local_database.dart';
import 'package:finance_tracker/core/database/local_expense_source.dart';
import 'package:finance_tracker/features/sync/data/sync_service.dart';
import 'package:finance_tracker/providers/auth_provider.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:sqflite/sqflite.dart';
import 'package:sqflite_common_ffi_web/sqflite_ffi_web.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Web platform SQLite setup.
  if (kIsWeb) {
    databaseFactory = databaseFactoryFfiWeb;
  }

  // Initialize local database.
  final db = await LocalDatabase.database;
  final localExpenseSource = LocalExpenseSource(db);
  final localCategorySource = LocalCategorySource(db);

  final container = ProviderContainer(
    overrides: [
      localExpenseSourceProvider.overrideWithValue(localExpenseSource),
      localCategorySourceProvider.overrideWithValue(localCategorySource),
    ],
  );

  // Attempt session restoration before showing UI.
  container.read(authStateProvider.notifier).tryRestoreSession();

  runApp(
    UncontrolledProviderScope(
      container: container,
      child: const FinanceApp(),
    ),
  );
}
