import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:flutter/material.dart';

/// A compact colored pill displaying a category's icon and name.
///
/// Used in expense history lists, chart legends, and anywhere a
/// category needs inline representation. The pill background uses
/// the category color at 15% opacity with icon and text in full color.
class CategoryChip extends StatelessWidget {
  /// Creates a [CategoryChip] for the given [category].
  const CategoryChip({required this.category, super.key});

  /// The category to display.
  final Category category;

  @override
  Widget build(BuildContext context) {
    final color = parseHexColor(category.color);
    final iconData = categoryIcons[category.icon] ?? Icons.category;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withAlpha(38), // ~15% opacity
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(iconData, size: 16, color: color),
          const SizedBox(width: 4),
          Text(
            category.name,
            style: TextStyle(color: color, fontWeight: FontWeight.w500),
          ),
        ],
      ),
    );
  }
}
