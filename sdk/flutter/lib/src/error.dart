class SDKException implements Exception {
  final String message;
  final int? statusCode;
  final String? code;

  SDKException({required this.message, this.statusCode, this.code});

  @override
  String toString() => 'SDKException: $message (code: $code, status: $statusCode)';
}

class VersionConflictException extends SDKException {
  final int serverVersion;

  VersionConflictException({required this.serverVersion})
      : super(message: 'Version conflict', code: 'VERSION_CONFLICT', statusCode: 409);
}

class RateLimitException extends SDKException {
  final DateTime? retryAfter;

  RateLimitException({this.retryAfter})
      : super(message: 'Rate limit exceeded', code: 'RATE_LIMIT', statusCode: 429);
}

class AuthenticationException extends SDKException {
  AuthenticationException({String? message})
      : super(message: message ?? 'Authentication required', code: 'UNAUTHORIZED', statusCode: 401);
}

class NotFoundException extends SDKException {
  NotFoundException({String? message})
      : super(message: message ?? 'Resource not found', code: 'NOT_FOUND', statusCode: 404);
}
