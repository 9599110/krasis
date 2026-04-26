import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';
import 'error.dart';
import 'types.dart';

class KrasisClient {
  final String apiBaseUrl;
  String? _token;
  final http.Client _http;

  KrasisClient({
    required this.apiBaseUrl,
    String? token,
    http.Client? client,
  }) : _http = client ?? http.Client() {
    _token = token;
  }

  String? get token => _token;

  bool get isAuthenticated => _token != null;

  Future<void> setToken(String token) async {
    _token = token;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('krasis_token', token);
  }

  Future<void> clearToken() async {
    _token = null;
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('krasis_token');
  }

  Future<void> loadToken() async {
    final prefs = await SharedPreferences.getInstance();
    _token = prefs.getString('krasis_token');
  }

  Map<String, String> _headers() {
    final h = <String, String>{'Content-Type': 'application/json'};
    final t = _token;
    if (t != null) {
      h['Authorization'] = 'Bearer $t';
    }
    return h;
  }

  Future<T> get<T>(String path, {Map<String, String>? headers}) async {
    final res = await _http.get(
      Uri.parse('$apiBaseUrl$path'),
      headers: {..._headers(), if (headers != null) ...headers},
    );
    return _handle<T>(res);
  }

  Future<T> post<T>(String path, dynamic body, {Map<String, String>? headers}) async {
    final res = await _http.post(
      Uri.parse('$apiBaseUrl$path'),
      headers: {..._headers(), if (headers != null) ...headers},
      body: jsonEncode(body),
    );
    return _handle<T>(res);
  }

  Future<T> put<T>(String path, dynamic body, {Map<String, String>? headers}) async {
    final res = await _http.put(
      Uri.parse('$apiBaseUrl$path'),
      headers: {..._headers(), if (headers != null) ...headers},
      body: jsonEncode(body),
    );
    return _handle<T>(res);
  }

  Future<T> delete<T>(String path, {Map<String, String>? headers}) async {
    final res = await _http.delete(
      Uri.parse('$apiBaseUrl$path'),
      headers: {..._headers(), if (headers != null) ...headers},
    );
    return _handle<T>(res);
  }

  Future<http.StreamedResponse> postStream(String path, dynamic body, {Map<String, String>? headers}) async {
    final request = http.Request('POST', Uri.parse('$apiBaseUrl$path'));
    request.headers.addAll({..._headers(), if (headers != null) ...headers});
    request.body = jsonEncode(body);
    return _http.send(request);
  }

  T _handle<T>(http.Response res) {
    if (res.statusCode >= 200 && res.statusCode < 300) {
      final data = jsonDecode(res.body) as Map<String, dynamic>;
      final payload = data['data'];
      if (payload is T) return payload;
      // For typed helpers, return payload cast
      return payload as T;
    }

    final body = jsonDecode(res.body) as Map<String, dynamic>?;
    final code = body?['code'] as String?;
    final message = body?['message'] as String? ?? 'Request failed';

    switch (res.statusCode) {
      case 401:
        throw AuthenticationException(message: message);
      case 404:
        throw NotFoundException(message: message);
      case 409:
        throw VersionConflictException(serverVersion: 0);
      case 429:
        throw RateLimitException();
      default:
        throw SDKException(message: message, statusCode: res.statusCode, code: code);
    }
  }

  void dispose() {
    _http.close();
  }
}

// Module base class
abstract class Module {
  final KrasisClient client;
  Module(this.client);
}

// Helper to parse paginated responses
PaginatedResponse<T> parsePaginated<T>(
  Map<String, dynamic> json,
  T Function(Map<String, dynamic>) fromJson,
) {
  final items = (json['items'] as List)
      .map((e) => fromJson(e as Map<String, dynamic>))
      .toList();
  return PaginatedResponse<T>(
    items: items,
    total: json['total'] as int,
    page: json['page'] as int,
    size: json['size'] as int,
  );
}
