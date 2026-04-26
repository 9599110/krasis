import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'types.dart';

class CollabModule {
  final String wsBaseUrl;
  String token;

  WebSocketChannel? _channel;
  String? _noteId;
  final _eventController = StreamController<CollabEvent>.broadcast();
  Timer? _reconnectTimer;
  int _reconnectAttempts = 0;
  final int maxReconnectAttempts;

  CollabModule({
    required this.wsBaseUrl,
    required this.token,
    this.maxReconnectAttempts = 5,
  });

  Stream<CollabEvent> get events => _eventController.stream;

  void connect(String noteId) {
    _noteId = noteId;
    _reconnectAttempts = 0;
    _doConnect();
  }

  void _doConnect() {
    _channel?.sink.close();

    final url = '$wsBaseUrl/ws/collab?note_id=$_noteId&token=$token';
    _channel = WebSocketChannel.connect(Uri.parse(url));

    _channel!.stream.listen(
      (data) {
        try {
          final msg = jsonDecode(data as String) as Map<String, dynamic>;
          _handleMessage(msg);
        } catch (_) {
          // ignore parse errors
        }
      },
      onDone: () {
        _emit(CollabEvent.close(code: 1000, reason: 'Connection closed'));
        _scheduleReconnect();
      },
      onError: (error) {
        _emit(CollabEvent.error(error: error));
      },
    );

    _reconnectAttempts = 0;
    _emit(const CollabEvent(type: CollabEventType.open));
  }

  void _handleMessage(Map<String, dynamic> msg) {
    final type = msg['type'] as String?;
    final payload = msg['payload'] as Map<String, dynamic>? ?? {};

    switch (type) {
      case 'sync':
        _emit(CollabEvent.sync(
          payload: SyncPayload.fromJson(payload),
          userId: payload['user_id'] as String? ?? '',
        ));
        break;
      case 'awareness':
        _emit(CollabEvent.awareness(awarenessPayload: AwarenessPayload.fromJson(payload)));
        break;
      case 'presence':
        final users = (payload['users'] as List?)
                ?.map((u) => {
                      'user_id': (u as Map)['user_id'] as String,
                      'username': u['username'] as String,
                    })
                .toList() ??
            [];
        _emit(CollabEvent.presence(users: users));
        break;
      default:
        break;
    }
  }

  void sendSync(String update, int version) {
    _send({
      'type': 'sync',
      'payload': {'update': update, 'version': version},
    });
  }

  void sendAwareness(AwarenessPayload payload) {
    _send({'type': 'awareness', 'payload': payload.toJson()});
  }

  void sendPresenceQuery() {
    _send({'type': 'awareness_query', 'payload': {}});
  }

  void _send(Map<String, dynamic> msg) {
    if (_channel != null) {
      _channel!.sink.add(jsonEncode(msg));
    }
  }

  void _scheduleReconnect() {
    if (_reconnectAttempts >= maxReconnectAttempts) return;

    _reconnectAttempts++;
    final delay = _min(1000 * _pow(2, _reconnectAttempts), 30000);
    _reconnectTimer = Timer(Duration(milliseconds: delay), _doConnect);
  }

  void disconnect() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    _reconnectAttempts = maxReconnectAttempts;
    _channel?.sink.close();
    _channel = null;
  }

  void _emit(CollabEvent event) {
    if (!_eventController.isClosed) {
      _eventController.add(event);
    }
  }

  static int _min(int a, int b) => a < b ? a : b;
  static int _pow(int base, int exp) {
    var result = 1;
    for (var i = 0; i < exp; i++) result *= base;
    return result;
  }
}

enum CollabEventType { open, close, error, sync, awareness, presence }

class CollabEvent {
  final CollabEventType type;
  final int? code;
  final String? reason;
  final Object? error;
  final SyncPayload? payload;
  final AwarenessPayload? awarenessPayload;
  final String? userId;
  final List<Map<String, String>>? users;

  const CollabEvent({
    required this.type,
    this.code,
    this.reason,
    this.error,
    this.payload,
    this.awarenessPayload,
    this.userId,
    this.users,
  });

  factory CollabEvent.close({required int code, required String reason}) =>
      CollabEvent(type: CollabEventType.close, code: code, reason: reason);

  factory CollabEvent.error({required Object error}) =>
      CollabEvent(type: CollabEventType.error, error: error);

  factory CollabEvent.sync({required SyncPayload payload, required String userId}) =>
      CollabEvent(type: CollabEventType.sync, payload: payload, userId: userId);

  factory CollabEvent.awareness({required AwarenessPayload awarenessPayload}) =>
      CollabEvent(type: CollabEventType.awareness, awarenessPayload: awarenessPayload);

  factory CollabEvent.presence({required List<Map<String, String>> users}) =>
      CollabEvent(type: CollabEventType.presence, users: users);
}
