class NoteModel {
  final String id;
  final String title;
  final String content;
  final String? contentHtml;
  final String ownerId;
  final String? folderId;
  final int version;
  final bool isPublic;
  final String? shareToken;
  final int viewCount;
  final DateTime createdAt;
  final DateTime updatedAt;

  NoteModel({
    required this.id,
    required this.title,
    this.content = '',
    this.contentHtml,
    required this.ownerId,
    this.folderId,
    this.version = 0,
    this.isPublic = false,
    this.shareToken,
    this.viewCount = 0,
    required this.createdAt,
    required this.updatedAt,
  });

  factory NoteModel.fromJson(Map<String, dynamic> json) {
    final folderIdRaw = json['folder_id'];
    String? folderId;
    if (folderIdRaw is Map) {
      folderId = folderIdRaw['Valid'] == true ? folderIdRaw['UUID'] as String? : null;
    } else if (folderIdRaw is String) {
      folderId = folderIdRaw;
    }
    final updatedAtRaw = json['updated_at'] as String?;

    return NoteModel(
      id: json['id'] as String,
      title: json['title'] as String,
      content: json['content'] as String? ?? '',
      contentHtml: json['content_html'] as String?,
      ownerId: json['owner_id'] as String,
      folderId: folderId,
      version: json['version'] as int? ?? 0,
      isPublic: json['is_public'] as bool? ?? false,
      shareToken: json['share_token'] as String?,
      viewCount: json['view_count'] as int? ?? 0,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: updatedAtRaw != null ? DateTime.parse(updatedAtRaw) : DateTime.parse(json['created_at'] as String),
    );
  }

  Map<String, dynamic> toJson() => {
        'title': title,
        'content': content,
        if (folderId != null) 'folder_id': folderId,
        'version': version,
        'is_public': isPublic,
      };

  NoteModel copyWith({
    String? title,
    String? content,
    String? folderId,
    int? version,
    bool? isPublic,
    String? shareToken,
  }) {
    return NoteModel(
      id: id,
      title: title ?? this.title,
      content: content ?? this.content,
      contentHtml: contentHtml,
      ownerId: ownerId,
      folderId: folderId ?? this.folderId,
      version: version ?? this.version,
      isPublic: isPublic ?? this.isPublic,
      shareToken: shareToken ?? this.shareToken,
      viewCount: viewCount,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }

  String get preview {
    if (content.isEmpty) return 'Empty note';
    final text = content.replaceAll(RegExp(r'[*#_~`>\[\]()!]'), '');
    return text.length > 120 ? '${text.substring(0, 120)}...' : text;
  }
}

class NoteVersionModel {
  final String id;
  final String noteId;
  final String? title;
  final String? content;
  final int version;
  final String? changedBy;
  final String? changeSummary;
  final DateTime createdAt;

  NoteVersionModel({
    required this.id,
    required this.noteId,
    this.title,
    this.content,
    required this.version,
    this.changedBy,
    this.changeSummary,
    required this.createdAt,
  });

  factory NoteVersionModel.fromJson(Map<String, dynamic> json) => NoteVersionModel(
        id: json['id'] as String,
        noteId: json['note_id'] as String,
        title: json['title'] as String?,
        content: json['content'] as String?,
        version: json['version'] as int,
        changedBy: json['changed_by'] as String?,
        changeSummary: json['change_summary'] as String?,
        createdAt: DateTime.parse(json['created_at'] as String),
      );
}

class FolderModel {
  final String id;
  final String name;
  final String? parentId;
  final String ownerId;
  final String? color;
  final int sortOrder;
  final DateTime createdAt;
  final DateTime updatedAt;

  FolderModel({
    required this.id,
    required this.name,
    this.parentId,
    required this.ownerId,
    this.color,
    this.sortOrder = 0,
    required this.createdAt,
    required this.updatedAt,
  });

  factory FolderModel.fromJson(Map<String, dynamic> json) => FolderModel(
        id: json['id'] as String,
        name: json['name'] as String,
        parentId: json['parent_id'] as String?,
        ownerId: json['owner_id'] as String,
        color: json['color'] as String?,
        sortOrder: json['sort_order'] as int? ?? 0,
        createdAt: DateTime.parse(json['created_at'] as String),
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );
}
