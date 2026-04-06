import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:flutter/material.dart';

/// Shows a modal bottom sheet with a grid of curated Material Icons.
///
/// Returns the selected icon name string, or `null` if dismissed.
/// The [currentIcon] (if provided) is highlighted with a ring.
Future<String?> showIconPicker(
  BuildContext context, {
  String? currentIcon,
}) {
  return showModalBottomSheet<String>(
    context: context,
    builder: (context) => _IconPickerContent(currentIcon: currentIcon),
  );
}

class _IconPickerContent extends StatelessWidget {
  const _IconPickerContent({this.currentIcon});

  final String? currentIcon;

  @override
  Widget build(BuildContext context) {
    final entries = categoryIcons.entries.toList();
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Choose icon', style: theme.textTheme.titleMedium),
          const SizedBox(height: 16),
          Flexible(
            child: GridView.builder(
              shrinkWrap: true,
              gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: 5,
                mainAxisSpacing: 8,
                crossAxisSpacing: 8,
              ),
              itemCount: entries.length,
              itemBuilder: (context, index) {
                final entry = entries[index];
                final isSelected = entry.key == currentIcon;

                return GestureDetector(
                  onTap: () => Navigator.pop(context, entry.key),
                  child: Container(
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: isSelected
                          ? theme.colorScheme.primaryContainer
                          : theme.colorScheme.surfaceContainerHighest,
                      border: isSelected
                          ? Border.all(
                              color: theme.colorScheme.primary,
                              width: 2,
                            )
                          : null,
                    ),
                    child: Icon(
                      entry.value,
                      color: isSelected
                          ? theme.colorScheme.primary
                          : theme.colorScheme.onSurface,
                    ),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
