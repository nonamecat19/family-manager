# Phase 10: Offline and Platform Polish - Research

**Researched:** 2026-03-16
**Domain:** Flutter offline-first architecture, local storage, connectivity detection, background sync
**Confidence:** MEDIUM-HIGH

## Summary

Phase 10 adds offline expense creation with background sync and ensures cross-platform polish. The existing app is purely online -- all repositories call Dio directly and fail if the network is unavailable. The goal is narrow: users can log expenses while offline, those expenses appear locally, and they sync to the server when connectivity returns. Conflict resolution must prevent data loss and duplicates.

The recommended approach is a **sync queue pattern** using sqflite for local persistence and connectivity_plus for network detection. This aligns with the official Flutter offline-first documentation. The app already uses Riverpod StateNotifier patterns throughout, so the offline layer integrates as a new data source alongside the existing Dio-based repositories.

**Primary recommendation:** Add sqflite for a local expense cache + pending sync queue, connectivity_plus StreamProvider to detect online/offline, and modify ExpenseNotifier to write locally first then sync in the background. No server-side changes needed -- the existing POST /expenses endpoint is idempotent-safe with client-generated UUIDs.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PLAT-02 | App works offline and syncs when connectivity is restored | sqflite local DB + connectivity_plus detection + sync queue pattern; official Flutter docs recommend this architecture |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| sqflite | ^2.4.2 | Local SQLite database for expense cache + sync queue | Industry standard for Flutter local storage; official Flutter docs recommend SQLite for offline-first |
| connectivity_plus | ^7.0.0 | Detect network connectivity changes | Official Flutter Community plugin; supports all platforms including web |
| sqflite_common_ffi_web | ^1.1.0 | Web platform SQLite support | Required for sqflite to work on Flutter web; uses IndexedDB under the hood |
| path_provider | ^2.1.0 | Get database file path | Required by sqflite for database file location |
| uuid | ^4.0.0 | Client-side UUID generation for offline expenses | Needed so expenses created offline have IDs before server round-trip |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| path | ^1.9.0 | Path joining for DB file | Already transitive dep; used with path_provider |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| sqflite | drift | Drift adds type-safe ORM but overkill for one table + sync queue; sqflite is simpler, matches project's raw-SQL style on server |
| sqflite | shared_preferences | shared_preferences can't handle relational queries or large datasets; not suitable for expense list caching |
| sqflite | hive | Hive is NoSQL, abandoned maintainer; sqflite is relational like the server DB |
| connectivity_plus | internet_connection_checker | connectivity_plus is lighter; actual HTTP reachability check adds latency and is unnecessary when we already guard with try/catch on Dio calls |

**Installation:**
```bash
cd app && flutter pub add sqflite connectivity_plus sqflite_common_ffi_web path_provider uuid
```

## Architecture Patterns

### Recommended Project Structure
```
lib/
  core/
    database/
      local_database.dart        # sqflite initialization, table creation
      sync_queue.dart             # Pending operations queue logic
    network/
      connectivity_provider.dart  # Riverpod StreamProvider for connectivity
  features/
    expenses/
      data/
        local_expense_source.dart # Local CRUD operations on sqflite
        expense_repository.dart   # Modified: local-first with sync
      domain/
        expense_state.dart        # Add pendingSync flag to state
    sync/
      data/
        sync_service.dart         # Background sync engine
      domain/
        sync_state.dart           # Sync status (idle/syncing/error)
```

### Pattern 1: Local-First Write with Sync Queue
**What:** When creating an expense, write to local sqflite first, then attempt server sync. If offline, queue for later sync.
**When to use:** All expense create operations.
**Example:**
```dart
// Source: https://docs.flutter.dev/app-architecture/design-patterns/offline-first
Future<bool> createExpense({
  required String categoryId,
  required int amountCents,
  required String note,
  required String expenseDate,
}) async {
  // Generate client-side UUID
  final localId = const Uuid().v4();
  final expense = Expense(
    id: localId,
    categoryId: categoryId,
    amountCents: amountCents,
    note: note,
    expenseDate: DateTime.parse(expenseDate),
  );

  // 1. Write to local DB immediately
  await _localSource.insertExpense(expense, synced: false);

  // 2. Update UI state instantly
  final current = _currentExpenses();
  state = ExpenseLoaded([expense, ...current]);

  // 3. Attempt server sync
  try {
    final serverExpense = await _repository.createExpense(
      categoryId: categoryId,
      amountCents: amountCents,
      note: note,
      expenseDate: expenseDate,
    );
    await _localSource.markSynced(localId, serverExpense.id);
    return true;
  } on DioException {
    // Stays in local DB with synced=false, will retry later
    return true; // Still "success" from user perspective
  }
}
```

