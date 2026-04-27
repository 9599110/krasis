import 'dart:io' show Platform;

class AppConfig {
  static String get apiBaseUrl => String.fromEnvironment(
        'API_BASE_URL',
        defaultValue: 'http://192.168.43.78:9091',
      );

  static String get wsBaseUrl => String.fromEnvironment(
        'WS_BASE_URL',
        defaultValue: 'ws://192.168.43.78:9091',
      );

  static const String appName = 'Krasis';
  static const int connectTimeoutMs = 30000;
  static const int receiveTimeoutMs = 30000;
  static const int maxRetries = 3;
}
