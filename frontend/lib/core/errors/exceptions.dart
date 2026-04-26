/// Version conflict error (409)
class VersionConflictException implements Exception {
  final int currentVersion;
  final String message;

  const VersionConflictException({required this.currentVersion, this.message = 'Version conflict'});

  @override
  String toString() => message;
}

/// Rate limit error (429)
class RateLimitException implements Exception {
  const RateLimitException();
  @override
  String toString() => 'Rate limit exceeded';
}

/// Authentication error (401)
class AuthenticationException implements Exception {
  const AuthenticationException();
  @override
  String toString() => 'Authentication required';
}

/// Not found error (404)
class NotFoundException implements Exception {
  final String resource;
  const NotFoundException([this.resource = 'Resource']);

  @override
  String toString() => '$resource not found';
}

/// General API error
class ApiException implements Exception {
  final String? code;
  final String message;
  final int? statusCode;

  ApiException({this.code, required this.message, this.statusCode});

  @override
  String toString() => 'API Error [$code]: $message';
}
