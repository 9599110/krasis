import 'dart:async';
import 'package:flutter/services.dart';
import 'package:shared_preferences/shared_preferences.dart';

class SecureStorage {
  static const _tokenKey = 'krasis_token';
  static const _userIdKey = 'krasis_user_id';
  static const _prefsTimeout = Duration(seconds: 2);
  static String? _cachedToken;

  Future<String?> getAccessToken() async {
    if (_cachedToken != null) return _cachedToken;
    try {
      final prefs = await SharedPreferences.getInstance().timeout(_prefsTimeout);
      _cachedToken = prefs.getString(_tokenKey);
      return _cachedToken;
    } on PlatformException {
      return null;
    } on TimeoutException {
      return null;
    }
  }

  Future<void> saveAccessToken(String token) async {
    _cachedToken = token;
    try {
      final prefs = await SharedPreferences.getInstance().timeout(_prefsTimeout);
      await prefs.setString(_tokenKey, token).timeout(_prefsTimeout);
    } on PlatformException {
      // ignore
    } on TimeoutException {
      // ignore
    }
  }

  Future<void> clearAccessToken() async {
    _cachedToken = null;
    try {
      final prefs = await SharedPreferences.getInstance().timeout(_prefsTimeout);
      await prefs.remove(_tokenKey).timeout(_prefsTimeout);
      await prefs.remove(_userIdKey).timeout(_prefsTimeout);
    } on PlatformException {
      // ignore
    } on TimeoutException {
      // ignore
    }
  }

  Future<String?> getUserId() async {
    try {
      final prefs = await SharedPreferences.getInstance().timeout(_prefsTimeout);
      return prefs.getString(_userIdKey);
    } on PlatformException {
      return null;
    } on TimeoutException {
      return null;
    }
  }

  Future<void> saveUserId(String userId) async {
    try {
      final prefs = await SharedPreferences.getInstance().timeout(_prefsTimeout);
      await prefs.setString(_userIdKey, userId).timeout(_prefsTimeout);
    } on PlatformException {
      // ignore
    } on TimeoutException {
      // ignore
    }
  }
}
