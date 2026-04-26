import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../providers/auth_provider.dart';

class OAuthCallbackScreen extends ConsumerStatefulWidget {
  final String provider;

  const OAuthCallbackScreen({super.key, required this.provider});

  @override
  ConsumerState<OAuthCallbackScreen> createState() => _OAuthCallbackScreenState();
}

class _OAuthCallbackScreenState extends ConsumerState<OAuthCallbackScreen> {
  @override
  void initState() {
    super.initState();
    _handleCallback();
  }

  Future<void> _handleCallback() async {
    // On web, extract code from URL query params
    // On mobile, the callback URL would have been intercepted by url_launcher
    final code = _extractCode();

    if (code == null || code.isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('OAuth 授权失败：缺少授权码')),
        );
        context.go('/login');
      }
      return;
    }

    try {
      await ref.read(authProvider.notifier).handleOAuthCallback(widget.provider, code);
      if (mounted) context.go('/notes');
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('OAuth 登录失败: $e')),
        );
        context.go('/login');
      }
    }
  }

  String? _extractCode() {
    // For web: parse the current URL
    // For mobile: the code should be passed via route parameters
    return null; // Fallback; actual implementation depends on platform
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            CircularProgressIndicator(color: Theme.of(context).colorScheme.primary),
            const SizedBox(height: 16),
            Text(
              '正在登录...',
              style: Theme.of(context).textTheme.titleMedium,
            ),
          ],
        ),
      ),
    );
  }
}
