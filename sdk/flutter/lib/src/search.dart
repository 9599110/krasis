import 'dart:async';
import 'client.dart';
import 'types.dart';

class SearchModule extends Module {
  SearchModule(super.client);

  Future<List<SearchResult>> query({
    required String q,
    int page = 1,
    int size = 20,
    String? type,
  }) async {
    final params = <String, String>{
      'q': q,
      'page': '$page',
      'size': '$size',
      if (type != null) 'type': type,
    };
    final queryStr = params.entries.map((e) => '${e.key}=${Uri.encodeQueryComponent(e.value)}').join('&');
    final json = await client.get<Map<String, dynamic>>('/search?$queryStr');
    final items = (json['items'] as List)
        .map((e) => SearchResult.fromJson(e as Map<String, dynamic>))
        .toList();
    return items;
  }
}

class FileModule extends Module {
  FileModule(super.client);

  Future<PresignResult> presignUpload({
    required String fileName,
    required String fileType,
    String? noteId,
    int? sizeBytes,
  }) async {
    final json = await client.post<Map<String, dynamic>>('/files/presign', {
      'file_name': fileName,
      'file_type': fileType,
      if (noteId != null) 'note_id': noteId,
      if (sizeBytes != null) 'size_bytes': sizeBytes,
    });
    return PresignResult.fromJson(json);
  }

  Future<void> confirmUpload(String fileId) async {
    await client.post('/files/$fileId/confirm', null);
  }

  Future<void> delete(String fileId) async {
    await client.delete('/files/$fileId');
  }
}
