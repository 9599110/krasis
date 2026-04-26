import 'dart:async';
import 'package:dio/dio.dart';
import '../storage/secure_storage.dart';
import '../errors/exceptions.dart';

class AuthInterceptor extends Interceptor {
  final SecureStorage _storage;

  AuthInterceptor(this._storage);

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = await _storage.getAccessToken();
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    if (err.response?.statusCode == 401) {
      // Token expired - let the provider handle it
      handler.next(err.copyWith(error: const AuthenticationException()));
      return;
    }
    handler.next(_mapError(err));
  }

  DioException _mapError(DioException err) {
    switch (err.response?.statusCode) {
      case 409:
        return err.copyWith(error: VersionConflictException(currentVersion: 0));
      case 429:
        return err.copyWith(error: RateLimitException());
      case 404:
        return err.copyWith(error: NotFoundException());
      default:
        return err;
    }
  }
}

class RetryInterceptor extends Interceptor {
  static const _maxRetries = 3;
  final Dio _dio;

  RetryInterceptor(this._dio);

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final retryCount = err.requestOptions.extra['retryCount'] as int? ?? 0;

    if (!_shouldRetry(err) || retryCount >= _maxRetries) {
      handler.next(err);
      return;
    }

    err.requestOptions.extra['retryCount'] = retryCount + 1;

    await Future.delayed(Duration(seconds: retryCount + 1));
    try {
      final response = await _dio.fetch(err.requestOptions);
      handler.resolve(response);
    } catch (e) {
      handler.next(err);
    }
  }

  bool _shouldRetry(DioException err) {
    // Don't retry timeouts - the caller's timeout() already handles those.
    // Only retry connection errors and server errors.
    return err.type == DioExceptionType.connectionError ||
           (err.response?.statusCode != null && err.response!.statusCode! >= 500);
  }
}