### Pattern 2: Connectivity-Triggered Sync
**What:** When connectivity is restored, automatically process all pending (unsynced) expenses.
**When to use:** App lifecycle and connectivity change events.
**Example:**
```dart
// Riverpod StreamProvider for connectivity
final connectivityProvider = StreamProvider<bool>((ref) {
  return Connectivity().onConnectivityChanged.map((results) {
    return results.any((r) =>
      r == ConnectivityResult.wifi ||
      r == ConnectivityResult.mobile ||
      r == ConnectivityResult.ethernet);
  });
});

// In sync service: listen for connectivity changes
ref.listen(connectivityProvider, (prev, next) {
  final isOnline = next.valueOrNull ?? false;
  if (isOnline) {
    syncPendingExpenses();
  }
});
```

### Pattern 3: Sync Queue Table
**What:** A dedicated sqflite table tracking pending operations with their payloads.
**When to use:** Storing offline mutations for later replay.
**Example:**
```sql
CREATE TABLE pending_sync (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  operation TEXT NOT NULL,  -- 'create', 'update', 'delete'
  entity_type TEXT NOT NULL, -- 'expense'
  local_id TEXT NOT NULL,
  payload TEXT NOT NULL,     -- JSON serialized request body
  created_at TEXT NOT NULL,
  retry_count INTEGER DEFAULT 0
);

CREATE TABLE local_expenses (
  id TEXT PRIMARY KEY,       -- client-generated UUID
  server_id TEXT,            -- NULL until synced
  category_id TEXT NOT NULL,
  amount_cents INTEGER NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  expense_date TEXT NOT NULL,
  synced INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
```

### Pattern 4: Offline Banner UI
**What:** Show a non-intrusive banner when the app is offline so users know their data will sync later.
**When to use:** Always visible at the top of the screen when offline.
**Example:**
```dart
// In AppScaffold or as a global overlay
Consumer(
  builder: (context, ref, child) {
    final connectivity = ref.watch(connectivityProvider);
    final isOffline = connectivity.valueOrNull == false;
    if (!isOffline) return const SizedBox.shrink();
    return MaterialBanner(
      content: const Text('You are offline. Expenses will sync when connected.'),
      actions: [const SizedBox.shrink()],
      backgroundColor: Colors.orange.shade100,
    );
  },
)
```

### Anti-Patterns to Avoid
- **Generating UUIDs server-side for offline creates:** If the client waits for a server ID, it can't work offline. Generate UUIDs client-side with the `uuid` package.
- **Using connectivity_plus as sole reachability check:** The plugin only reports connection type, not actual internet access. Always wrap Dio calls in try/catch regardless of reported connectivity.
- **Syncing on every connectivity change without deduplication:** If connectivity flaps rapidly, multiple sync attempts can fire. Use a lock/mutex so only one sync runs at a time.
- **Full offline CRUD without server endpoints:** Keep scope narrow -- offline CREATE is the priority (PLAT-02 says "log expenses while offline"). Offline edit/delete adds complexity for limited value in v1.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Local SQLite access | Raw dart:ffi SQLite bindings | sqflite ^2.4.2 | Platform-specific initialization, WAL mode, threading handled |
| Network state detection | Manual socket probing | connectivity_plus ^7.0.0 | Handles all platforms, stream API, battery-efficient |
| UUID generation | Random string concatenation | uuid ^4.0.0 | RFC 4122 compliant, collision-resistant v4 UUIDs |
| Web SQLite | Custom IndexedDB wrapper | sqflite_common_ffi_web | Bridges sqflite API to WASM SQLite on web |

**Key insight:** The offline layer is fundamentally a local database + queue + connectivity listener. Each component has a well-maintained Flutter package. The custom work is the glue: the sync service that processes the queue and the modified notifiers that write locally first.

## Common Pitfalls

### Pitfall 1: Duplicate Expenses After Sync
**What goes wrong:** Client creates expense offline, generates UUID. On sync, server creates a new expense with a different UUID. Now two copies exist.
**Why it happens:** Server generates its own UUID via gen_random_uuid() in PostgreSQL.
**How to avoid:** Option A: Send client-generated UUID as the expense ID to the server (requires server change to accept client ID). Option B: Use a `client_id` field in the sync queue and deduplicate on the client after sync. Option A is simpler and more reliable.
**Warning signs:** Expense count doubles after going online.

### Pitfall 2: sqflite Web Initialization
**What goes wrong:** App crashes on web because sqflite doesn't work without explicit web factory setup.
**Why it happens:** sqflite doesn't natively support web; needs sqflite_common_ffi_web.
**How to avoid:** Conditionally set the database factory in main.dart based on platform:
```dart
import 'package:sqflite_common_ffi_web/sqflite_ffi_web.dart';
import 'package:flutter/foundation.dart';

if (kIsWeb) {
  databaseFactory = databaseFactoryFfiWeb;
}
```
**Warning signs:** "MissingPluginException" on web platform.

