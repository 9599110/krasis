import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/network/api_client.dart';
import '../../../core/errors/exceptions.dart';
import '../../../data/models/note_model.dart';
import 'auth_provider.dart';

final noteListProvider = StateNotifierProvider<NoteListNotifier, AsyncValue<List<NoteModel>>>((ref) {
  return NoteListNotifier(ref.watch(apiClientProvider));
});

class NoteListNotifier extends StateNotifier<AsyncValue<List<NoteModel>>> {
  final ApiClient _api;
  String? _currentFolderId;
  bool _initialized = false;

  NoteListNotifier(this._api) : super(const AsyncValue.loading());

  Future<void> loadNotes({String? folderId}) async {
    if (_initialized && _currentFolderId == folderId) return;
    _initialized = true;
    _currentFolderId = folderId;
    state = const AsyncValue.loading();
    try {
      // ignore: avoid_print
      print('[notes] loadNotes start folderId=$folderId');
      final response = await _api
          .get('/notes', queryParameters: {
            'page': 1,
            'size': 50,
            if (folderId != null) 'folder_id': folderId,
          })
          .timeout(const Duration(seconds: 12));
      final data = response.data!['data'] as Map<String, dynamic>;
      final items = (data['items'] as List)
          .map((e) => NoteModel.fromJson(e as Map<String, dynamic>))
          .toList();
      // ignore: avoid_print
      print('[notes] loadNotes success count=${items.length}');
      state = AsyncValue.data(items);
    } catch (e, st) {
      // ignore: avoid_print
      print('[notes] loadNotes error: $e');
      state = AsyncValue.error(e, st);
    }
  }

  Future<void> refresh() async {
    await loadNotes(folderId: _currentFolderId);
  }
}

final noteEditorProvider = StateNotifierProvider.family<NoteEditorNotifier, AsyncValue<NoteModel?>, String>((ref, noteId) {
  return NoteEditorNotifier(ref.watch(apiClientProvider), noteId);
});

class NoteEditorNotifier extends StateNotifier<AsyncValue<NoteModel?>> {
  final ApiClient _api;
  final String noteId;

  NoteEditorNotifier(this._api, this.noteId) : super(const AsyncValue.loading()) {
    if (noteId != 'new') load();
  }

  Future<NoteModel?> load() async {
    state = const AsyncValue.loading();
    try {
      final response = await _api.get('/notes/$noteId');
      final data = response.data!['data'] as Map<String, dynamic>;
      final note = NoteModel.fromJson(data);
      state = AsyncValue.data(note);
      return note;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      return null;
    }
  }

  Future<NoteModel> createNote({required String title, String content = '', String? folderId}) async {
    final response = await _api.post('/notes', data: {
      'title': title,
      'content': content,
      if (folderId != null) 'folder_id': folderId,
    });
    final data = response.data!['data'] as Map<String, dynamic>;
    return NoteModel.fromJson(data);
  }

  Future<void> updateNote({
    required String title,
    required String content,
    required int version,
    String? changeSummary,
  }) async {
    try {
      final response = await _api.put(
        '/notes/$noteId',
        data: {
          'title': title,
          'content': content,
          'version': version,
          if (changeSummary != null) 'change_summary': changeSummary,
        },
        headers: {'If-Match': version.toString()},
      );
      final data = response.data!['data'] as Map<String, dynamic>;
      state = AsyncValue.data(NoteModel.fromJson(data));
    } on DioException catch (e) {
      if (e.response?.statusCode == 409) {
        throw VersionConflictException(currentVersion: 0);
      }
      rethrow;
    }
  }

  Future<void> deleteNote() async {
    await _api.delete('/notes/$noteId');
  }
}

final folderListProvider = StateNotifierProvider<FolderListNotifier, AsyncValue<List<FolderModel>>>((ref) {
  return FolderListNotifier(ref.watch(apiClientProvider));
});

class FolderListNotifier extends StateNotifier<AsyncValue<List<FolderModel>>> {
  final ApiClient _api;

  FolderListNotifier(this._api) : super(const AsyncValue.loading()) {
    loadFolders();
  }

  Future<void> loadFolders() async {
    state = const AsyncValue.loading();
    try {
      final response = await _api.get('/folders').timeout(const Duration(seconds: 12));
      final items = (response.data!['data'] as List)
          .map((e) => FolderModel.fromJson(e as Map<String, dynamic>))
          .toList();
      state = AsyncValue.data(items);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }
}
