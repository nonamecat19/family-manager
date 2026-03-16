import 'package:finance_tracker/core/network/connectivity_provider.dart';
import 'package:finance_tracker/features/sync/data/sync_service.dart';
import 'package:finance_tracker/shared/widgets/offline_banner.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// Provides the current count of unsynced expenses from the local DB.
///
/// Autodispose so it re-fetches each time the scaffold rebuilds.
final unsyncedCountProvider = FutureProvider.autoDispose<int>((ref) async {
  return ref.read(localExpenseSourceProvider).getUnsyncedCount();
});

/// The app's main scaffold with bottom navigation, FAB, and offline banner.
///
/// Wraps each tab screen with a [NavigationBar] containing 3 destinations
/// (History, Charts, Settings), a [FloatingActionButton], and an
/// [OfflineBanner] that appears when offline with the pending sync count.
class AppScaffold extends ConsumerWidget {
  /// Creates an [AppScaffold].
  ///
  /// [selectedIndex] determines which tab is highlighted.
  /// [child] is the screen content for the selected tab.
  const AppScaffold({
    required this.selectedIndex,
    required this.child,
    super.key,
  });

  /// The index of the currently selected tab.
  final int selectedIndex;

  /// The screen content displayed in the body.
  final Widget child;

  static const _destinations = [
    NavigationDestination(
      icon: Icon(Icons.receipt_long),
      label: 'History',
    ),
    NavigationDestination(
      icon: Icon(Icons.pie_chart),
      label: 'Charts',
    ),
    NavigationDestination(
      icon: Icon(Icons.settings),
      label: 'Settings',
    ),
  ];

  static const _routes = ['/history', '/charts', '/settings'];

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Read pending expense count for the offline banner.
    final pendingCount = ref.watch(unsyncedCountProvider).valueOrNull ?? 0;

    // Listen for connectivity restore to trigger sync.
    ref.listen<AsyncValue<bool>>(connectivityProvider, (prev, next) {
      final wasOffline = prev?.valueOrNull == false;
      final isOnline = next.valueOrNull == true;
      if (wasOffline && isOnline) {
        ref.read(syncServiceProvider).syncPendingExpenses().then((result) {
          if (!context.mounted) return;
          // Invalidate the unsynced count so banner updates after sync.
          ref.invalidate(unsyncedCountProvider);
          if (result.syncedCount > 0) {
            ScaffoldMessenger.of(context).showSnackBar(SnackBar(
              content: Row(children: [
                Icon(Icons.check_circle,
                    color: Theme.of(context).colorScheme.primary, size: 20),
                const SizedBox(width: 8),
                Text(result.syncedCount == 1
                    ? '1 expense synced'
                    : '${result.syncedCount} expenses synced'),
              ]),
              duration: const Duration(seconds: 3),
              behavior: SnackBarBehavior.floating,
            ));
          }
          if (result.hasFailures) {
            ScaffoldMessenger.of(context).showSnackBar(SnackBar(
              content: Row(children: [
                Icon(Icons.error_outline,
                    color: Colors.orange.shade700, size: 20),
                const SizedBox(width: 8),
                const Text(
                    "Some expenses couldn't sync. Will retry automatically."),
              ]),
              duration: const Duration(seconds: 4),
              behavior: SnackBarBehavior.floating,
            ));
          }
        });
      }
    });

    return Scaffold(
      body: Column(
        children: [
          OfflineBanner(pendingCount: pendingCount),
          Expanded(child: child),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        elevation: 0,
        highlightElevation: 0,
        onPressed: () => context.push('/expenses/new'),
        child: const Icon(Icons.add),
      ),
      bottomNavigationBar: NavigationBar(
        elevation: 0,
        shadowColor: Colors.transparent,
        selectedIndex: selectedIndex,
        onDestinationSelected: (index) => context.go(_routes[index]),
        destinations: _destinations,
      ),
    );
  }
}
