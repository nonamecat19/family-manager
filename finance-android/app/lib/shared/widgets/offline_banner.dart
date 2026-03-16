import 'package:finance_tracker/core/network/connectivity_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Banner shown at the top of shell screens when the app is offline.
///
/// Displays "You're offline" with an optional pending expense count.
/// Collapses to [SizedBox.shrink] when online.
class OfflineBanner extends ConsumerWidget {
  /// Creates an [OfflineBanner].
  const OfflineBanner({this.pendingCount = 0, super.key});

  /// Number of expenses pending sync. Shown when > 0.
  final int pendingCount;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final connectivity = ref.watch(connectivityProvider);
    final isOffline = connectivity.valueOrNull == false;
    if (!isOffline) return const SizedBox.shrink();

    final colorScheme = Theme.of(context).colorScheme;
    return MaterialBanner(
      backgroundColor: colorScheme.surfaceContainerHighest,
      leading:
          Icon(Icons.cloud_off, color: Colors.orange.shade700, size: 24),
      content: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text("You're offline",
              style: Theme.of(context).textTheme.bodyLarge),
          if (pendingCount > 0)
            Text(
              pendingCount == 1
                  ? '1 expense will sync when connected'
                  : '$pendingCount expenses will sync when connected',
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
            ),
        ],
      ),
      actions: const [SizedBox.shrink()],
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
    );
  }
}
