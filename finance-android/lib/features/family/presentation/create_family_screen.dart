import 'package:finance_tracker/features/family/domain/family_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart' hide Family;
import 'package:go_router/go_router.dart';

/// Screen for creating a new family group.
///
/// Shows a simple form with a family name field and a create button.
class CreateFamilyScreen extends ConsumerStatefulWidget {
  /// Creates a [CreateFamilyScreen].
  const CreateFamilyScreen({super.key});

  @override
  ConsumerState<CreateFamilyScreen> createState() =>
      _CreateFamilyScreenState();
}

class _CreateFamilyScreenState
    extends ConsumerState<CreateFamilyScreen> {
  final _formKey = GlobalKey<FormState>();
  final _nameController = TextEditingController();
  bool _submitting = false;

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);

    await ref
        .read(familyStateProvider.notifier)
        .createFamily(_nameController.text.trim());

    if (mounted) {
      context.go('/settings/family');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Create Family')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Form(
            key: _formKey,
            child: TextFormField(
              controller: _nameController,
              decoration: const InputDecoration(
                labelText: 'Family name',
                hintText: 'e.g. The Smiths',
              ),
              validator: (value) {
                if (value == null || value.trim().isEmpty) {
                  return 'Please enter a family name';
                }
                return null;
              },
            ),
          ),
          const SizedBox(height: 32),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: _submitting ? null : _submit,
              child: const Text('Create Family'),
            ),
          ),
        ],
      ),
    );
  }
}
