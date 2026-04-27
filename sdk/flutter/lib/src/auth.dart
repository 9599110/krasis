import 'client.dart';
import 'types.dart';

class AuthModule extends Module {
  AuthModule(super.client);

  String getOAuthUrl(String provider, {String? redirectUri, String? state}) {
    final params = <String, String>{'provider': provider};
    if (redirectUri != null) params['redirect_uri'] = redirectUri;
    if (state != null) params['state'] = state;
    final query = params.entries.map((e) => '${e.key}=${Uri.encodeQueryComponent(e.value)}').join('&');
    return '${client.apiBaseUrl}/auth/oauth?$query';
  }

  Future<Map<String, dynamic>> callback(String provider, String code, {String? state}) async {
    return client.post<Map<String, dynamic>>(
      '/auth/oauth/callback',
      {'provider': provider, 'code': code, if (state != null) 'state': state},
    );
  }

  Future<void> login(String username, String password) async {
    final res = await client.post<Map<String, dynamic>>(
      '/auth/login',
      {'username': username, 'password': password},
    );
    final token = res['token'] as String;
    await client.setToken(token);
  }

  Future<void> register(String email, String password, String username) async {
    await client.post<Map<String, dynamic>>(
      '/auth/register',
      {'email': email, 'password': password, 'username': username},
    );
  }

  Future<void> logout() async {
    try {
      await client.post('/auth/logout', null);
    } finally {
      await client.clearToken();
    }
  }

  Future<User> getMe() async {
    return client.get<Map<String, dynamic>>('/auth/me')
        .then((json) => User.fromJson(json));
  }
}

class UserModule extends Module {
  UserModule(super.client);

  Future<List<Session>> listSessions() async {
    final json = await client.get<List<dynamic>>('/users/sessions');
    return json.map((e) => Session.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<void> revokeSession(String sessionId) async {
    await client.delete('/users/sessions/$sessionId');
  }

  Future<void> updateProfile({String? username, String? avatarUrl}) async {
    await client.put('/users/profile', {
      if (username != null) 'username': username,
      if (avatarUrl != null) 'avatar_url': avatarUrl,
    });
  }
}