### Pitfall 3: Category References While Offline
**What goes wrong:** User tries to create expense offline but categories weren't cached locally.
**Why it happens:** Categories are fetched from server only; if first app open is offline, category list is empty.
**How to avoid:** Cache categories in sqflite when they're fetched from server. On load, show cached categories if server is unreachable.
**Warning signs:** "No categories available" message when offline.

### Pitfall 4: Sync Queue Ordering
**What goes wrong:** If a user creates then deletes an expense while offline, syncing the create followed by the delete works. But if the queue processes out of order or partially fails, state becomes inconsistent.
**How to avoid:** Process sync queue in FIFO order. For v1, limit offline operations to CREATE only (the requirement says "log expenses while offline"). Offline edit/delete can be v2.
**Warning signs:** Server has expenses the client deleted, or vice versa.

### Pitfall 5: connectivity_plus Reports WiFi But No Internet
**What goes wrong:** Plugin reports ConnectivityResult.wifi but the network has no internet (e.g., captive portal). Sync attempts fail repeatedly.
**Why it happens:** connectivity_plus checks connection type, not actual internet reachability.
**How to avoid:** Always wrap sync HTTP calls in try/catch. On DioException (timeout, connection refused), re-queue for later. Don't treat connectivity_plus as definitive.
**Warning signs:** Sync spinner never stops despite "connected" status.

## Code Examples

### Local Database Initialization
```dart
// Source: sqflite package docs + Flutter offline-first pattern
import 'package:path/path.dart';
import 'package:path_provider/path_provider.dart';
import 'package:sqflite/sqflite.dart';

class LocalDatabase {
  static Database? _db;

  static Future<Database> get database async {
    _db ??= await _initDb();
    return _db!;
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
```

### Sync Service
```dart
// Source: Flutter offline-first docs pattern adapted to project
class SyncService {
  SyncService(this._localSource, this._remoteRepo);

  final LocalExpenseSource _localSource;
  final ExpenseRepository _remoteRepo;
  bool _syncing = false;

  Future<void> syncPendingExpenses() async {
    if (_syncing) return; // Prevent concurrent syncs
    _syncing = true;

    try {
      final pending = await _localSource.getUnsyncedExpenses();
      for (final expense in pending) {
        try {
          final serverExpense = await _remoteRepo.createExpense(
            categoryId: expense.categoryId,
            amountCents: expense.amountCents,
            note: expense.note,
            expenseDate: DateFormat('yyyy-MM-dd').format(expense.expenseDate),
          );
          await _localSource.markSynced(expense.id, serverExpense.id);
        } on DioException {
          break; // Stop processing queue on network error
        }
      }
    } finally {
      _syncing = false;
    }
  }
}
```

