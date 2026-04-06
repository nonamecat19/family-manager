import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:flutter/material.dart';

/// Shows a modal bottom sheet with a palette of 16 color circles.
///
/// Returns the selected [Color], or `null` if dismissed.
/// The [currentColor] (if provided) shows a check mark overlay.
Future<Color?> showColorPicker(
  BuildContext context, {
  Color? currentColor,
}) {
  return showModalBottomSheet<Color>(
    context: context,
    builder: (context) => _ColorPickerContent(currentColor: currentColor),
  );
}

class _ColorPickerContent extends StatelessWidget {
  const _ColorPickerContent({this.currentColor});

  final Color? currentColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Choose color', style: theme.textTheme.titleMedium),
          const SizedBox(height: 16),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: categoryColors.map((color) {
              // ignore: deprecated_member_use
              final isSelected = currentColor?.value == color.value;
              return GestureDetector(
                onTap: () => Navigator.pop(context, color),
                child: Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: color,
                    shape: BoxShape.circle,
                    border: isSelected
                        ? Border.all(
                            color: theme.colorScheme.onSurface,
                            width: 3,
                          )
                        : null,
                  ),
                  child: isSelected
                      ? const Icon(Icons.check, color: Colors.white)
                      : null,
                ),
              );
            }).toList(),
          ),
        ],
      ),
    );
  }
}
