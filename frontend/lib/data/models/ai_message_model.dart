class AIMessageModel {
  final String id;
  final String conversationId;
  final String role;
  final String content;
  final List<dynamic> references;
  final int? tokenCount;
  final String? model;
  final DateTime createdAt;

  AIMessageModel({
    required this.id,
    required this.conversationId,
    required this.role,
    required this.content,
    this.references = const [],
    this.tokenCount,
    this.model,
    required this.createdAt,
  });

  factory AIMessageModel.fromJson(Map<String, dynamic> json) => AIMessageModel(
        id: json['id'] as String,
        conversationId: json['conversation_id'] as String,
        role: json['role'] as String,
        content: json['content'] as String,
        references: json['references'] as List? ?? [],
        tokenCount: json['token_count'] as int?,
        model: json['model'] as String?,
        createdAt: DateTime.parse(json['created_at'] as String),
      );

  bool get isUser => role == 'user';
  bool get isAssistant => role == 'assistant';
}

class ConversationModel {
  final String id;
  final String userId;
  final String title;
  final String model;
  final DateTime createdAt;
  final DateTime updatedAt;

  ConversationModel({
    required this.id,
    required this.userId,
    required this.title,
    required this.model,
    required this.createdAt,
    required this.updatedAt,
  });

  factory ConversationModel.fromJson(Map<String, dynamic> json) => ConversationModel(
        id: json['id'] as String,
        userId: json['user_id'] as String,
        title: json['title'] as String,
        model: json['model'] as String,
        createdAt: DateTime.parse(json['created_at'] as String),
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );
}
