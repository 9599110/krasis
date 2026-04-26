import 'dart:async';
import 'package:dio/dio.dart';
import '../storage/secure_storage.dart';
import '../errors/exceptions.dart';
import 'api_interceptor.dart';
import '../../config/app_config.dart';

class ApiClient {
  final Dio _dio;

  ApiClient({required SecureStorage secureStorage})
      : _dio = Dio(BaseOptions(
          baseUrl: AppConfig.apiBaseUrl,
          connectTimeout: const Duration(milliseconds: AppConfig.connectTimeoutMs),
          receiveTimeout: const Duration(milliseconds: AppConfig.receiveTimeoutMs),
          headers: {'Content-Type': 'application/json'},
        )) {
    _dio.interceptors.add(AuthInterceptor(secureStorage));
    _dio.interceptors.add(RetryInterceptor(_dio));
    _dio.interceptors.add(LogInterceptor(
      request: true,
      requestHeader: true,
      requestBody: true,
      responseHeader: true,
      responseBody: true,
      error: true,
    ));
  }

  Dio get dio => _dio;

  T _extractData<T>(Response<Map<String, dynamic>> response) {
    final data = response.data?['data'];
    if (data == null) {
      throw ApiException(message: 'Empty response');
    }
    return data as T;
  }

  Future<Response<Map<String, dynamic>>> get(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) async {
    try {
      return await _dio.get<Map<String, dynamic>>(
        path,
        queryParameters: queryParameters,
      );
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<Response<Map<String, dynamic>>> post(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    try {
      return await _dio.post<Map<String, dynamic>>(
        path,
        data: data,
        queryParameters: queryParameters,
      );
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<Response<Map<String, dynamic>>> put(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
    Map<String, dynamic>? headers,
  }) async {
    try {
      return await _dio.put<Map<String, dynamic>>(
        path,
        data: data,
        queryParameters: queryParameters,
        options: headers != null ? Options(headers: headers) : null,
      );
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<Response<Map<String, dynamic>>> delete(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) async {
    try {
      return await _dio.delete<Map<String, dynamic>>(
        path,
        queryParameters: queryParameters,
      );
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Exception _handleError(DioException e) {
    switch (e.response?.statusCode) {
      case 401:
        return const AuthenticationException();
      case 404:
        return const NotFoundException();
      case 409:
        return VersionConflictException(currentVersion: 0);
      case 429:
        return RateLimitException();
      default:
        final body = e.response?.data;
        if (body is Map) {
          return ApiException(
            code: body['code'] as String?,
            message: body['message'] as String? ?? 'Request failed',
            statusCode: e.response?.statusCode,
          );
        }
        return ApiException(message: e.message ?? 'Unknown error');
    }
  }
}
