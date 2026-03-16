import 'package:dio/dio.dart';
import 'package:finance_tracker/core/network/api_client.dart';
import 'package:finance_tracker/features/family/data/models/family.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart' hide Family;

/// Repository for family group API calls.
///
/// Communicates with the Go backend family endpoints
/// at `/families` and `/invitations`.
class FamilyRepository {
  /// Creates a [FamilyRepository] with the given [Dio] client.
  const FamilyRepository(this._dio);

  final Dio _dio;

  /// Fetches the current user's family with members and invitations.
  ///
  /// Returns raw JSON map with `family`, `members`, and `invitations` keys.
  Future<Map<String, dynamic>> getMyFamily() async {
    final response =
        await _dio.get<Map<String, dynamic>>('/families/me');
    return response.data!;
  }

  /// Creates a new family with the given [name].
  Future<Family> createFamily(String name) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/families',
      data: {'name': name},
    );
    return Family.fromJson(response.data!);
  }

  /// Deletes the current user's family (admin only).
  Future<void> deleteFamily() async {
    await _dio.delete<void>('/families/me');
  }

  /// Removes a member from the family by their user ID (admin only).
  Future<void> removeMember(String userId) async {
    await _dio.delete<void>('/families/me/members/$userId');
  }

  /// Leaves the current family.
  Future<void> leaveFamily() async {
    await _dio.post<void>('/families/me/leave');
  }

  /// Creates a new invitation and returns the raw token string.
  Future<String> createInvitation() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/families/me/invitations',
    );
    return response.data!['token'] as String;
  }

  /// Revokes a pending invitation by ID (admin only).
  Future<void> revokeInvitation(String id) async {
    await _dio.delete<void>('/families/me/invitations/$id');
  }

  /// Fetches invitation info for a token (public endpoint).
  ///
  /// Returns JSON with `family_name` key.
  Future<Map<String, dynamic>> getInvitationInfo(String token) async {
    final response =
        await _dio.get<Map<String, dynamic>>('/invitations/$token');
    return response.data!;
  }

  /// Accepts an invitation using the given [token].
  Future<void> acceptInvitation(String token) async {
    await _dio.post<void>(
      '/invitations/accept',
      data: {'token': token},
    );
  }
}

/// Provides a [FamilyRepository] using the app's Dio client.
final familyRepositoryProvider = Provider<FamilyRepository>((ref) {
  return FamilyRepository(ref.read(dioProvider));
});
