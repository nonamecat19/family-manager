import 'package:finance_tracker/features/family/data/models/family.dart';
import 'package:finance_tracker/features/family/data/models/family_invitation.dart';
import 'package:finance_tracker/features/family/data/models/family_member.dart';

/// Represents the state of family group management.
sealed class FamilyState {
  const FamilyState();
}

/// Initial state before family data has been loaded.
class FamilyInitial extends FamilyState {
  const FamilyInitial();
}

/// Loading state while fetching family data from the API.
class FamilyLoading extends FamilyState {
  const FamilyLoading();
}

/// Family data successfully loaded.
class FamilyLoaded extends FamilyState {
  /// Creates a [FamilyLoaded] state.
  const FamilyLoaded(this.family, this.members, this.invitations);

  /// The user's family group.
  final Family family;

  /// Members of the family.
  final List<FamilyMember> members;

  /// Pending invitations (admin only).
  final List<FamilyInvitation> invitations;
}

/// The user does not belong to any family.
class NoFamily extends FamilyState {
  const NoFamily();
}

/// An error occurred during a family operation.
class FamilyError extends FamilyState {
  /// Creates a [FamilyError] with the given [message].
  const FamilyError(this.message);

  /// Human-readable error description.
  final String message;
}
