import 'package:finance_tracker/features/categories/data/models/category.dart'
    as models;
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:finance_tracker/features/categories/presentation/widgets/category_tile.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Key for the shared preferences flag that tracks whether the
/// starter categories prompt has been dismissed.
const _starterDismissedKey = 'categories_starter_dismissed';

/// Screen displaying the user's categories with drag-to-reorder
/// and swipe-to-delete support.
///
/// Accessed via Settings > Categories. Shows a one-time starter
/// prompt when the list is empty and the user hasn't dismissed it.
class CategoriesScreen extends ConsumerStatefulWidget {
  /// Creates a [CategoriesScreen].
  const CategoriesScreen({super.key});

  @override
  ConsumerState<CategoriesScreen> createState() => _CategoriesScreenState();
}

class _CategoriesScreenState extends ConsumerState<CategoriesScreen> {
  bool _starterDismissed = true;

  @override
  void initState() {
    super.initState();
    // Load categories on first build.
    Future.microtask(
      () => ref.read(categoryStateProvider.notifier).loadCategories(),
    );
    _loadStarterFlag();
  }

  Future<void> _loadStarterFlag() async {
    final prefs = await SharedPreferences.getInstance();
    if (mounted) {
      setState(() {
        _starterDismissed =
            prefs.getBool(_starterDismissedKey) ?? false;
      });
    }
  }

  Future<void> _dismissStarter() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(_starterDismissedKey, true);
    if (mounted) {
      setState(() => _starterDismissed = true);
    }
  }

  Future<void> _addStarters() async {
    await ref.read(categoryStateProvider.notifier).bulkCreateStarters();
    await _dismissStarter();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(categoryStateProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Categories')),
      floatingActionButton: FloatingActionButton(
        onPressed: () => context.push('/settings/categories/new'),
        child: const Icon(Icons.add),
      ),
      body: switch (state) {
        CategoryInitial() || CategoryLoading() => const Center(
            child: CircularProgressIndicator(),
          ),
        CategoryError(message: final msg) => Center(
            child: Text(msg),
          ),
        CategoryLoaded(categories: final categories) =>
          _buildList(context, categories),
      },
    );
  }

  Widget _buildList(BuildContext context, List<models.Category> categories) {
    if (categories.isEmpty && !_starterDismissed) {
      return _StarterPrompt(
        onAdd: _addStarters,
        onSkip: _dismissStarter,
      );
    }

    if (categories.isEmpty) {
      return const Center(
        child: Text('No categories yet. Tap + to create one.'),
      );
    }

    return ReorderableListView.builder(
      itemCount: categories.length,
      onReorderItem: (oldIndex, newIndex) {
        ref
            .read(categoryStateProvider.notifier)
            .reorderCategories(oldIndex, newIndex);
      },
      itemBuilder: (context, index) {
        final category = categories[index];
        return Dismissible(
          key: ValueKey(category.id),
          direction: DismissDirection.endToStart,
          background: Container(
            alignment: Alignment.centerRight,
            padding: const EdgeInsets.only(right: 16),
            color: Colors.red,
            child: const Icon(Icons.delete, color: Colors.white),
          ),
          confirmDismiss: (_) => _confirmDelete(context, category.name),
          onDismissed: (_) {
            ref
                .read(categoryStateProvider.notifier)
                .deleteCategory(category.id);
          },
          child: CategoryTile(
            category: category,
            onTap: () => context.push(
              '/settings/categories/edit',
              extra: category,
            ),
          ),
        );
      },
    );
  }

  Future<bool> _confirmDelete(BuildContext context, String name) async {
    final result = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('Delete $name?'),
        content: const Text(
          'This category will be permanently deleted.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    return result ?? false;
  }
}

class _StarterPrompt extends StatelessWidget {
  const _StarterPrompt({required this.onAdd, required this.onSkip});

  final VoidCallback onAdd;
  final VoidCallback onSkip;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Card(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  'Get started quickly?',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 8),
                const Text(
                  'We can add 6 common categories to get you started. '
                  'You can always edit or delete them later.',
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: onAdd,
                  child: const Text('Add these'),
                ),
                const SizedBox(height: 8),
                TextButton(
                  onPressed: onSkip,
                  child: const Text('Skip'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