### Modified Expense Creation Flow
```dart
// In ExpenseNotifier - local-first create
Future<bool> createExpense({
  required String categoryId,
  required int amountCents,
  required String note,
  required String expenseDate,
}) async {
  final localId = const Uuid().v4();
  final expense = Expense(
    id: localId,
    categoryId: categoryId,
    amountCents: amountCents,
    note: note,
    expenseDate: DateTime.parse(expenseDate),
  );

  // Always write locally first
  await _localSource.insertExpense(expense);

  // Update UI immediately
  final current = switch (state) {
    ExpenseLoaded(:final expenses) => expenses,
    _ => <Expense>[],
  };
  state = ExpenseLoaded([expense, ...current]);

  // Attempt remote sync (non-blocking for UI)
  _syncService.syncPendingExpenses();
  return true;
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hive for offline storage | sqflite / drift for relational data | 2024 | Hive maintainer abandoned project; sqflite actively maintained |
| Manual HTTP polling for connectivity | connectivity_plus StreamProvider | 2023 | Battery-efficient, reactive, cross-platform |
| Server-generated UUIDs only | Client-generated UUIDs for offline support | Ongoing | Enables true offline-first without server round-trip |
| connectivity_plus single ConnectivityResult | connectivity_plus List<ConnectivityResult> | v6.0.0 (2024) | Multiple simultaneous connections (WiFi + mobile) |

**Deprecated/outdated:**
- `data_connection_checker`: Abandoned, replaced by `internet_connection_checker_plus`
- `connectivity` (without _plus): Deprecated, replaced by `connectivity_plus`
- Hive: Maintainer left, no updates since 2023

## Open Questions

1. **Client-generated UUID acceptance on server**
   - What we know: Server uses `gen_random_uuid()` in PostgreSQL DEFAULT. The CreateExpense handler doesn't accept a client ID field.
   - What's unclear: Whether to modify the server to accept client-provided IDs or use a dedup approach.
   - Recommendation: Modify the server CreateExpense to accept an optional `client_id` field. If provided, use it as the expense ID. This is the simplest dedup strategy. Alternatively, the client can track its local_id-to-server_id mapping and just use the server-returned ID after sync.

2. **Scope of offline operations**
   - What we know: PLAT-02 says "log expenses while offline and they sync." Success criteria mention creating, syncing, and conflict handling.
   - What's unclear: Whether offline edit/delete is required or just offline create.
   - Recommendation: Implement offline CREATE only for v1. The requirement says "log expenses while offline." Edit/delete offline adds significant conflict complexity for minimal v1 value.

3. **Category caching depth**
   - What we know: Expense creation requires selecting a category. If categories aren't cached, offline expense creation is blocked.
   - What's unclear: Whether to cache all data (expenses, charts, family) or just categories.
   - Recommendation: Cache categories locally so the expense form works offline. Full expense list caching is a nice-to-have but not required by PLAT-02. Read operations can show "offline - pull to refresh when connected" for non-cached data.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | flutter_test + mocktail ^1.0.0 |
| Config file | app/pubspec.yaml (dev_dependencies) |
| Quick run command | `cd app && flutter test test/features/sync/` |
| Full suite command | `cd app && flutter test` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PLAT-02a | Expense saved locally when offline | unit | `cd app && flutter test test/features/sync/data/local_expense_source_test.dart -x` | Wave 0 |
| PLAT-02b | Pending expenses sync when online | unit | `cd app && flutter test test/features/sync/data/sync_service_test.dart -x` | Wave 0 |
| PLAT-02c | No duplicate expenses after sync | unit | `cd app && flutter test test/features/sync/data/sync_service_test.dart -x` | Wave 0 |
| PLAT-02d | Connectivity provider emits state changes | unit | `cd app && flutter test test/core/network/connectivity_provider_test.dart -x` | Wave 0 |
| PLAT-02e | Offline banner shows when disconnected | widget | `cd app && flutter test test/shared/widgets/offline_banner_test.dart -x` | Wave 0 |
| PLAT-02f | Category cache available offline | unit | `cd app && flutter test test/features/categories/data/local_category_source_test.dart -x` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd app && flutter test test/features/sync/`
- **Per wave merge:** `cd app && flutter test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `test/features/sync/data/local_expense_source_test.dart` -- covers PLAT-02a
- [ ] `test/features/sync/data/sync_service_test.dart` -- covers PLAT-02b, PLAT-02c
- [ ] `test/core/network/connectivity_provider_test.dart` -- covers PLAT-02d
- [ ] `test/shared/widgets/offline_banner_test.dart` -- covers PLAT-02e
- [ ] `test/features/categories/data/local_category_source_test.dart` -- covers PLAT-02f
- [ ] sqflite test setup: use `sqflite_common_ffi` for unit tests (provides in-memory SQLite)

## Sources

### Primary (HIGH confidence)
- [Flutter official offline-first docs](https://docs.flutter.dev/app-architecture/design-patterns/offline-first) - Architecture pattern, sync strategies, recommended packages
- [sqflite pub.dev](https://pub.dev/packages/sqflite) - v2.4.2, platform support, API reference
- [connectivity_plus pub.dev](https://pub.dev/packages/connectivity_plus) - v7.0.0, List<ConnectivityResult> API, platform support

### Secondary (MEDIUM confidence)
- [sqflite_common_ffi_web pub.dev](https://pub.dev/packages/sqflite_common_ffi_web) - Web support via WASM SQLite
- [Flutter offline-first DEV.to series](https://dev.to/anurag_dev/implementing-offline-first-architecture-in-flutter-part-1-local-storage-with-conflict-resolution-4mdl) - Sync queue patterns, conflict resolution approaches
- [Riverpod + connectivity_plus patterns](https://medium.com/@adanlab4/check-internet-connectivity-globally-with-connectivity-plus-and-riverpod-9e354933cff4) - StreamProvider integration

### Tertiary (LOW confidence)
- Web search results on sqflite web limitations (cross-tab safety concerns in Android Chrome)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - sqflite and connectivity_plus are the official/standard choices per Flutter docs
- Architecture: HIGH - Sync queue + local-first write is the official Flutter recommended pattern
- Pitfalls: MEDIUM - Based on community experience and package documentation; some edge cases may not be covered
- Web platform support: MEDIUM - sqflite_common_ffi_web is "experimental" per pub.dev but functional

**Research date:** 2026-03-16
**Valid until:** 2026-04-16 (stable ecosystem, 30-day validity)
