import 'dart:async';
import 'client.dart';
import 'types.dart';

class NotesModule extends Module {
  NotesModule(super.client);

  Future<PaginatedResponse<Note>> list({
    String? folderId,
    int page = 1,
    int size = 20,
  }) async {
    final params = <String, String>{
      'page': '$page',
      'size': '$size',
      if (folderId != null) 'folder_id': folderId,
    };
    final query = params.entries.map((e) => '${e.key}=${Uri.encodeQueryComponent(e.value)}').join('&');
    final json = await client.get<Map<String, dynamic>>('/notes?$query');
    return parsePaginated(json, (e) => Note.fromJson(e));
  }

  Future<Note> create({
    required String title,
    String content = '',
    String? folderId,
    bool isPublic = false,
  }) async {
    final json = await client.post<Map<String, dynamic>>('/notes', {
      'title': title,
      'content': content,
      if (folderId != null) 'folder_id': folderId,
      'is_public': isPublic,
    });
    return Note.fromJson(json);
  }

  Future<Note> get(String id) async {
    final json = await client.get<Map<String, dynamic>>('/notes/$id');
    return Note.fromJson(json);
  }

  Future<Note> update({
    required String id,
    String? title,
    String? content,
    String? folderId,
    bool? isPublic,
    int? version,
    String? changeSummary,
  }) async {
    final headers = <String, String>{};
    if (version != null) {
      headers['If-Match'] = version.toString();
    }
    final json = await client.put<Map<String, dynamic>>(
      '/notes/$id',
      {
        if (title != null) 'title': title,
        if (content != null) 'content': content,
        if (folderId != null) 'folder_id': folderId,
        if (isPublic != null) 'is_public': isPublic,
        if (changeSummary != null) 'change_summary': changeSummary,
      },
      headers: headers,
    );
    return Note.fromJson(json);
  }

  Future<void> delete(String id) async {
    await client.delete('/notes/$id');
  }

  Future<List<NoteVersion>> versions(String id) async {
    final json = await client.get<List<dynamic>>('/notes/$id/versions');
    return json.map((e) => NoteVersion.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<Note> restore(String id, {int? version}) async {
    final json = await client.post<Map<String, dynamic>>(
      '/notes/$id/restore',
      version != null ? {'version': version} : {},
    );
    return Note.fromJson(json);
  }
}

class FoldersModule extends Module {
  FoldersModule(super.client);

  Future<List<Folder>> list() async {
    final json = await client.get<List<dynamic>>('/folders');
    return json.map((e) => Folder.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<Folder> create({
    required String name,
    String? parentId,
    String? color,
    int sortOrder = 0,
  }) async {
    final json = await client.post<Map<String, dynamic>>('/folders', {
      'name': name,
      if (parentId != null) 'parent_id': parentId,
      if (color != null) 'color': color,
      'sort_order': sortOrder,
    });
    return Folder.fromJson(json);
  }

  Future<Folder> update(String id, {String? name, String? parentId, String? color, int? sortOrder}) async {
    final json = await client.put<Map<String, dynamic>>('/folders/$id', {
      if (name != null) 'name': name,
      if (parentId != null) 'parent_id': parentId,
      if (color != null) 'color': color,
      if (sortOrder != null) 'sort_order': sortOrder,
    });
    return Folder.fromJson(json);
  }

  Future<void> delete(String id) async {
    await client.delete('/folders/$id');
  }
}

class ShareModule extends Module {
  ShareModule(super.client);

  Future<ShareStatus> create(String noteId, {String permission = 'read', String? password, DateTime? expiresAt}) async {
    final json = await client.post<Map<String, dynamic>>('/notes/$noteId/share', {
      'permission': permission,
      if (password != null) 'password': password,
      if (expiresAt != null) 'expires_at': expiresAt.toIso8601String(),
    });
    return ShareStatus.fromJson(json);
  }

  Future<ShareStatus> get(String noteId) async {
    final json = await client.get<Map<String, dynamic>>('/notes/$noteId/share');
    return ShareStatus.fromJson(json);
  }

  Future<void> revoke(String noteId) async {
    await client.delete('/notes/$noteId/share');
  }

  Future<Note> accessByToken(String token, {String? password}) async {
    final params = <String, String>{};
    if (password != null) params['password'] = password;
    final query = params.isNotEmpty ? '?${params.entries.map((e) => '${e.key}=${Uri.encodeQueryComponent(e.value)}').join('&')}' : '';
    final json = await client.get<Map<String, dynamic>>('/share/$token$query');
    return Note.fromJson(json);
  }
}
