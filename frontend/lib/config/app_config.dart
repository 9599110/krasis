import 'dart:io' show Platform;

class AppConfig {
  static String get _host => Platform.isAndroid ? '10.0.2.2' : '127.0.0.1';

  static String get apiBaseUrl => String.fromEnvironment(
        'API_BASE_URL',
        defaultValue: 'http://$_host:9091',
      );

  static String get wsBaseUrl => String.fromEnvironment(
        'WS_BASE_URL',
        defaultValue: 'ws://$_host:9091',
      );

  static const String appName = 'Krasis';
  static const int connectTimeoutMs = 30000;
  static const int receiveTimeoutMs = 30000;
  static const int maxRetries = 3;
}
