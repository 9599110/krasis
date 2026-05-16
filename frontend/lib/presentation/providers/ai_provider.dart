import 'dart:async';
import 'dart:convert';
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

      // Use a separate Dio for streaming — ResponseType.stream returns ResponseBody
      final dio = Dio(BaseOptions(
        baseUrl: AppConfig.apiBaseUrl,
        responseType: ResponseType.stream,
      ));
      if (token != null) {
        dio.options.headers['Authorization'] = 'Bearer $token';
      }

      final response = await dio.post<ResponseBody>(
        '/ai/ask/stream',
        data: {
          'question': question,
          if (conversationId.isNotEmpty) 'conversation_id': conversationId,
          'stream': true,
        },
      );

      final body = response.data;
      if (body == null) {
        throw Exception('Empty stream response');
      }

      await for (final chunk in body.stream) {
        final decoded = utf8.decode(chunk);
        final tokens = _parseSSEChunk(decoded);
        for (final token in tokens) {
          if (token == null) continue; // done event
          yield token;
          state = state.copyWith(streamingText: state.streamingText + token);
        }
      }

      _finalizeMessage();
    } catch (e) {
      if (e is DioException && e.response?.statusCode == 404) {
        // Model not configured, let user know
        state = state.copyWith(isStreaming: false);
        yield '';
        state = state.copyWith(
          messages: [
            ...state.messages,
            AIMessageModel(
              id: 'local_err_${DateTime.now().millisecondsSinceEpoch}',
              conversationId: conversationId,
              role: 'assistant',
              content: '❌ AI 模型未配置或不存在，请在服务端设置正确的模型',
              createdAt: DateTime.now(),
            ),
          ],
        );
        return;
      }
      state = state.copyWith(isStreaming: false);
      rethrow;
    }
  }

  /// Parse an SSE chunk and return a list of tokens (null = done event).
  List<String?> _parseSSEChunk(String chunk) {
    final results = <String?>[];
    final lines = chunk.split('\n');

    for (final line in lines) {
      final trimmed = line.trim();
      if (trimmed.isEmpty) continue;

      if (trimmed.startsWith('event: done')) {
        results.add(null); // signal done
        continue;
      }

      if (trimmed.startsWith('data: ')) {
        try {
          final data = jsonDecode(trimmed.substring(6)) as Map<String, dynamic>;
          final token = data['token'] as String?;
          if (token != null) {
            results.add(token);
          }
        } catch (_) {
          // skip malformed json
        }
      }
    }

    return results;
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
    } else {
      state = state.copyWith(isStreaming: false);
    }
  }
}
