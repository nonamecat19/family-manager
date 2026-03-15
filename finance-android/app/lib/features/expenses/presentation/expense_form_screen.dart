import 'package:finance_tracker/features/categories/data/category_colors.dart';
import 'package:finance_tracker/features/categories/data/category_icons.dart';
import 'package:finance_tracker/features/categories/data/models/category.dart';
import 'package:finance_tracker/features/categories/domain/category_state.dart';
import 'package:finance_tracker/features/expenses/data/models/expense.dart';
import 'package:finance_tracker/providers/category_provider.dart';
import 'package:finance_tracker/providers/expense_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

/// Full-screen form for creating a new expense.
///
/// Optimized for quick entry: amount auto-focuses with numpad,
/// category chips are tappable, and save completes in 3 taps
/// after FAB (amount -> chip -> save).
class ExpenseFormScreen extends ConsumerStatefulWidget {
  /// Creates an [ExpenseFormScreen].
  const ExpenseFormScreen({super.key});

  @override
  ConsumerState<ExpenseFormScreen> createState() => _ExpenseFormScreenState();
}

class _ExpenseFormScreenState extends ConsumerState<ExpenseFormScreen> {
  final _amountController = TextEditingController();
  final _noteController = TextEditingController();

  String? _selectedCategoryId;
  late DateTime _selectedDate;
  bool _submitted = false;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _selectedDate = DateTime.now();

    // Ensure categories are loaded for the picker.
    final catState = ref.read(categoryStateProvider);
    if (catState is CategoryInitial) {
      Future.microtask(
        () => ref.read(categoryStateProvider.notifier).loadCategories(),
      );
    }
  }

  @override
  void dispose() {
    _amountController.dispose();
    _noteController.dispose();
    super.dispose();
  }

  String? _amountError() {
    if (!_submitted) return null;
    final text = _amountController.text.trim();
    if (text.isEmpty) return 'Amount is required';
    if (Expense.parseAmountToCents(text) == null) return 'Enter a valid amount';
    return null;
  }

  String? _categoryError() {
    if (!_submitted) return null;
    if (_selectedCategoryId == null) return 'Select a category';
    return null;
  }

  Future<void> _pickDate() async {
    final now = DateTime.now();
    final picked = await showDatePicker(
      context: context,
      initialDate: _selectedDate,
      firstDate: now.subtract(const Duration(days: 365)),
      lastDate: now,
    );
    if (picked != null && mounted) {
      setState(() => _selectedDate = picked);
    }
  }

  Future<void> _submit() async {
    setState(() => _submitted = true);

    final amountCents =
        Expense.parseAmountToCents(_amountController.text.trim());
    if (amountCents == null || _selectedCategoryId == null) return;

    setState(() => _saving = true);

    final dateStr = DateFormat('yyyy-MM-dd').format(_selectedDate);
    final success =
        await ref.read(expenseStateProvider.notifier).createExpense(
              categoryId: _selectedCategoryId!,
              amountCents: amountCents,
              note: _noteController.text.trim(),
              expenseDate: dateStr,
            );

    if (!mounted) return;
    setState(() => _saving = false);

    if (success) {
      Navigator.pop(context);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Expense saved')),
      );
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to save expense')),
      );
    }
  }

  String _formatDate(DateTime date) {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final selected = DateTime(date.year, date.month, date.day);
    if (selected == today) return 'Today';
    return DateFormat.yMMMd().format(date);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final catState = ref.watch(categoryStateProvider);
    final categories =
        catState is CategoryLoaded ? catState.categories : <Category>[];

    return Scaffold(
      appBar: AppBar(title: const Text('New Expense')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // --- Amount field ---
          TextField(
            controller: _amountController,
            autofocus: true,
            keyboardType:
                const TextInputType.numberWithOptions(decimal: true),
            decoration: InputDecoration(
              labelText: 'Amount',
              prefixText: r'$ ',
              errorText: _amountError(),
            ),
            onChanged: (_) {
              if (_submitted) setState(() {});
            },
          ),
          const SizedBox(height: 24),

          // --- Category picker ---
          Text('Category', style: theme.textTheme.bodyLarge),
          const SizedBox(height: 8),
          SizedBox(
            height: 40,
            child: categories.isEmpty
                ? Text(
                    'No categories available',
                    style: TextStyle(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  )
                : SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: Row(
                      children: [
                        for (final cat in categories)
                          Padding(
                            padding: const EdgeInsets.only(right: 8),
                            child: _SelectableCategoryChip(
                              category: cat,
                              isSelected: _selectedCategoryId == cat.id,
                              onTap: () {
                                setState(
                                  () => _selectedCategoryId = cat.id,
                                );
                              },
                            ),
                          ),
                      ],
                    ),
                  ),
          ),
          if (_categoryError() != null)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                _categoryError()!,
                style: TextStyle(
                  color: theme.colorScheme.error,
                  fontSize: 12,
                ),
              ),
            ),
          const SizedBox(height: 24),

          // --- Note field ---
          TextField(
            controller: _noteController,
            decoration: const InputDecoration(
              labelText: 'Note (optional)',
            ),
            textCapitalization: TextCapitalization.sentences,
          ),
          const SizedBox(height: 24),

          // --- Date picker ---
          InkWell(
            onTap: _pickDate,
            borderRadius: BorderRadius.circular(8),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Row(
                children: [
                  const Icon(Icons.calendar_today, size: 20),
                  const SizedBox(width: 12),
                  Text(
                    _formatDate(_selectedDate),
                    style: theme.textTheme.bodyLarge,
                  ),
                  const Spacer(),
                  const Icon(Icons.chevron_right),
                ],
              ),
            ),
          ),
          const SizedBox(height: 32),

          // --- Save button ---
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: _saving ? null : _submit,
              child: _saving
                  ? const SizedBox(
                      height: 20,
                      width: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Save'),
            ),
          ),
        ],
      ),
    );
  }
}

/// A category chip that shows selected state with a border.
class _SelectableCategoryChip extends StatelessWidget {
  const _SelectableCategoryChip({
    required this.category,
    required this.isSelected,
    required this.onTap,
  });

  final Category category;
  final bool isSelected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final color = parseHexColor(category.color);
    final iconData = categoryIcons[category.icon] ?? Icons.category;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
        decoration: BoxDecoration(
          color: isSelected ? color.withAlpha(51) : color.withAlpha(26),
          borderRadius: BorderRadius.circular(16),
          border: isSelected
              ? Border.all(color: color, width: 2)
              : Border.all(color: Colors.transparent, width: 2),
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
      ),
    );
  }
}
