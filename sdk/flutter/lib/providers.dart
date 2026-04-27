library krasis_sdk_providers;

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'krasis_sdk.dart';

/// Configuration for initializing the Krasis SDK provider.
class KrasisConfig {
  final String baseUrl;
  final String? initialToken;

  const KrasisConfig({
    required this.baseUrl,
    this.initialToken,
  });
}

/// Provider for the KrasisSDK instance.
///
/// Usage:
/// ```dart
/// runApp(
///   ProviderScope(
///     overrides: [
///       krasisConfigProvider.overrideWith(
///         () => const KrasisConfig(baseUrl: 'https://api.example.com'),
///       ),
///     ],
///     child: MyApp(),
///   ),
/// );
/// ```
final krasisConfigProvider = Provider<KrasisConfig>((ref) {
  throw UnimplementedError('Override krasisConfigProvider with your API base URL');
});

final krasisSdkProvider = AutoDisposeNotifierProvider<KrasisSdkNotifier, KrasisSDK?>((ref) {
  final config = ref.watch(krasisConfigProvider);
  return KrasisSdkNotifier(baseUrl: config.baseUrl, initialToken: config.initialToken);
});

class KrasisSdkNotifier extends Notifier<KrasisSDK?> {
  final String baseUrl;
  final String? initialToken;

  KrasisSdkNotifier({required this.baseUrl, this.initialToken});

  @override
  KrasisSDK? build() {
    final sdk = KrasisSDK(baseUrl: baseUrl, token: initialToken);
    ref.onDispose(() => sdk.dispose());
    return sdk;
  }

  void setToken(String token) {
    state?.setToken(token);
  }

  Future<void> login(String username, String password) async {
    final sdk = state;
    if (sdk == null) return;
    final result = await sdk.auth.login(username, password);
    sdk.setToken(result.token);
    state = sdk; // trigger rebuild
  }

  Future<void> logout() async {
    await state?.clearToken();
    state = null;
  }
}

/// Provider for auth state.
final authStateProvider = AutoDisposeProvider<bool>((ref) {
  final sdk = ref.watch(krasisSdkProvider);
  return sdk?.isAuthenticated ?? false;
});

/// Provider for the auth module.
final authModuleProvider = AutoDisposeProvider<AuthModule?>((ref) {
  return ref.watch(krasisSdkProvider)?.auth;
});

/// Provider for the notes module.
final notesModuleProvider = AutoDisposeProvider<NotesModule?>((ref) {
  return ref.watch(krasisSdkProvider)?.notes;
});

/// Provider for the AI module.
final aiModuleProvider = AutoDisposeProvider<AIModule?>((ref) {
  return ref.watch(krasisSdkProvider)?.ai;
});

/// Provider for the search module.
final searchModuleProvider = AutoDisposeProvider<SearchModule?>((ref) {
  return ref.watch(krasisSdkProvider)?.search;
});

/// Provider for the collab module.
final collabModuleProvider = AutoDisposeProvider<CollabModule?>((ref) {
  return ref.watch(krasisSdkProvider)?.collab;
});
