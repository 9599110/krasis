import 'dart:convert';
import 'dart:async';
import 'client.dart';
import 'types.dart';

class AIModule extends Module {
  AIModule(super.client);

  Future<AskResponse> ask({
    required String question,
    String? conversationId,
    String? modelId,
    int? topK,
  }) async {
    final json = await client.post<Map<String, dynamic>>('/ai/ask', {
      'question': question,
      if (conversationId != null) 'conversation_id': conversationId,
      if (modelId != null) 'model_id': modelId,
      if (topK != null) 'top_k': topK,
    });
    return AskResponse.fromJson(json);
  }

  Stream<String> askStream({
    required String question,
    String? conversationId,
    String? modelId,
    int? topK,
  }) {
    final controller = StreamController<String>();

    final body = {
      'question': question,
      if (conversationId != null) 'conversation_id': conversationId,
      if (modelId != null) 'model_id': modelId,
      if (topK != null) 'top_k': topK,
      'stream': true,
    };

    client.postStream('/ai/ask/stream', body).then((response) {
      final stream = response.stream;
      var buffer = '';

      stream.transform(utf8.decoder).listen(
        (chunk) {
          buffer += chunk;
          final lines = buffer.split('\n');
          buffer = lines.removeLast();

          for (final line in lines) {
            if (line.startsWith('event: token')) {
              // next line should be data:
            } else if (line.startsWith('data: ') && line.contains('token')) {
              try {
                final data = jsonDecode(line.substring(6)) as Map<String, dynamic>;
                final token = data['token'] as String?;
                if (token != null) controller.add(token);
              } catch (_) {
                // skip parse errors
              }
            } else if (line.startsWith('event: done')) {
              // stream complete
            }
          }
        },
        onDone: () {
          controller.close();
        },
        onError: (err) {
          controller.addError(err);
          controller.close();
        },
      );
    }).catchError((err) {
      controller.addError(err);
      controller.close();
    });

    return controller.stream;
  }

  Future<List<Conversation>> listConversations() async {
    final json = await client.get<List<dynamic>>('/ai/conversations');
    return json.map((e) => Conversation.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<List<Message>> getMessages(String conversationId) async {
    final json = await client.get<List<dynamic>>('/ai/conversations/$conversationId/messages');
    return json.map((e) => Message.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<Conversation> createConversation({String? title, String? model}) async {
    final json = await client.post<Map<String, dynamic>>('/ai/conversations', {
      if (title != null) 'title': title,
      if (model != null) 'model': model,
    });
    return Conversation.fromJson(json);
  }
}
