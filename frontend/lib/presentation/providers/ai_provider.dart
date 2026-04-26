import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/network/api_client.dart';
import '../../../core/storage/secure_storage.dart';
import '../../../config/app_config.dart';
import '../../../data/models/ai_message_model.dart';
import 'auth_provider.dart';

class AIChatState {
  final List<AIMessageModel> messages;
  final String streamingText;
  final bool isStreaming;

  const AIChatState({
    this.messages = const [],
    this.streamingText = '',
    this.isStreaming = false,
  });

  AIChatState copyWith({
    List<AIMessageModel>? messages,
    String? streamingText,
    bool? isStreaming,
  }) {
    return AIChatState(
      messages: messages ?? this.messages,
      streamingText: streamingText ?? this.streamingText,
      isStreaming: isStreaming ?? this.isStreaming,
    );
  }
}

final aiChatProvider = StateNotifierProvider.family<AIChatNotifier, AIChatState, String>((ref, conversationId) {
  return AIChatNotifier(ref.watch(apiClientProvider), ref.watch(secureStorageProvider), conversationId);
});

class AIChatNotifier extends StateNotifier<AIChatState> {
  final ApiClient _api;
  final SecureStorage _storage;
  final String conversationId;

  AIChatNotifier(this._api, this._storage, this.conversationId) : super(const AIChatState()) {
    if (conversationId.isNotEmpty) _loadMessages();
  }

  Future<void> _loadMessages() async {
    try {
      final response = await _api.get('/ai/conversations/$conversationId/messages');
      final items = (response.data!['data'] as List)
          .map((e) => AIMessageModel.fromJson(e as Map<String, dynamic>))
          .toList();
      state = AIChatState(messages: items);
    } catch (_) {
      // Start with empty state
    }
  }

  Stream<String> ask(String question) async* {
    final userMsg = AIMessageModel(
      id: 'local_${DateTime.now().millisecondsSinceEpoch}',
      conversationId: conversationId,
      role: 'user',
      content: question,
      createdAt: DateTime.now(),
    );
    state = state.copyWith(
      messages: [...state.messages, userMsg],
      isStreaming: true,
      streamingText: '',
    );

    try {
      final token = await _storage.getAccessToken();
      final dio = Dio(BaseOptions(
        baseUrl: AppConfig.apiBaseUrl,
        responseType: ResponseType.stream,
      ));
      if (token != null) {
        dio.options.headers['Authorization'] = 'Bearer $token';
      }

      final response = await dio.post<Map<String, dynamic>>(
        '/ai/ask/stream',
        data: {
          'question': question,
          if (conversationId.isNotEmpty) 'conversation_id': conversationId,
          'stream': true,
        },
      );

      final stream = response.data!['data'] as Stream<Uint8List>? ?? (response.data as Stream<dynamic>);

      await for (final token in _parseSSEStream(stream)) {
        yield token;
      }
    } catch (e) {
      state = state.copyWith(isStreaming: false);
      rethrow;
    }
  }

  Stream<String> _parseSSEStream(Stream stream) async* {
    var buffer = '';

    await for (final chunk in stream) {
      String text;
      if (chunk is List<int>) {
        text = utf8.decode(chunk);
      } else {
        text = chunk.toString();
      }
      buffer += text;
      final lines = buffer.split('\n');
      buffer = lines.removeLast();

      for (final line in lines) {
        if (line.startsWith('event: done')) {
          _finalizeMessage();
          return;
        }
        if (line.startsWith('data: ')) {
          try {
            final data = jsonDecode(line.substring(6)) as Map<String, dynamic>;
            final token = data['token'] as String?;
            if (token != null) {
              yield token;
              state = state.copyWith(streamingText: state.streamingText + token);
            }
          } catch (_) {
            // skip parse errors
          }
        }
      }
    }
    _finalizeMessage();
  }

  void _finalizeMessage() {
    if (state.streamingText.isNotEmpty) {
      state = state.copyWith(
        isStreaming: false,
        messages: [
          ...state.messages,
          AIMessageModel(
            id: 'local_resp_${DateTime.now().millisecondsSinceEpoch}',
            conversationId: conversationId,
            role: 'assistant',
            content: state.streamingText,
            createdAt: DateTime.now(),
          ),
        ],
      );
    }
  }
}
