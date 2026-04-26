class UserModel {
  final String id;
  final String email;
  final String username;
  final String avatarUrl;
  final String role;
  final int status;
  final DateTime createdAt;

  UserModel({
    required this.id,
    required this.email,
    required this.username,
    this.avatarUrl = '',
    this.role = 'viewer',
    this.status = 1,
    required this.createdAt,
  });

  factory UserModel.fromJson(Map<String, dynamic> json) => UserModel(
        id: json['id'] as String,
        email: json['email'] as String,
        username: json['username'] as String,
        avatarUrl: json['avatar_url'] as String? ?? '',
        role: json['role'] as String? ?? 'viewer',
        status: json['status'] as int? ?? 1,
        createdAt: DateTime.parse(json['created_at'] as String),
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'email': email,
        'username': username,
        'avatar_url': avatarUrl,
        'role': role,
        'status': status,
        'created_at': createdAt.toIso8601String(),
      };

  UserModel copyWith({
    String? username,
    String? avatarUrl,
  }) {
    return UserModel(
      id: id,
      email: email,
      username: username ?? this.username,
      avatarUrl: avatarUrl ?? this.avatarUrl,
      role: role,
      status: status,
      createdAt: createdAt,
    );
  }
}

class SessionModel {
  final String sessionId;
  final String deviceName;
  final String deviceType;
  final String ipAddress;
  final String userAgent;
  final DateTime lastActiveAt;
  final DateTime createdAt;
  final bool isCurrent;

  SessionModel({
    required this.sessionId,
    required this.deviceName,
    required this.deviceType,
    required this.ipAddress,
    required this.userAgent,
    required this.lastActiveAt,
    required this.createdAt,
    this.isCurrent = false,
  });

  factory SessionModel.fromJson(Map<String, dynamic> json) => SessionModel(
        sessionId: json['session_id'] as String,
        deviceName: json['device_name'] as String,
        deviceType: json['device_type'] as String,
        ipAddress: json['ip_address'] as String,
        userAgent: json['user_agent'] as String,
        lastActiveAt: DateTime.parse(json['last_active_at'] as String),
        createdAt: DateTime.parse(json['created_at'] as String),
        isCurrent: json['is_current'] as bool? ?? false,
      );
}
