import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:finance_tracker/features/categories/presentation/widgets/color_picker_sheet.dart';
import 'package:finance_tracker/features/categories/presentation/widgets/icon_picker_sheet.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Screen for creating or editing a category.
///
/// In create mode ([category] is null), the form starts empty.
/// In edit mode ([category] is provided), the form is pre-filled
/// with the category's current values.
class CategoryFormScreen extends ConsumerStatefulWidget {
  /// Creates a [CategoryFormScreen].
  ///
  /// Pass [category] for edit mode; omit for create mode.
  const CategoryFormScreen({this.category, super.key});

  /// The category to edit, or null for create mode.
  final Category? category;

  @override
  ConsumerState<CategoryFormScreen> createState() =>
      _CategoryFormScreenState();
}

class _CategoryFormScreenState extends ConsumerState<CategoryFormScreen> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameController;

  String? _selectedIcon;
  Color? _selectedColor;
  bool _submitted = false;

  bool get _isEditing => widget.category != null;

  @override
  void initState() {
    super.initState();
    _nameController =
        TextEditingController(text: widget.category?.name ?? '');
    _selectedIcon = widget.category?.icon;
    if (widget.category != null) {
      _selectedColor = parseHexColor(widget.category!.color);
    }
  }

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _pickIcon() async {
    final icon = await showIconPicker(context, currentIcon: _selectedIcon);
    if (icon != null && mounted) {
      setState(() => _selectedIcon = icon);
    }
  }

  Future<void> _pickColor() async {
    final color =
        await showColorPicker(context, currentColor: _selectedColor);
    if (color != null && mounted) {
      setState(() => _selectedColor = color);
    }
  }

  Future<void> _submit() async {
    setState(() => _submitted = true);

    if (!_formKey.currentState!.validate() ||
        _selectedIcon == null ||
        _selectedColor == null) {
      return;
    }

    final name = _nameController.text.trim();
    final icon = _selectedIcon!;
    final color = colorToHex(_selectedColor!);

    final notifier = ref.read(categoryStateProvider.notifier);

    if (_isEditing) {
      await notifier.updateCategory(
        id: widget.category!.id,
        name: name,
        icon: icon,
        color: color,
      );
    } else {
      await notifier.createCategory(name: name, icon: icon, color: color);
    }

    if (mounted) Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(_isEditing ? 'Edit Category' : 'New Category'),
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextFormField(
              controller: _nameController,
              decoration: const InputDecoration(
                labelText: 'Name',
                hintText: 'e.g. Groceries',
              ),
              textCapitalization: TextCapitalization.sentences,
              validator: (value) {
                if (value == null || value.trim().isEmpty) {
                  return 'Name is required';
                }
                return null;
              },
            ),
            const SizedBox(height: 24),
            // Icon picker row
            _PickerRow(
              label: 'Icon',
              onTap: _pickIcon,
              error: _submitted && _selectedIcon == null
                  ? 'Icon is required'
                  : null,
              child: _selectedIcon != null
                  ? Icon(
                      categoryIcons[_selectedIcon!] ?? Icons.category,
                      color: _selectedColor ?? theme.colorScheme.primary,
                    )
                  : Text(
                      'Choose icon',
                      style: TextStyle(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
            ),
            const SizedBox(height: 16),
            // Color picker row
            _PickerRow(
              label: 'Color',
              onTap: _pickColor,
              error: _submitted && _selectedColor == null
                  ? 'Color is required'
                  : null,
              child: _selectedColor != null
                  ? Container(
                      width: 24,
                      height: 24,
                      decoration: BoxDecoration(
                        color: _selectedColor,
                        shape: BoxShape.circle,
                      ),
                    )
                  : Text(
                      'Choose color',
                      style: TextStyle(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
            ),
            const SizedBox(height: 32),
            FilledButton(
              onPressed: _submit,
              child: Text(
                _isEditing ? 'Save Changes' : 'Create Category',
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A tappable row used for the icon and color picker fields.
class _PickerRow extends StatelessWidget {
  const _PickerRow({
    required this.label,
    required this.onTap,
    required this.child,
    this.error,
  });

  final String label;
  final VoidCallback onTap;
  final Widget child;
  final String? error;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(8),
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 12),
            child: Row(
              children: [
                Text(
                  label,
                  style: Theme.of(context).textTheme.bodyLarge,
                ),
                const Spacer(),
                child,
                const SizedBox(width: 8),
                const Icon(Icons.chevron_right),
              ],
            ),
          ),
        ),
        if (error != null)
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Text(
              error!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
                fontSize: 12,
              ),
            ),
          ),
      ],
    );
  }
}
