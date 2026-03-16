import 'package:dio/dio.dart';
import 'package:finance_tracker/features/family/data/family_repository.dart';
import 'package:finance_tracker/features/family/domain/family_notifier.dart';
import 'package:finance_tracker/features/family/domain/family_state.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart'
    hide Family;
import 'package:go_router/go_router.dart';

/// Screen for accepting a family invitation via deep link.
///
/// Takes a [token] from the route path and shows a preview
/// of the family to join, an error state, or a message if
/// the user is already in a family.
class AcceptInviteScreen extends ConsumerStatefulWidget {
  /// Creates an [AcceptInviteScreen] with the given [token].
  const AcceptInviteScreen({required this.token, super.key});

  /// The invitation token from the deep link path.
  final String token;

  @override
  ConsumerState<AcceptInviteScreen> createState() =>
      _AcceptInviteScreenState();
}

enum _InviteViewState { loading, preview, error, alreadyInFamily }

class _AcceptInviteScreenState
    extends ConsumerState<AcceptInviteScreen> {
  _InviteViewState _viewState = _InviteViewState.loading;
  String _familyName = '';
  bool _joining = false;

  @override
  void initState() {
    super.initState();
    Future.microtask(_loadInviteInfo);
  }

  Future<void> _loadInviteInfo() async {
    try {
      // Check if user is already in a family.
      final familyState = ref.read(familyStateProvider);
      if (familyState is FamilyLoaded) {
        if (mounted) {
          setState(
            () => _viewState = _InviteViewState.alreadyInFamily,
          );
        }
        return;
      }

      final repo = ref.read(familyRepositoryProvider);
      final info = await repo.getInvitationInfo(widget.token);
      if (mounted) {
        setState(() {
          _familyName = info['family_name'] as String? ?? '';
          _viewState = _InviteViewState.preview;
        });
      }
    } on DioException {
      if (mounted) {
        setState(() => _viewState = _InviteViewState.error);
      }
    } on Exception {
      if (mounted) {
        setState(() => _viewState = _InviteViewState.error);
      }
    }
  }

  Future<void> _joinFamily() async {
    setState(() => _joining = true);
    await ref
        .read(familyStateProvider.notifier)
        .acceptInvitation(widget.token);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Welcome to $_familyName!'),
        ),
      );
      context.go('/settings/family');
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('Join Family')),
      body: switch (_viewState) {
        _InviteViewState.loading => const Center(
            child: CircularProgressIndicator(),
          ),
        _InviteViewState.preview => _buildPreview(theme),
        _InviteViewState.error => _buildError(theme),
        _InviteViewState.alreadyInFamily =>
          _buildAlreadyInFamily(theme),
      },
    );
  }

  Widget _buildPreview(ThemeData theme) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Card(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.group,
                  size: 48,
                  color: theme.colorScheme.primary,
                ),
                const SizedBox(height: 16),
                Text(
                  "You're invited!",
                  style: theme.textTheme.titleLarge,
                ),
                const SizedBox(height: 8),
                Text(
                  'Join $_familyName',
                  style: theme.textTheme.bodyLarge,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: _joining ? null : _joinFamily,
                    child: const Text('Join Family'),
                  ),
                ),
                const SizedBox(height: 8),
                TextButton(
                  onPressed: () => context.pop(),
                  child: const Text('Decline Invite'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildError(ThemeData theme) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              'Invalid Invitation',
              style: theme.textTheme.titleLarge,
            ),
            const SizedBox(height: 8),
            Text(
              'This invite link has expired or '
              'already been used.',
              style: theme.textTheme.bodyMedium,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: () => context.pop(),
              child: const Text('Go Back'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildAlreadyInFamily(ThemeData theme) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              "You're already in a family. Leave your "
              'current family first to join a new one.',
              style: theme.textTheme.bodyMedium,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: () => context.pop(),
              child: const Text('Go Back'),
            ),
          ],
        ),
      ),
    );
  }
}
