import 'package:finance_tracker/providers/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// Settings screen with app preferences and logout.
class SettingsScreen extends ConsumerWidget {
  /// Creates a [SettingsScreen].
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ListView(
      children: [
        const SizedBox(height: 16),
        ListTile(
          leading: const Icon(Icons.category),
          title: const Text('Categories'),
          onTap: () => context.push('/settings/categories'),
        ),
        ListTile(
          leading: const Icon(Icons.group),
          title: const Text('Family'),
          onTap: () => context.push('/settings/family'),
        ),
        ListTile(
          leading: const Icon(Icons.logout),
          title: const Text('Log Out'),
          onTap: () {
            ref.read(authStateProvider.notifier).logout();
          },
        ),
      ],
    );
  }
}
