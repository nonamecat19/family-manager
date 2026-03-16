import 'package:dio/dio.dart';
import 'package:finance_tracker/features/family/data/family_repository.dart';
import 'package:finance_tracker/features/family/data/models/family.dart';
import 'package:finance_tracker/features/family/data/models/family_invitation.dart';
import 'package:finance_tracker/features/family/data/models/family_member.dart';
import 'package:finance_tracker/features/family/domain/family_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart' hide Family;

/// Manages family group state for the application.
///
/// Handles family CRUD, member management, and invitation lifecycle
/// via [FamilyRepository].
class FamilyNotifier extends StateNotifier<FamilyState> {
  /// Creates a [FamilyNotifier].
  FamilyNotifier(this._repository) : super(const FamilyInitial());

  final FamilyRepository _repository;

  /// Loads the current user's family data.
  ///
  /// Sets [FamilyLoaded] on success, [NoFamily] on 404,
  /// [FamilyError] otherwise.
  Future<void> loadFamily() async {
    state = const FamilyLoading();
    try {
      final data = await _repository.getMyFamily();
      final family =
          Family.fromJson(data['family'] as Map<String, dynamic>);
      final members = (data['members'] as List<dynamic>)
          .cast<Map<String, dynamic>>()
          .map(FamilyMember.fromJson)
          .toList();
      final invitations = (data['invitations'] as List<dynamic>? ?? [])
          .cast<Map<String, dynamic>>()
          .map(FamilyInvitation.fromJson)
          .toList();
      state = FamilyLoaded(family, members, invitations);
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) {
        state = const NoFamily();
      } else {
        state = FamilyError(e.message ?? 'Failed to load family');
      }
    } on Exception catch (e) {
      state = FamilyError(e.toString());
    }
  }

  /// Creates a new family group, then reloads.
  Future<void> createFamily(String name) async {
    try {
      await _repository.createFamily(name);
      await loadFamily();
    } on Exception catch (e) {
      state = FamilyError(e.toString());
    }
  }

  /// Deletes the family group (admin only).
  Future<void> deleteFamily() async {
    try {
      await _repository.deleteFamily();
      state = const NoFamily();
    } on Exception catch (e) {
      state = FamilyError(e.toString());
    }
  }

  /// Removes a member from the family (admin only), then reloads.
  Future<void> removeMember(String userId) async {
    try {
      await _repository.removeMember(userId);
      await loadFamily();
    } on Exception catch (e) {
      state = FamilyError(e.toString());
    }
  }

  /// Leaves the current family.
  Future<void> leaveFamily() async {
    try {
      await _repository.leaveFamily();
      state = const NoFamily();
    } on Exception catch (e) {
      state = FamilyError(e.toString());
    }
  }

  /// Creates a new invitation and returns the raw token.
  ///
  /// Reloads family data to refresh the invitation list.
  Future<String?> createInvitation() async {
    try {
      final token = await _repository.createInvitation();
      await loadFamily();
      return token;
    } on Exception catch (e) {
      state = FamilyError(e.toString());
      return null;
    }
  }

  /// Revokes a pending invitation (admin only), then reloads.
  Future<void> revokeInvitation(String id) async {
    try {
      await _repository.revokeInvitation(id);
      await loadFamily();
    } on Exception catch (e) {
      state = FamilyError(e.toString());
    }
  }

  /// Accepts an invitation, then loads the family.
  Future<void> acceptInvitation(String token) async {
    try {
      await _repository.acceptInvitation(token);
      await loadFamily();
    } on Exception catch (e) {
      state = FamilyError(e.toString());
    }
  }
}

/// Provides the family state and notifier.
final familyStateProvider =
    StateNotifierProvider<FamilyNotifier, FamilyState>((ref) {
  return FamilyNotifier(ref.read(familyRepositoryProvider));
});

/// A fake [FamilyNotifier] for widget test mocking.
///
/// All methods are no-ops. Override [state] in tests to control UI.
class FakeFamilyNotifier extends StateNotifier<FamilyState>
    implements FamilyNotifier {
  /// Creates a [FakeFamilyNotifier] with the given initial [state].
  FakeFamilyNotifier([super.initial = const FamilyInitial()]);

  @override
  // ignore: unused_field
  FamilyRepository get _repository => throw UnimplementedError();

  @override
  Future<void> loadFamily() async {}

  @override
  Future<void> createFamily(String name) async {}

  @override
  Future<void> deleteFamily() async {}

  @override
  Future<void> removeMember(String userId) async {}

  @override
  Future<void> leaveFamily() async {}

  @override
  Future<String?> createInvitation() async => null;

  @override
  Future<void> revokeInvitation(String id) async {}

  @override
  Future<void> acceptInvitation(String token) async {}
}
