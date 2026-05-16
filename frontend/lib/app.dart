import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:go_router/go_router.dart';
import 'package:dio/dio.dart' show Response;
import 'config/app_config.dart';
import 'config/theme.dart';
import 'presentation/providers/auth_provider.dart';
import 'presentation/screens/auth/login_screen.dart';
import 'presentation/screens/auth/register_screen.dart';
import 'presentation/screens/auth/splash_screen.dart';
import 'presentation/screens/home/home_screen.dart';
import 'presentation/screens/note/note_editor_screen.dart';
import 'presentation/screens/note/version_history/version_history_screen.dart';
import 'presentation/screens/search/search_screen.dart';
import 'presentation/screens/ai/ai_chat_screen.dart';
import 'presentation/screens/profile/profile_screen.dart';
import 'presentation/screens/settings/settings_screen.dart';
import 'presentation/screens/devices/devices_screen.dart';
import 'presentation/screens/keys/key_manage_screen.dart';
import 'presentation/screens/photos/photos_screen.dart';
import 'presentation/widgets/ai_floating_dialog.dart';
import 'core/crypto/key_store.dart' as key_store;

final _rootNavigatorKey = GlobalKey<NavigatorState>();

final router = GoRouter(
  navigatorKey: _rootNavigatorKey,
  initialLocation: '/splash',
  routes: [
    GoRoute(
      path: '/splash',
      builder: (context, state) => const SplashScreen(),
    ),
    GoRoute(
      path: '/login',
      builder: (context, state) => const LoginScreen(),
    ),
    GoRoute(
      path: '/register',
      builder: (context, state) => const RegisterScreen(),
    ),
    ShellRoute(
      builder: (context, state, child) => MainShell(child: child),
      routes: [
        GoRoute(
          path: '/',
          redirect: (context, state) => '/notes',
        ),
        GoRoute(
          path: '/notes',
          builder: (context, state) => const HomeScreen(),
          routes: [
            GoRoute(
              path: 'note/:noteId',
              builder: (context, state) {
                final noteId = state.pathParameters['noteId']!;
                return NoteEditorScreen(noteId: noteId);
              },
              routes: [
                GoRoute(
                  path: 'versions',
                  builder: (context, state) {
                    final noteId = state.pathParameters['noteId']!;
                    return VersionHistoryScreen(noteId: noteId);
                  },
                ),
              ],
            ),
          ],
        ),
        GoRoute(
          path: '/ai',
          builder: (context, state) => const AIChatScreen(),
        ),
        GoRoute(
          path: '/profile',
          builder: (context, state) => const ProfileScreen(),
        ),
        GoRoute(
          path: '/search',
          builder: (context, state) => const SearchScreen(),
        ),
        GoRoute(
          path: '/settings',
          builder: (context, state) => const SettingsScreen(),
        ),
        GoRoute(
          path: '/devices',
          builder: (context, state) => const DevicesScreen(),
        ),
        GoRoute(
          path: '/keys',
          builder: (context, state) => const KeyManageScreen(),
        ),
        GoRoute(
          path: '/photos',
          builder: (context, state) => const PhotosScreen(),
        ),
      ],
    ),
  ],
);

final routerProvider = Provider<GoRouter>((ref) => router);

class KrasisApp extends ConsumerWidget {
  const KrasisApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeMode = ref.watch(themeModeProvider);

    return MaterialApp.router(
      title: AppConfig.appName,
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      themeMode: themeMode,
      routerConfig: router,
      localizationsDelegates: const [
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        FlutterQuillLocalizations.delegate,
      ],
      supportedLocales: const [Locale('zh'), Locale('en')],
    );
  }
}

class MainShell extends ConsumerStatefulWidget {
  final Widget child;
  const MainShell({super.key, required this.child});

  @override
  ConsumerState<MainShell> createState() => _MainShellState();
}

class _MainShellState extends ConsumerState<MainShell> {
  bool _sidebarCollapsed = false;
  bool _aiDialogVisible = false;
  bool _keySyncInitialized = false;

