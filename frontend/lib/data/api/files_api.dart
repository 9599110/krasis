/// API functions for file operations (hidden/unhide/hide files).
///
/// Hidden files are encrypted files that are stored on the server but NOT
/// synced to the frontend by default. They require explicit "unhide" before
/// appearing in normal file listings.
library;

import '../../../core/network/api_client.dart';

/// Represents a file returned by the file listing endpoints.
class FolderFile {
  final String id;
  final String? noteId;
  final String? folderId;
  final String userId;
  final String fileName;
  final String? fileType;
  final String? mimeType;
  final String storagePath;
  final String bucket;
  final int? sizeBytes;
  final int? width;
  final int? height;
  final String? thumbnailUrl;
  final Map<String, dynamic>? metadata;
  final int status;
  final DateTime createdAt;

  FolderFile({
    required this.id,
    this.noteId,
    this.folderId,
    required this.userId,
    required this.fileName,
    this.fileType,
    this.mimeType,
    required this.storagePath,
    required this.bucket,
    this.sizeBytes,
    this.width,
    this.height,
    this.thumbnailUrl,
    this.metadata,
    required this.status,
    required this.createdAt,
  });

  factory FolderFile.fromJson(Map<String, dynamic> json) {
    return FolderFile(
      id: json['id'] as String,
      noteId: json['note_id'] as String?,
      folderId: json['folder_id'] as String?,
      userId: json['user_id'] as String,
      fileName: json['file_name'] as String,
      fileType: json['file_type'] as String?,
      mimeType: json['mime_type'] as String?,
      storagePath: json['storage_path'] as String,
      bucket: json['bucket'] as String,
      sizeBytes: json['size_bytes'] as int?,
      width: json['width'] as int?,
      height: json['height'] as int?,
      thumbnailUrl: json['thumbnail_url'] as String?,
      metadata: json['metadata'] as Map<String, dynamic>?,
      status: json['status'] as int,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }

  String get fileSize {
    if (sizeBytes == null || sizeBytes! <= 0) return '';
    const units = ['B', 'KB', 'MB', 'GB'];
    double size = sizeBytes!.toDouble();
    int i = 0;
    while (size >= 1024 && i < units.length - 1) {
      size /= 1024;
      i++;
    }
    return '${size.toStringAsFixed(i == 0 ? 0 : 1)} ${units[i]}';
  }

  bool get isImage {
    if (mimeType != null && mimeType!.startsWith('image/')) return true;
    if (fileName.contains('.')) {
      final ext = fileName.split('.').last.toLowerCase();
      const imgExts = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico', 'avif'];
      return imgExts.contains(ext);
    }
    return false;
  }

  String get iconType {
    if (isImage) return 'image';
    if (fileName.contains('.')) {
      final ext = fileName.split('.').last.toLowerCase();
      const iconMap = {
        'pdf': 'file-pdf', 'doc': 'file-word', 'docx': 'file-word',
        'xls': 'file-excel', 'xlsx': 'file-excel',
        'ppt': 'file-ppt', 'pptx': 'file-ppt',
        'zip': 'file-zip', 'rar': 'file-zip', '7z': 'file-zip',
        'mp4': 'video', 'mov': 'video', 'mp3': 'audio', 'wav': 'audio',
        'txt': 'file-text', 'json': 'file-code', 'js': 'file-code',
        'ts': 'file-code', 'py': 'file-code', 'go': 'file-code',
        'md': 'file-text', 'csv': 'file-excel',
      };
      return iconMap[ext] ?? 'file';
    }
    return 'file';
  }
}

/// List hidden files in a folder.
/// Hidden files are NOT returned by normal ListByFolder calls.
Future<List<FolderFile>> listHiddenFiles(ApiClient api, String folderId) async {
  final res = await api.get('/files/folder/$folderId/hidden');
  final data = res.data?['data'];
  if (data is List) {
    return data.map((e) => FolderFile.fromJson(e as Map<String, dynamic>)).toList();
  }
  return [];
}

/// Unhide a file – makes it visible in normal file listings.
/// Only after calling this will the file appear in ListByFolder/ListByNote results.
Future<void> unhideFile(ApiClient api, String fileId) async {
  await api.post('/files/$fileId/unhide');
}

/// Mark a file as hidden – hides it from normal file listings.
/// The file remains stored but is no longer synced to the frontend by default.
Future<void> markHidden(ApiClient api, String fileId) async {
  await api.post('/files/$fileId/hide');
}
