import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:flutter/material.dart';

/// A list tile displaying a category with its icon in a colored circle.
///
/// Used inside [ReorderableListView] in the categories screen.
/// Shows a drag handle on the trailing side for reorder support.
class CategoryTile extends StatelessWidget {
  /// Creates a [CategoryTile] for the given [category].
  const CategoryTile({
    required this.category,
    this.onTap,
    super.key,
  });

  /// The category to display.
  final Category category;

  /// Called when the tile is tapped (navigates to edit form).
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final color = parseHexColor(category.color);
    final iconData = categoryIcons[category.icon] ?? Icons.category;

    return ListTile(
      leading: CircleAvatar(
        backgroundColor: color.withAlpha(38),
        child: Icon(iconData, color: color),
      ),
      title: Text(category.name),
      trailing: const Icon(Icons.drag_handle),
      onTap: onTap,
    );
  }
}
