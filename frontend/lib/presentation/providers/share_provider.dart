import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/network/api_client.dart';
import 'auth_provider.dart';

class ShareStatus {
  final String id;
  final String noteId;
  final String token;
  final String permission;
  final bool isActive;
  final DateTime? expiresAt;
  final DateTime createdAt;

  ShareStatus({
    required this.id,
    required this.noteId,
    required this.token,
    this.permission = 'read',
    this.isActive = true,
    this.expiresAt,
    required this.createdAt,
  });

  factory ShareStatus.fromJson(Map<String, dynamic> json) => ShareStatus(
        id: json['id'] as String,
        noteId: json['note_id'] as String,
        token: json['token'] as String,
        permission: json['permission'] as String? ?? 'read',
        isActive: json['is_active'] as bool? ?? true,
        expiresAt: json['expires_at'] != null
            ? DateTime.parse(json['expires_at'] as String)
            : null,
        createdAt: DateTime.parse(json['created_at'] as String),
      );
}

final shareProvider = StateNotifierProvider.family<ShareNotifier, AsyncValue<ShareStatus?>, String>((ref, noteId) {
  return ShareNotifier(ref.watch(apiClientProvider), noteId);
});

class ShareNotifier extends StateNotifier<AsyncValue<ShareStatus?>> {
  final ApiClient _api;
  final String noteId;

  ShareNotifier(this._api, this.noteId) : super(const AsyncValue.loading()) {
    loadShare();
  }

  Future<void> loadShare() async {
    state = const AsyncValue.loading();
    try {
      final response = await _api.get('/notes/$noteId/share');
      final data = response.data!['data'] as Map<String, dynamic>;
      state = AsyncValue.data(ShareStatus.fromJson(data));
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  Future<void> createShare({
    String permission = 'read',
    String password = '',
    DateTime? expiresAt,
  }) async {
    state = const AsyncValue.loading();
    try {
      final body = <String, dynamic>{'permission': permission};
      if (password.isNotEmpty) body['password'] = password;
      if (expiresAt != null) body['expires_at'] = expiresAt.toIso8601String();

      final response = await _api.post('/notes/$noteId/share', data: body);
      final data = response.data!['data'] as Map<String, dynamic>;
      state = AsyncValue.data(ShareStatus.fromJson(data));
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  Future<void> revokeShare() async {
    state = const AsyncValue.loading();
    try {
      await _api.delete('/notes/$noteId/share');
      state = const AsyncValue.data(null);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }
}
