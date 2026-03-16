import 'package:finance_tracker/features/auth/domain/auth_state.dart';
import 'package:finance_tracker/features/family/data/models/family_member.dart';
import 'package:finance_tracker/features/family/domain/family_notifier.dart';
import 'package:finance_tracker/features/family/domain/family_state.dart';
import 'package:finance_tracker/providers/auth_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart'
    hide Family;
import 'package:go_router/go_router.dart';

/// Screen for managing the user's family group.
///
/// Shows different views based on [FamilyState]:
/// - Loading: progress indicator
/// - Error: error message
/// - NoFamily: empty state with create/join options
/// - FamilyLoaded: admin or member view with member list
class FamilyScreen extends ConsumerStatefulWidget {
  /// Creates a [FamilyScreen].
  const FamilyScreen({super.key});

  @override
  ConsumerState<FamilyScreen> createState() => _FamilyScreenState();
}

class _FamilyScreenState extends ConsumerState<FamilyScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(
      () => ref.read(familyStateProvider.notifier).loadFamily(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final familyState = ref.watch(familyStateProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Family')),
      body: switch (familyState) {
        FamilyInitial() || FamilyLoading() => const Center(
            child: CircularProgressIndicator(),
          ),
        FamilyError(message: final msg) => Center(
            child: Text(msg),
          ),
        NoFamily() => _buildEmptyState(context),
        FamilyLoaded() => _buildLoadedState(context, familyState),
      },
    );
  }

  Widget _buildEmptyState(BuildContext context) {
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
                  Icons.group_add,
                  size: 48,
                  color: Theme.of(context)
                      .colorScheme
                      .onSurfaceVariant,
                ),
                const SizedBox(height: 16),
                Text(
                  'No family yet',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 8),
                Text(
                  'Create a family group to share expense '
                  'tracking with your household.',
                  style: Theme.of(context).textTheme.bodyMedium,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: () =>
                      context.push('/settings/family/create'),
                  child: const Text('Create Family'),
                ),
                const SizedBox(height: 8),
                TextButton(
                  onPressed: () => _showInviteCodeDialog(context),
                  child: const Text('Have an invite code?'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _showInviteCodeDialog(BuildContext context) async {
    final controller = TextEditingController();
    final result = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Enter Invite Code'),
        content: TextField(
          controller: controller,
          decoration: const InputDecoration(
            hintText: 'Paste the invite code here',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Dismiss'),
          ),
          TextButton(
            onPressed: () =>
                Navigator.pop(context, controller.text.trim()),
            child: const Text('Join Family'),
          ),
        ],
      ),
    );
    controller.dispose();

    if (result != null && result.isNotEmpty && mounted) {
      await ref
          .read(familyStateProvider.notifier)
          .acceptInvitation(result);
    }
  }

  Widget _buildLoadedState(
    BuildContext context,
    FamilyLoaded state,
  ) {
    final authState = ref.watch(authStateProvider);
    final currentUserId =
        authState is Authenticated ? authState.userId : '';
    final isAdmin = state.family.adminUserId == currentUserId;
    final theme = Theme.of(context);

    return ListView(
      padding: const EdgeInsets.only(top: 16, bottom: 16),
      children: [
        // Family info card.
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Card(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    state.family.name,
                    style: theme.textTheme.titleLarge,
                  ),
                  Text(
                    '${state.members.length} members',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
        const SizedBox(height: 16),

        // Members section header.
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Text(
            'Members',
            style: theme.textTheme.titleMedium,
          ),
        ),
        const SizedBox(height: 8),

        // Member list.
        ...state.members.map(
          (member) => ListTile(
            leading: CircleAvatar(
              child: Text(
                member.email.isNotEmpty
                    ? member.email[0].toUpperCase()
                    : '?',
              ),
            ),
            title: Text(
              member.email,
              style: theme.textTheme.bodyLarge,
            ),
            subtitle: Text(
              member.role,
              style: theme.textTheme.bodyMedium,
            ),
            trailing: isAdmin && member.userId != currentUserId
                ? IconButton(
                    icon: const Icon(Icons.remove_circle_outline),
                    color: theme.colorScheme.error,
                    tooltip: 'Remove member',
                    onPressed: () =>
                        _showRemoveMemberDialog(context, member),
                  )
                : null,
          ),
        ),
        const SizedBox(height: 16),

        // Admin-only: Pending invitations section.
        if (isAdmin) ...[
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Text(
              'Pending Invitations',
              style: theme.textTheme.titleMedium,
            ),
          ),
          const SizedBox(height: 8),
          ...state.invitations.asMap().entries.map(
                (entry) => ListTile(
                  leading: const Icon(Icons.mail_outline),
                  title: Text(
                    'Invite #${entry.key + 1}',
                    style: theme.textTheme.bodyLarge,
                  ),
                  subtitle: Text(
                    'Expires ${_formatDate(entry.value.expiresAt)}',
                    style: theme.textTheme.bodyMedium,
                  ),
                  trailing: IconButton(
                    icon: const Icon(Icons.close),
                    color: theme.colorScheme.error,
                    tooltip: 'Revoke invitation',
                    onPressed: () => ref
                        .read(familyStateProvider.notifier)
                        .revokeInvitation(entry.value.id),
                  ),
                ),
              ),
          const SizedBox(height: 16),

          // Copy Invite Link button.
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                icon: const Icon(Icons.link),
                label: const Text('Copy Invite Link'),
                onPressed: () => _copyInviteLink(context),
              ),
            ),
          ),
        ],
        const SizedBox(height: 32),

        // Leave Family button.
        Center(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: TextButton(
              onPressed: () => _showLeaveDialog(
                context,
                state.family.name,
                isAdmin,
              ),
              child: Text(
                'Leave Family',
                style: TextStyle(
                  color: theme.colorScheme.error,
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }

  String _formatDate(DateTime date) {
    return '${date.year}-${date.month.toString().padLeft(2, '0')}'
        '-${date.day.toString().padLeft(2, '0')}';
  }

  Future<void> _showRemoveMemberDialog(
    BuildContext context,
    FamilyMember member,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('Remove ${member.email}?'),
        content: const Text(
          'This person will be removed from the family group. '
          'Their expenses will stay with them.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Keep Member'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(
              'Remove',
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
              ),
            ),
          ),
        ],
      ),
    );
    if (confirmed ?? false) {
      if (!mounted) return;
      await ref
          .read(familyStateProvider.notifier)
          .removeMember(member.userId);
    }
  }

  Future<void> _copyInviteLink(BuildContext context) async {
    final token = await ref
        .read(familyStateProvider.notifier)
        .createInvitation();
    if (token != null && mounted) {
      await Clipboard.setData(
        ClipboardData(text: 'financetracker://invite/$token'),
      );
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Invite link copied')),
        );
      }
    }
  }

  Future<void> _showLeaveDialog(
    BuildContext context,
    String familyName,
    bool isAdmin,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(
          isAdmin ? 'Delete $familyName?' : 'Leave $familyName?',
        ),
        content: Text(
          isAdmin
              ? 'As the admin, leaving will dissolve the family '
                  'group and remove all members.'
              : 'You will be removed from this family group. '
                  'Your expenses will stay with your account.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(isAdmin ? 'Stay' : 'Stay in Family'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(
              isAdmin ? 'Delete Family' : 'Leave',
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
              ),
            ),
          ),
        ],
      ),
    );

    if (!(confirmed ?? false)) return;
    if (!mounted) return;
    if (isAdmin) {
      await ref.read(familyStateProvider.notifier).deleteFamily();
    } else {
      await ref.read(familyStateProvider.notifier).leaveFamily();
    }
    if (mounted) {
      context.pop();
    }
  }
}
