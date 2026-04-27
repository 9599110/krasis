import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/network/api_client.dart';
import '../../../core/storage/secure_storage.dart';
import '../../../data/models/user_model.dart';
import '../../../config/app_config.dart';

final secureStorageProvider = Provider<SecureStorage>((ref) => SecureStorage());

final apiClientProvider = Provider<ApiClient>((ref) {
  return ApiClient(secureStorage: ref.watch(secureStorageProvider));
});

class AuthState {
  final bool isAuthenticated;
  final UserModel? user;
  final bool isLoading;
  final String? error;

  const AuthState({
    this.isAuthenticated = false,
    this.user,
    this.isLoading = false,
    this.error,
  });

  const AuthState.unauthenticated()
      : isAuthenticated = false,
        user = null,
        isLoading = false,
        error = null;

  const AuthState.loading()
      : isAuthenticated = false,
        user = null,
        isLoading = true,
        error = null;

  AuthState.authenticated(UserModel u)
      : isAuthenticated = true,
        user = u,
        isLoading = false,
        error = null;

  AuthState.withError(String e)
      : isAuthenticated = false,
        user = null,
        isLoading = false,
        error = e;
}

class AuthNotifier extends StateNotifier<AuthState> {
  final ApiClient _api;
  final SecureStorage _storage;

  AuthNotifier(this._api, this._storage) : super(const AuthState.unauthenticated());

  Future<void> init() async {
    final token = await _storage.getAccessToken();
    if (token == null) {
      state = const AuthState.unauthenticated();
      return;
    }

    try {
      state = const AuthState.loading();
      final response = await _api.get('/auth/me');
      final data = response.data!['data'] as Map<String, dynamic>;
      final user = UserModel.fromJson(data);
      state = AuthState.authenticated(user);
    } catch (e) {
      await _storage.clearAccessToken();
      state = const AuthState.unauthenticated();
    }
  }

  Future<void> login(String username, String password) async {
    debugPrint('[auth] login start, state=loading');
    state = const AuthState.loading();
    try {
      debugPrint('[auth] calling POST /auth/login');
      final response = await _api.post('/auth/login', data: {
        'username': username,
        'password': password,
      });
      debugPrint('[auth] login response received, statusCode=${response.statusCode}');
      final data = response.data!['data'] as Map<String, dynamic>;
      final token = data['token'] as String;
      debugPrint('[auth] token received, length=${token.length}');
      // Make token immediately available to subsequent requests (don't depend on prefs IO).
      _api.dio.options.headers['Authorization'] = 'Bearer $token';
      final userJson = (data['user'] as Map?)?.cast<String, dynamic>() ?? <String, dynamic>{};
      debugPrint('[auth] userJson keys=${userJson.keys}, isEmpty=${userJson.isEmpty}');
      if (userJson.isNotEmpty) {
        debugPrint('[auth] setting authenticated state');
        state = AuthState.authenticated(UserModel.fromJson(userJson));
        debugPrint('[auth] login success -> authenticated');
        unawaited(_storage.saveAccessToken(token));
      } else {
        unawaited(_storage.saveAccessToken(token));
        debugPrint('[auth] login success -> init() fallback');
        await init();
      }
      debugPrint('[auth] login method complete');
    } catch (e) {
      debugPrint('[auth] login error: $e');
      state = AuthState.withError(e.toString());
      rethrow;
    }
  }

  Future<void> register(String email, String password, String username) async {
    await _api.post('/auth/register', data: {
      'email': email,
      'password': password,
      'username': username,
    });
  }

  String getOAuthUrl(String provider) {
    return '${AppConfig.apiBaseUrl}/auth/oauth?provider=$provider';
  }

  Future<void> handleOAuthCallback(String provider, String code) async {
    final response = await _api.post('/auth/oauth/callback', data: {
      'provider': provider,
      'code': code,
    });
    final data = response.data!['data'] as Map<String, dynamic>;
    final token = data['token'] as String;
    await _storage.saveAccessToken(token);
    await init();
  }

  Future<void> logout() async {
    try {
      await _api.post('/auth/logout');
    } finally {
      await _storage.clearAccessToken();
      state = const AuthState.unauthenticated();
    }
  }

  Future<void> refreshToken() async {
    await init();
  }

  Future<void> updateProfile({String? name}) async {
    final data = <String, dynamic>{};
    if (name != null) data['name'] = name;
    final response = await _api.put('/user/profile', data: data);
    final userData = response.data!['data'] as Map<String, dynamic>;
    state = AuthState.authenticated(UserModel.fromJson(userData));
  }
}

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(ref.watch(apiClientProvider), ref.watch(secureStorageProvider));
});

final themeModeProvider = StateProvider<ThemeMode>((ref) => ThemeMode.system);
