import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/network/api_client.dart';
import '../../../data/models/note_model.dart';
import 'auth_provider.dart';

final versionListProvider = StateNotifierProvider.family<VersionListNotifier, AsyncValue<List<NoteVersionModel>>, String>((ref, noteId) {
  return VersionListNotifier(ref.watch(apiClientProvider), noteId);
});

class VersionListNotifier extends StateNotifier<AsyncValue<List<NoteVersionModel>>> {
  final ApiClient _api;
  final String noteId;

  VersionListNotifier(this._api, this.noteId) : super(const AsyncValue.loading()) {
    loadVersions();
  }

  Future<void> loadVersions() async {
    state = const AsyncValue.loading();
    try {
      final response = await _api.get('/notes/$noteId/versions');
      final items = (response.data!['data'] as List)
          .map((e) => NoteVersionModel.fromJson(e as Map<String, dynamic>))
          .toList();
      state = AsyncValue.data(items);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  Future<void> restoreVersion(int version) async {
    try {
      await _api.post('/notes/$noteId/versions/$version/restore');
      await loadVersions();
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }
}
