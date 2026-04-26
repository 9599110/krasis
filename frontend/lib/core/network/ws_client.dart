import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';

class WSClient {
  WebSocketChannel? _channel;
  final String _wsBaseUrl;
  final String Function() _getToken;
  final _controller = StreamController<Map<String, dynamic>>.broadcast();
  Timer? _reconnectTimer;
  int _reconnectAttempts = 0;
  final int _maxReconnectAttempts = 5;
  String? _noteId;

  WSClient({
    required String wsBaseUrl,
    required String Function() getToken,
  }) : _wsBaseUrl = wsBaseUrl.replaceFirst(RegExp(r'/+$'), ''),
       _getToken = getToken;

  Stream<Map<String, dynamic>> get stream => _controller.stream;

  bool get isConnected => _channel != null;

  void connect(String noteId) {
    _noteId = noteId;
    _reconnectAttempts = 0;
    _doConnect();
  }

  void _disconnect() {
    _channel?.sink.close();
    _channel = null;
  }

  void _doConnect() {
    _disconnect();

    final token = _getToken();
    if (token.isEmpty) return;

    final url = '$_wsBaseUrl/ws/collab?note_id=$_noteId&token=$token';
    _channel = WebSocketChannel.connect(Uri.parse(url));

    _channel!.stream.listen(
      (data) {
        if (data is String) {
          try {
            _controller.add(jsonDecode(data) as Map<String, dynamic>);
          } catch (_) {
            // ignore parse errors
          }
        }
      },
      onError: (_) => _scheduleReconnect(),
      onDone: () => _scheduleReconnect(),
    );
  }

  void send(Map<String, dynamic> message) {
    _channel?.sink.add(jsonEncode(message));
  }

  void sendSync(String update, int version) {
    send({'type': 'sync', 'payload': {'update': update, 'version': version}});
  }

  void sendAwareness(Map<String, dynamic> payload) {
    send({'type': 'awareness', 'payload': payload});
  }

  void _scheduleReconnect() {
    _channel = null;
    if (_reconnectAttempts >= _maxReconnectAttempts) return;

    _reconnectAttempts++;
    final delay = Duration(
      milliseconds: (1000 * _pow(2, _reconnectAttempts)).clamp(0, 30000),
    );
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(delay, _doConnect);
  }

  int _pow(int base, int exp) {
    var result = 1;
    for (var i = 0; i < exp; i++) result *= base;
    return result;
  }

  void disconnect() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    _reconnectAttempts = _maxReconnectAttempts;
    _channel?.sink.close();
    _channel = null;
  }

  void dispose() {
    disconnect();
    _controller.close();
  }
}
