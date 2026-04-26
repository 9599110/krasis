class User {
  final String id;
  final String email;
  final String username;
  final String avatarUrl;
  final String role;
  final int status;
  final DateTime createdAt;

  User({
    required this.id,
    required this.email,
    required this.username,
    this.avatarUrl = '',
    this.role = 'viewer',
    this.status = 1,
    required this.createdAt,
  });

  factory User.fromJson(Map<String, dynamic> json) => User(
        id: json['id'] as String,
        email: json['email'] as String,
        username: json['username'] as String,
        avatarUrl: json['avatar_url'] as String? ?? '',
        role: json['role'] as String? ?? 'viewer',
        status: json['status'] as int? ?? 1,
        createdAt: DateTime.parse(json['created_at'] as String),
      );
}

class Note {
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

  Note({
    required this.id,
    required this.title,
    required this.content,
    this.contentHtml,
    required this.ownerId,
    this.folderId,
    required this.version,
    this.isPublic = false,
    this.shareToken,
    this.viewCount = 0,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Note.fromJson(Map<String, dynamic> json) => Note(
        id: json['id'] as String,
        title: json['title'] as String,
        content: json['content'] as String? ?? '',
        contentHtml: json['content_html'] as String?,
        ownerId: json['owner_id'] as String,
        folderId: json['folder_id'] as String?,
        version: json['version'] as int? ?? 0,
        isPublic: json['is_public'] as bool? ?? false,
        shareToken: json['share_token'] as String?,
        viewCount: json['view_count'] as int? ?? 0,
        createdAt: DateTime.parse(json['created_at'] as String),
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );

  Map<String, dynamic> toJson() => {
        'title': title,
        'content': content,
        if (folderId != null) 'folder_id': folderId,
        'version': version,
        'is_public': isPublic,
      };
}

class NoteVersion {
  final String id;
  final String noteId;
  final String? title;
  final String? content;
  final int version;
  final String? changedBy;
  final String? changeSummary;
  final DateTime createdAt;

  NoteVersion({
    required this.id,
    required this.noteId,
    this.title,
    this.content,
    required this.version,
    this.changedBy,
    this.changeSummary,
    required this.createdAt,
  });

  factory NoteVersion.fromJson(Map<String, dynamic> json) => NoteVersion(
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

class Folder {
  final String id;
  final String name;
  final String? parentId;
  final String ownerId;
  final String? color;
  final int sortOrder;
  final DateTime createdAt;
  final DateTime updatedAt;

  Folder({
    required this.id,
    required this.name,
    this.parentId,
    required this.ownerId,
    this.color,
    this.sortOrder = 0,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Folder.fromJson(Map<String, dynamic> json) => Folder(
        id: json['id'] as String,
        name: json['name'] as String,
        parentId: json['parent_id'] as String?,
        ownerId: json['owner_id'] as String,
        color: json['color'] as String?,
        sortOrder: json['sort_order'] as int? ?? 0,
        createdAt: DateTime.parse(json['created_at'] as String),
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );

  Map<String, dynamic> toJson() => {
        'name': name,
        if (parentId != null) 'parent_id': parentId,
        if (color != null) 'color': color,
        'sort_order': sortOrder,
      };
}

class ShareStatus {
  final String shareToken;
  final String shareUrl;
  final String permission;
  final bool passwordProtected;
  final DateTime? expiresAt;
  final String status;
  final String statusDescription;
  final DateTime createdAt;
  final String? rejectionReason;

  ShareStatus({
    required this.shareToken,
    required this.shareUrl,
    required this.permission,
    this.passwordProtected = false,
    this.expiresAt,
    required this.status,
    required this.statusDescription,
    required this.createdAt,
    this.rejectionReason,
  });

  factory ShareStatus.fromJson(Map<String, dynamic> json) => ShareStatus(
        shareToken: json['share_token'] as String,
        shareUrl: json['share_url'] as String,
        permission: json['permission'] as String,
        passwordProtected: json['password_protected'] as bool? ?? false,
        expiresAt: json['expires_at'] != null
            ? DateTime.parse(json['expires_at'] as String)
            : null,
        status: json['status'] as String,
        statusDescription: json['status_description'] as String,
        createdAt: DateTime.parse(json['created_at'] as String),
        rejectionReason: json['rejection_reason'] as String?,
      );
}

class SearchResult {
  final String type;
  final String id;
  final String title;
  final List<String> highlights;
  final double score;
  final DateTime updatedAt;

  SearchResult({
    required this.type,
    required this.id,
    required this.title,
    required this.highlights,
    required this.score,
    required this.updatedAt,
  });

  factory SearchResult.fromJson(Map<String, dynamic> json) => SearchResult(
        type: json['type'] as String,
        id: json['id'] as String,
        title: json['title'] as String,
        highlights: (json['highlights'] as List?)?.cast<String>() ?? [],
        score: (json['score'] as num?)?.toDouble() ?? 0.0,
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );
}

class FileItem {
  final String id;
  final String? noteId;
  final String userId;
  final String fileName;
  final String fileType;
  final String storagePath;
  final String bucket;
  final int? sizeBytes;
  final int status;
  final DateTime createdAt;

  FileItem({
    required this.id,
    this.noteId,
    required this.userId,
    required this.fileName,
    required this.fileType,
    required this.storagePath,
    required this.bucket,
    this.sizeBytes,
    this.status = 0,
    required this.createdAt,
  });

  factory FileItem.fromJson(Map<String, dynamic> json) => FileItem(
        id: json['id'] as String,
        noteId: json['note_id'] as String?,
        userId: json['user_id'] as String,
        fileName: json['file_name'] as String,
        fileType: json['file_type'] as String,
        storagePath: json['storage_path'] as String,
        bucket: json['bucket'] as String,
        sizeBytes: json['size_bytes'] as int?,
        status: json['status'] as int? ?? 0,
        createdAt: DateTime.parse(json['created_at'] as String),
      );
}

class PresignResult {
  final String fileId;
  final String uploadUrl;
  final int expiresIn;

  PresignResult({
    required this.fileId,
    required this.uploadUrl,
    required this.expiresIn,
  });

  factory PresignResult.fromJson(Map<String, dynamic> json) => PresignResult(
        fileId: json['file_id'] as String,
        uploadUrl: json['upload_url'] as String,
        expiresIn: json['expires_in'] as int,
      );
}

class Session {
  final String sessionId;
  final String deviceName;
  final String deviceType;
  final String ipAddress;
  final String userAgent;
  final DateTime lastActiveAt;
  final DateTime createdAt;
  final bool isCurrent;

  Session({
    required this.sessionId,
    required this.deviceName,
    required this.deviceType,
    required this.ipAddress,
    required this.userAgent,
    required this.lastActiveAt,
    required this.createdAt,
    this.isCurrent = false,
  });

  factory Session.fromJson(Map<String, dynamic> json) => Session(
        sessionId: json['session_id'] as String,
        deviceName: json['device_name'] as String,
        deviceType: json['device_type'] as String,
        ipAddress: json['ip_address'] as String,
        userAgent: json['user_agent'] as String,
        lastActiveAt: DateTime.parse(json['last_active_at'] as String),
        createdAt: DateTime.parse(json['created_at'] as String),
        isCurrent: json['is_current'] as bool? ?? false,
      );
}

class AskRequest {
  final String question;
  final String? conversationId;
  final String? modelId;
  final int? topK;
  final bool? stream;

  AskRequest({
    required this.question,
    this.conversationId,
    this.modelId,
    this.topK,
    this.stream,
  });

  Map<String, dynamic> toJson() => {
        'question': question,
        if (conversationId != null) 'conversation_id': conversationId,
        if (modelId != null) 'model_id': modelId,
        if (topK != null) 'top_k': topK,
        if (stream != null) 'stream': stream,
      };
}

class AskResponse {
  final String answer;
  final List<Reference> references;
  final String conversationId;
  final String? messageId;

  AskResponse({
    required this.answer,
    this.references = const [],
    required this.conversationId,
    this.messageId,
  });

  factory AskResponse.fromJson(Map<String, dynamic> json) {
    final refs = json['references'] as List?;
    return AskResponse(
      answer: json['answer'] as String? ?? '',
      references: refs != null
          ? refs.map((r) => Reference.fromJson(r as Map<String, dynamic>)).toList()
          : [],
      conversationId: json['conversation_id'] as String,
      messageId: json['message_id'] as String?,
    );
  }
}

class Reference {
  final String noteId;
  final String noteTitle;
  final String text;
  final int chunkIndex;

  Reference({
    required this.noteId,
    required this.noteTitle,
    required this.text,
    required this.chunkIndex,
  });

  factory Reference.fromJson(Map<String, dynamic> json) => Reference(
        noteId: json['note_id'] as String,
        noteTitle: json['note_title'] as String,
        text: json['text'] as String,
        chunkIndex: json['chunk_index'] as int,
      );
}

class Conversation {
  final String id;
  final String userId;
  final String title;
  final String model;
  final DateTime createdAt;
  final DateTime updatedAt;

  Conversation({
    required this.id,
    required this.userId,
    required this.title,
    required this.model,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Conversation.fromJson(Map<String, dynamic> json) => Conversation(
        id: json['id'] as String,
        userId: json['user_id'] as String,
        title: json['title'] as String,
        model: json['model'] as String,
        createdAt: DateTime.parse(json['created_at'] as String),
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );
}

class Message {
  final String id;
  final String conversationId;
  final String role;
  final String content;
  final List<dynamic> references;
  final int? tokenCount;
  final String? model;
  final DateTime createdAt;

  Message({
    required this.id,
    required this.conversationId,
    required this.role,
    required this.content,
    this.references = const [],
    this.tokenCount,
    this.model,
    required this.createdAt,
  });

  factory Message.fromJson(Map<String, dynamic> json) => Message(
        id: json['id'] as String,
        conversationId: json['conversation_id'] as String,
        role: json['role'] as String,
        content: json['content'] as String,
        references: json['references'] as List? ?? [],
        tokenCount: json['token_count'] as int?,
        model: json['model'] as String?,
        createdAt: DateTime.parse(json['created_at'] as String),
      );
}

class AIModel {
  final String id;
  final String name;
  final String provider;
  final String type;
  final String endpoint;
  final String modelName;
  final String? apiVersion;
  final int maxTokens;
  final double temperature;
  final double? topP;
  final int? dimensions;
  final bool isEnabled;
  final bool isDefault;
  final int priority;
  final DateTime createdAt;
  final DateTime updatedAt;

  AIModel({
    required this.id,
    required this.name,
    required this.provider,
    required this.type,
    required this.endpoint,
    required this.modelName,
    this.apiVersion,
    required this.maxTokens,
    required this.temperature,
    this.topP,
    this.dimensions,
    this.isEnabled = false,
    this.isDefault = false,
    this.priority = 0,
    required this.createdAt,
    required this.updatedAt,
  });

  factory AIModel.fromJson(Map<String, dynamic> json) => AIModel(
        id: json['id'] as String,
        name: json['name'] as String,
        provider: json['provider'] as String,
        type: json['type'] as String,
        endpoint: json['endpoint'] as String,
        modelName: json['model_name'] as String,
        apiVersion: json['api_version'] as String?,
        maxTokens: json['max_tokens'] as int,
        temperature: (json['temperature'] as num).toDouble(),
        topP: (json['top_p'] as num?)?.toDouble(),
        dimensions: json['dimensions'] as int?,
        isEnabled: json['is_enabled'] as bool? ?? false,
        isDefault: json['is_default'] as bool? ?? false,
        priority: json['priority'] as int? ?? 0,
        createdAt: DateTime.parse(json['created_at'] as String),
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );
}

class PaginatedResponse<T> {
  final List<T> items;
  final int total;
  final int page;
  final int size;

  PaginatedResponse({
    required this.items,
    required this.total,
    required this.page,
    required this.size,
  });
}

class AwarenessPayload {
  final String userId;
  final String username;
  final Map<String, dynamic>? cursor;
  final Map<String, dynamic>? selection;

  AwarenessPayload({
    required this.userId,
    required this.username,
    this.cursor,
    this.selection,
  });

  factory AwarenessPayload.fromJson(Map<String, dynamic> json) => AwarenessPayload(
        userId: json['user_id'] as String,
        username: json['username'] as String,
        cursor: json['cursor'] as Map<String, dynamic>?,
        selection: json['selection'] as Map<String, dynamic>?,
      );

  Map<String, dynamic> toJson() => {
        'user_id': userId,
        'username': username,
        if (cursor != null) 'cursor': cursor,
        if (selection != null) 'selection': selection,
      };
}

class SyncPayload {
  final String update;
  final int version;

  SyncPayload({required this.update, required this.version});

  factory SyncPayload.fromJson(Map<String, dynamic> json) => SyncPayload(
        update: json['update'] as String,
        version: json['version'] as int,
      );

  Map<String, dynamic> toJson() => {
        'update': update,
        'version': version,
      };
}