  @override
  void initState() {
    super.initState();
    // Initialize key sync and try to auto-sync keys after auth.
    _initKeySync();
  }

  void _initKeySync() {
    // Set up the API caller for key_store so it can sync keys via the server.
    key_store.setSyncApiCaller((String method, String path,
        {Map<String, dynamic>? data}) async {
      final api = ref.read(apiClientProvider);
      Response<Map<String, dynamic>> response;
      switch (method) {
        case 'GET':
          response = await api.get(path);
          break;
        case 'PUT':
          response = await api.put(path, data: data);
          break;
        case 'DELETE':
          response = await api.delete(path);
          break;
        default:
          throw UnsupportedError('Unsupported method: $method');
      }
      return response.data ?? {};
    });
  }

  Future<void> _tryAutoSyncKeys() async {
    if (_keySyncInitialized) return;
    _keySyncInitialized = true;
    await key_store.autoSyncKeys();
  }

  @override
  Widget build(BuildContext context) {
    // Trigger key sync once when the user is authenticated
    final authState = ref.watch(authProvider);
    if (authState.isAuthenticated && !_keySyncInitialized) {
      // Use post-frame callback to avoid build-side effects
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _tryAutoSyncKeys();
      });
    }
    final width = MediaQuery.of(context).size.width;
    final isMobile = width < 768;
    final isTablet = width >= 768 && width < 1024;

    if (isMobile) {
      return _buildMobileLayout(context);
    }

    return _buildDesktopLayout(context, isTablet);
  }

  Widget _buildMobileLayout(BuildContext context) {
    return Scaffold(
      body: widget.child,
      bottomNavigationBar: NavigationBar(
        destinations: const [
          NavigationDestination(icon: Icon(Icons.note_outlined), selectedIcon: Icon(Icons.note), label: '笔记'),
          NavigationDestination(icon: Icon(Icons.photo_library_outlined), selectedIcon: Icon(Icons.photo_library), label: '照片'),
          NavigationDestination(icon: Icon(Icons.smart_toy_outlined), selectedIcon: Icon(Icons.smart_toy), label: 'AI'),
          NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: '我的'),
        ],
        selectedIndex: _currentIndex(context),
        onDestinationSelected: (i) => _navigate(context, i),
      ),
    );
  }

  Widget _buildDesktopLayout(BuildContext context, bool isTablet) {
    final sidebarWidth = _sidebarCollapsed ? 60.0 : 240.0;

    return Scaffold(
      body: Stack(
        children: [
          Row(
            children: [
              AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                width: sidebarWidth,
                constraints: const BoxConstraints(minWidth: 60, maxWidth: 240),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.surfaceVariant.withOpacity(0.4),
                  border: Border(
                    right: BorderSide(color: Theme.of(context).dividerColor, width: 0.5),
                  ),
                ),
                child: _buildSidebar(context, isTablet),
              ),
              Expanded(
                child: Column(
                  children: [
                    _buildTopBar(context),
                    Expanded(
                      child: Container(
                        color: Theme.of(context).colorScheme.surfaceVariant.withOpacity(0.3),
                        child: widget.child,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          if (_aiDialogVisible)
            AIFloatingDialog(
              onClose: () => setState(() => _aiDialogVisible = false),
            ),
          Positioned(
            right: 16,
            bottom: 16,
            child: FloatingActionButton(
              heroTag: 'ai_fab',
              onPressed: () => setState(() => _aiDialogVisible = !_aiDialogVisible),
              child: Icon(_aiDialogVisible ? Icons.close : Icons.smart_toy_outlined),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSidebar(BuildContext context, bool isTablet) {
    final currentLocation = GoRouterState.of(context).matchedLocation;

    return Column(
      children: [
        SizedBox(
          height: 56,
          child: Row(
            children: [
              if (!_sidebarCollapsed)
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                    child: Text(
                      'Krasis',
                      style: Theme.of(context).textTheme.titleLarge?.copyWith(
                            fontWeight: FontWeight.bold,
                            color: Theme.of(context).colorScheme.primary,
                          ),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                )
              else
                const SizedBox(width: 56),
              if (!isTablet)
                IconButton(
                  icon: Icon(_sidebarCollapsed ? Icons.chevron_right : Icons.chevron_left),
                  onPressed: () => setState(() => _sidebarCollapsed = !_sidebarCollapsed),
                  iconSize: 20,
                ),
            ],
          ),
        ),
        const Divider(height: 1),
        Padding(
          padding: const EdgeInsets.all(8),
          child: _sidebarCollapsed
              ? IconButton(
                  icon: const Icon(Icons.add),
                  tooltip: '新建笔记',
                  onPressed: () => context.go('/notes/note/new'),
                )
              : FilledButton.icon(
                  onPressed: () => context.go('/notes/note/new'),
                  icon: const Icon(Icons.add, size: 18),
                  label: const Text('新建笔记'),
                ),
        ),
        const Divider(height: 1),
        Expanded(
          child: ListView(
            padding: const EdgeInsets.symmetric(vertical: 8),
            children: [
              _SidebarTile(
                icon: Icons.article_outlined,
                activeIcon: Icons.article,
                label: '全部笔记',
                collapsed: _sidebarCollapsed,
                selected: currentLocation.startsWith('/notes'),
                onTap: () => context.go('/notes'),
              ),
              _SidebarTile(
                icon: Icons.smart_toy_outlined,
                activeIcon: Icons.smart_toy,
                label: 'AI 对话',
                collapsed: _sidebarCollapsed,
                selected: currentLocation.startsWith('/ai'),
                onTap: () => context.go('/ai'),
              ),
              _SidebarTile(
                icon: Icons.search_outlined,
                activeIcon: Icons.search,
                label: '搜索',
                collapsed: _sidebarCollapsed,
                selected: currentLocation.startsWith('/search'),
                onTap: () => context.go('/search'),
              ),
              _SidebarTile(
                icon: Icons.photo_library_outlined,
                activeIcon: Icons.photo_library,
                label: '照片',
                collapsed: _sidebarCollapsed,
                selected: currentLocation.startsWith('/photos'),
                onTap: () => context.go('/photos'),
              ),
              const Divider(),
              _SidebarTile(
                icon: Icons.person_outline,
                activeIcon: Icons.person,
                label: '个人中心',
                collapsed: _sidebarCollapsed,
                selected: currentLocation.startsWith('/profile'),
                onTap: () => context.go('/profile'),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildTopBar(BuildContext context) {
    return Container(
      height: 48,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: Theme.of(context).dividerColor, width: 0.5)),
      ),
      child: Row(
        children: [
          IconButton(
            icon: const Icon(Icons.search),
            onPressed: () => context.push('/search'),
            tooltip: '搜索',
          ),
        ],
      ),
    );
  }

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    if (location.startsWith('/photos')) return 1;
    if (location.startsWith('/ai')) return 2;
    if (location.startsWith('/profile')) return 3;
    return 0;
  }

  void _navigate(BuildContext context, int index) {
    switch (index) {
      case 0:
        context.go('/notes');
        break;
      case 1:
        context.go('/photos');
        break;
      case 2:
        context.go('/ai');
        break;
      case 3:
        context.go('/profile');
        break;
    }
  }
}

class _SidebarTile extends StatelessWidget {
  final IconData icon;
  final IconData activeIcon;
  final String label;
  final bool collapsed;
  final bool selected;
  final VoidCallback onTap;

  const _SidebarTile({
    required this.icon,
    required this.activeIcon,
    required this.label,
    required this.collapsed,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    if (collapsed) {
      return IconButton(
        icon: Icon(selected ? activeIcon : icon),
        tooltip: label,
        onPressed: onTap,
        style: IconButton.styleFrom(
          backgroundColor: selected ? Theme.of(context).colorScheme.primaryContainer : null,
        ),
      );
    }

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      child: ListTile(
        leading: Icon(selected ? activeIcon : icon),
        title: Text(label),
        selected: selected,
        onTap: onTap,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
        contentPadding: const EdgeInsets.symmetric(horizontal: 12),
      ),
    );
  }
}
