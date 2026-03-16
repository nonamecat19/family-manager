/// Status of the background sync engine.
enum SyncStatus {
  /// No sync in progress.
  idle,

  /// Currently syncing pending expenses.
  syncing,

  /// Last sync attempt encountered an error.
  error,
}

/// Result of a sync attempt.
class SyncResult {
  /// Creates a [SyncResult].
  const SyncResult({this.syncedCount = 0, this.failedCount = 0});

  /// Number of expenses successfully synced to the server.
  final int syncedCount;

  /// Number of expenses that failed to sync.
  final int failedCount;

  /// Whether any expenses failed during this sync attempt.
  bool get hasFailures => failedCount > 0;
}
