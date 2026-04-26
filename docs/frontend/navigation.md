# Flutter 路由与导航设计

## 1. 路由架构

### 1.1 路由配置

```dart
// lib/config/routes.dart

enum AppRoute {
  splash('/'),
  login('/login'),
  oauthCallback('/login/oauth/callback'),
  
  // 主页面
  home('/home'),
  noteList('/notes'),
  noteEditor('/notes/:noteId'),
  folder('/folders'),
  search('/search'),
  aiChat('/ai'),
  aiHistory('/ai/history'),
  aiConversation('/ai/:conversationId'),
  
  // 个人中心
  profile('/profile'),
  devices('/profile/devices'),
  settings('/profile/settings'),
  
  // 分享
  shareView('/share/:token'),
  
  // 管理
  admin('/admin'),
  adminUsers('/admin/users'),
  adminUserDetail('/admin/users/:userId'),
  
  // 错误页面
  notFound('/404'),
  error('/error');

  final String path;
  const AppRoute(this.path);
}

@Riverpod
GoRouter router(RouterRef ref) {
  // 监听认证状态
  final authNotifier = ref.watch(authNotifierProvider);
  final isAuthenticated = authNotifier.maybeWhen(
    data: (state) => state.isAuthenticated,
    orElse: () => false,
  );
  
  final isLoading = authNotifier.isLoading;

  return GoRouter(
    initialLocation: AppRoute.splash.path,
    debugLogDiagnostics: true,
    routes: [
      // 闪屏页
      GoRoute(
        path: AppRoute.splash.path,
        name: 'splash',
        builder: (context, state) => const SplashScreen(),
      ),
      
      // 登录相关
      GoRoute(
        path: AppRoute.login.path,
        name: 'login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: AppRoute.oauthCallback.path,
        name: 'oauth-callback',
        builder: (context, state) {
          final provider = state.uri.queryParameters['provider'] ?? 'github';
          final code = state.uri.queryParameters['code'] ?? '';
          return OAuthCallbackScreen(provider: provider, code: code);
        },
      ),
      
      // 分享页面（无需登录）
      GoRoute(
        path: AppRoute.shareView.path,
        name: 'share-view',
        builder: (context, state) {
          final token = state.pathParameters['token']!;
          return ShareViewScreen(token: token);
        },
      ),
      
      // 主应用 Shell
      ShellRoute(
        builder: (context, state, child) {
          return MainShell(
            currentPath: state.uri.path,
            child: child,
          );
        },
        routes: [
          // 首页/笔记列表
          GoRoute(
            path: AppRoute.home.path,
            name: 'home',
            redirect: (context, state) => AppRoute.noteList.path,
          ),
          GoRoute(
            path: AppRoute.noteList.path,
            name: 'note-list',
            builder: (context, state) => const NoteListScreen(),
            routes: [
              GoRoute(
                path: ':noteId',
                name: 'note-editor',
                builder: (context, state) {
                  final noteId = state.pathParameters['noteId']!;
                  return NoteEditorScreen(noteId: noteId);
                },
                routes: [
                  // 笔记子路由
                  GoRoute(
                    path: 'versions',
                    name: 'note-versions',
                    builder: (context, state) {
                      final noteId = state.pathParameters['noteId']!;
                      return NoteVersionsScreen(noteId: noteId);
                    },
                  ),
                  GoRoute(
                    path: 'share',
                    name: 'note-share',
                    builder: (context, state) {
                      final noteId = state.pathParameters['noteId']!;
                      return NoteShareScreen(noteId: noteId);
                    },
                  ),
                ],
              ),
            ],
          ),
          
          // 文件夹
          GoRoute(
            path: AppRoute.folder.path,
            name: 'folders',
            builder: (context, state) => const FolderScreen(),
            routes: [
              GoRoute(
                path: ':folderId',
                name: 'folder-detail',
                builder: (context, state) {
                  final folderId = state.pathParameters['folderId']!;
                  return FolderDetailScreen(folderId: folderId);
                },
              ),
            ],
          ),
          
          // 搜索
          GoRoute(
            path: AppRoute.search.path,
            name: 'search',
            builder: (context, state) {
              final keyword = state.uri.queryParameters['q'];
              return SearchScreen(keyword: keyword);
            },
          ),
          
          // AI 对话
          GoRoute(
            path: AppRoute.ai.path,
            name: 'ai-chat',
            builder: (context, state) => const AIChatScreen(),
            routes: [
              GoRoute(
                path: 'history',
                name: 'ai-history',
                builder: (context, state) => const AIHistoryScreen(),
              ),
              GoRoute(
                path: ':conversationId',
                name: 'ai-conversation',
                builder: (context, state) {
                  final conversationId = state.pathParameters['conversationId']!;
                  return AIChatScreen(conversationId: conversationId);
                },
              ),
            ],
          ),
          
          // 个人中心
          GoRoute(
            path: AppRoute.profile.path,
            name: 'profile',
            builder: (context, state) => const ProfileScreen(),
            routes: [
              GoRoute(
                path: 'devices',
                name: 'devices',
                builder: (context, state) => const DevicesScreen(),
              ),
              GoRoute(
                path: 'settings',
                name: 'settings',
                builder: (context, state) => const SettingsScreen(),
              ),
            ],
          ),
          
          // 管理后台
          GoRoute(
            path: AppRoute.admin.path,
            name: 'admin',
            builder: (context, state) => const AdminDashboardScreen(),
            routes: [
              GoRoute(
                path: 'users',
                name: 'admin-users',
                builder: (context, state) => const AdminUsersScreen(),
                routes: [
                  GoRoute(
                    path: ':userId',
                    name: 'admin-user-detail',
                    builder: (context, state) {
                      final userId = state.pathParameters['userId']!;
                      return AdminUserDetailScreen(userId: userId);
                    },
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
      
      // 错误页面
      GoRoute(
        path: AppRoute.notFound.path,
        name: 'not-found',
        builder: (context, state) => const NotFoundScreen(),
      ),
      GoRoute(
        path: AppRoute.error.path,
        name: 'error',
        builder: (context, state) {
          final message = state.uri.queryParameters['message'] ?? 'Unknown error';
          return ErrorScreen(message: message);
        },
      ),
    ],
    
    // 错误处理
    errorBuilder: (context, state) => ErrorScreen(
      message: state.error?.message ?? 'Page not found',
    ),
    
    // 全局重定向
    redirect: (context, state) {
      // 加载中状态，停留在当前页
      if (isLoading) return null;
      
      final isOnAuthPage = state.matchedLocation.startsWith('/login') ||
                           state.matchedLocation == '/';
      final isOnSharePage = state.matchedLocation.startsWith('/share');
      final isOnErrorPage = state.matchedLocation == '/404' ||
                             state.matchedLocation == '/error';
      
      // 未登录，跳转登录
      if (!isAuthenticated && !isOnAuthPage && !isOnSharePage && !isOnErrorPage) {
        return AppRoute.login.path;
      }
      
      // 已登录，在登录页，跳转首页
      if (isAuthenticated && isOnAuthPage) {
        return AppRoute.noteList.path;
      }
      
      return null;
    },
  );
}
```

## 2. 导航组件

### 2.1 主 Shell 组件

```dart
// lib/presentation/widgets/shell/main_shell.dart

class MainShell extends StatelessWidget {
  final String currentPath;
  final Widget child;

  const MainShell({
    super.key,
    required this.currentPath,
    required this.child,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Row(
        children: [
          // 侧边导航栏（桌面端）
          if (Responsive.isDesktop(context))
            _buildSideNav(context),
          
          // 主内容区
          Expanded(child: child),
          
          // AI 助手悬浮按钮/面板
          _buildAIFloatingButton(context),
        ],
      ),
      // 底部导航栏（移动端）
      bottomNavigationBar: Responsive.isMobile(context)
          ? _buildBottomNav(context)
          : null,
    );
  }

  Widget _buildSideNav(BuildContext context) {
    final theme = Theme.of(context);
    
    return Container(
      width: 250,
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          right: BorderSide(color: theme.dividerColor),
        ),
      ),
      child: Column(
        children: [
          // Logo
          Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                Image.asset('assets/images/logo.png', height: 32),
                const SizedBox(width: 12),
                Text('NoteKeeper', style: theme.textTheme.titleLarge),
              ],
            ),
          ),
          
          const Divider(),
          
          // 导航项
          Expanded(
            child: ListView(
              padding: const EdgeInsets.symmetric(vertical: 8),
              children: [
                _NavItem(
                  icon: Icons.note_outlined,
                  label: '笔记',
                  path: '/notes',
                  isSelected: currentPath.startsWith('/notes'),
                ),
                _NavItem(
                  icon: Icons.folder_outlined,
                  label: '文件夹',
                  path: '/folders',
                  isSelected: currentPath.startsWith('/folders'),
                ),
                _NavItem(
                  icon: Icons.search,
                  label: '搜索',
                  path: '/search',
                  isSelected: currentPath == '/search',
                ),
                _NavItem(
                  icon: Icons.smart_toy_outlined,
                  label: 'AI 助手',
                  path: '/ai',
                  isSelected: currentPath.startsWith('/ai'),
                ),
              ],
            ),
          ),
          
          const Divider(),
          
          // 用户信息
          _buildUserSection(context),
        ],
      ),
    );
  }

  Widget _buildUserSection(BuildContext context) {
    final user = ref.watch(authNotifierProvider).valueOrNull?.user;
    
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          CircleAvatar(
            radius: 16,
            backgroundImage: user?.avatarUrl != null
                ? NetworkImage(user!.avatarUrl!)
                : null,
            child: user?.avatarUrl == null
                ? const Icon(Icons.person)
                : null,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(user?.username ?? 'User', style: const TextStyle(fontWeight: FontWeight.w500)),
                Text(user?.email ?? '', style: Theme.of(context).textTheme.bodySmall),
              ],
            ),
          ),
          PopupMenuButton<String>(
            icon: const Icon(Icons.more_vert),
            onSelected: (value) {
              switch (value) {
                case 'profile':
                  context.push('/profile');
                case 'devices':
                  context.push('/profile/devices');
                case 'settings':
                  context.push('/profile/settings');
                case 'logout':
                  ref.read(authNotifierProvider.notifier).logout();
              }
            },
            itemBuilder: (context) => [
              const PopupMenuItem(value: 'profile', child: Text('个人资料')),
              const PopupMenuItem(value: 'devices', child: Text('登录设备')),
              const PopupMenuItem(value: 'settings', child: Text('设置')),
              const PopupMenuDivider(),
              const PopupMenuItem(value: 'logout', child: Text('退出登录')),
            ],
          ),
        ],
      ),
    );
  }
}

class _NavItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final String path;
  final bool isSelected;

  const _NavItem({
    required this.icon,
    required this.label,
    required this.path,
    required this.isSelected,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      child: ListTile(
        leading: Icon(icon),
        title: Text(label),
        selected: isSelected,
        selectedTileColor: theme.colorScheme.primaryContainer,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
        onTap: () => context.go(path),
      ),
    );
  }
}
```

### 2.2 底部导航栏

```dart
Widget _buildBottomNav(BuildContext context) {
  int currentIndex = _getIndexFromPath(currentPath);
  
  return NavigationBar(
    selectedIndex: currentIndex,
    onDestinationSelected: (index) {
      final path = _getPathFromIndex(index);
      context.go(path);
    },
    destinations: const [
      NavigationDestination(
        icon: Icon(Icons.note_outlined),
        selectedIcon: Icon(Icons.note),
        label: '笔记',
      ),
      NavigationDestination(
        icon: Icon(Icons.folder_outlined),
        selectedIcon: Icon(Icons.folder),
        label: '文件夹',
      ),
      NavigationDestination(
        icon: Icon(Icons.search),
        selectedIcon: Icon(Icons.search),
        label: '搜索',
      ),
      NavigationDestination(
        icon: Icon(Icons.smart_toy_outlined),
        selectedIcon: Icon(Icons.smart_toy),
        label: 'AI',
      ),
      NavigationDestination(
        icon: Icon(Icons.person_outlined),
        selectedIcon: Icon(Icons.person),
        label: '我的',
      ),
    ],
  );
}

int _getIndexFromPath(String path) {
  if (path.startsWith('/notes')) return 0;
  if (path.startsWith('/folders')) return 1;
  if (path.startsWith('/search')) return 2;
  if (path.startsWith('/ai')) return 3;
  if (path.startsWith('/profile')) return 4;
  return 0;
}

String _getPathFromIndex(int index) {
  switch (index) {
    case 0: return '/notes';
    case 1: return '/folders';
    case 2: return '/search';
    case 3: return '/ai';
    case 4: return '/profile';
    default: return '/notes';
  }
}
```

## 3. 导航守卫

```dart
// lib/core/router/auth_guard.dart

class AuthGuard {
  final Ref _ref;
  
  AuthGuard(this._ref);
  
  bool get isAuthenticated {
    return _ref.read(authNotifierProvider).valueOrNull?.isAuthenticated ?? false;
  }
  
  bool canAccess(String path) {
    final user = _ref.read(authNotifierProvider).valueOrNull?.user;
    final role = user?.role;
    
    // 管理员功能
    if (path.startsWith('/admin')) {
      return role == 'admin';
    }
    
    return true;
  }
}

// 使用
GoRoute(
  path: '/admin',
  builder: (context, state) {
    final authGuard = ref.read(authGuardProvider);
    
    if (!authGuard.canAccess('/admin')) {
      return const AccessDeniedScreen();
    }
    
    return const AdminScreen();
  },
)
```

## 4. 深度链接配置

### 4.1 Web 配置

```dart
// lib/config/links.dart

// Web 深度链接域名
const deepLinkDomains = [
  'notekeeper.com',
  'www.notekeeper.com',
  'app.notekeeper.com',
];

// 链接模式
const linkPatterns = {
  'note': r'/notes/([a-zA-Z0-9-]+)',
  'share': r'/share/([a-zA-Z0-9]+)',
  'folder': r'/folders/([a-zA-Z0-9-]+)',
};

String? matchDeepLink(Uri uri) {
  // 检查域名
  if (!deepLinkDomains.contains(uri.host)) {
    return null;
  }
  
  final path = uri.path;
  
  // 匹配笔记
  final noteMatch = RegExp(linkPatterns['note']!).firstMatch(path);
  if (noteMatch != null) {
    return '/notes/${noteMatch.group(1)}';
  }
  
  // 匹配分享
  final shareMatch = RegExp(linkPatterns['share']!).firstMatch(path);
  if (shareMatch != null) {
    return '/share/${shareMatch.group(1)}';
  }
  
  return path;
}
```

### 4.2 App Links 配置

```xml
<!-- android/app/src/main/AndroidManifest.xml -->
<intent-filter android:autoVerify="true">
  <action android:name="android.intent.action.VIEW" />
  <category android:name="android.intent.category.DEFAULT" />
  <category android:name="android.intent.category.BROWSABLE" />
  <data
    android:scheme="https"
    android:host="notekeeper.com" />
</intent-filter>
```

```swift
// ios/Runner/AppDelegate.swift
import Flutter
import flutter_apphl

_ = FlutterAppDelegate()

// 配置 App Links
if #available(iOS 9.0, *) {
  // Links to be handled (must be configured in Apple Developer Portal)
}
```

## 5. 导航动画

```dart
// 自定义过渡动画
final _slideTransition = CustomTransitionPage(
  transitionDuration: const Duration(milliseconds: 300),
  transitionsBuilder: (context, animation, secondaryAnimation, child) {
    return SlideTransition(
      position: Tween<Offset>(
        begin: const Offset(1, 0),
        end: Offset.zero,
      ).animate(CurvedAnimation(
        parent: animation,
        curve: Curves.easeOutCubic,
      )),
      child: child,
    );
  },
);

GoRoute(
  path: '/note/:id',
  pageBuilder: (context, state) => _slideTransition(
    key: state.pageKey,
    child: NoteEditorScreen(noteId: state.pathParameters['id']!),
  ),
)

// 模态框动画
final _fadeScaleTransition = CustomTransitionPage(
  fullscreenDialog: true,
  transitionsBuilder: (context, animation, secondaryAnimation, child) {
    return FadeTransition(
      opacity: animation,
      child: ScaleTransition(
        scale: Tween<double>(begin: 0.9, end: 1.0).animate(
          CurvedAnimation(parent: animation, curve: Curves.easeOutCubic),
        ),
        child: child,
      ),
    );
  },
);

GoRoute(
  path: '/share/:token',
  pageBuilder: (context, state) => _fadeScaleTransition(
    key: state.pageKey,
    child: ShareViewScreen(token: state.pathParameters['token']!),
  ),
)
```

## 6. 导航流程图

```
┌─────────────────────────────────────────────────────────────────┐
│                         应用启动流程                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐                                                   │
│  │  Splash  │                                                   │
│  └────┬─────┘                                                   │
│       │                                                          │
│       ▼ 初始化 SDK                                               │
│  ┌──────────┐                                                   │
│  │ 加载状态  │──── 检查 Token ───> ┌──────────┐                  │
│  └────┬─────┘                    │  有效    │                  │
│       │ 无 Token                  └────┬─────┘                  │
│       ▼                                │                         │
│  ┌──────────┐                         │                         │
│  │  登录页   │                         ▼                         │
│  └────┬─────┘                    ┌──────────┐                  │
│       │ OAuth 登录                │   首页   │                  │
│       ▼                          └──────────┘                  │
│  ┌──────────┐                                                   │
│  │ OAuth    │                                                   │
│  │ Callback │                                                   │
│  └────┬─────┘                                                   │
│       │ Token 获取成功                                           │
│       ▼                                                         │
│  ┌──────────┐                                                   │
│  │   首页   │                                                   │
│  └──────────┘                                                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                         页面导航流程                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  首页 ──┬── 笔记列表 ──┬── 笔记编辑 ──┬── 版本历史               │
│         │              │              │                          │
│         │              │              └── 分享设置               │
│         │              │                                         │
│         │              └── 新建笔记                               │
│         │                                                    │
│         ├── 文件夹 ──┬── 文件夹详情 ──┬── 笔记列表               │
│         │            │                                          │
│         │            └── 新建文件夹                               │
│         │                                                    │
│         ├── 搜索 ───── 搜索结果 ─── 笔记详情                     │
│         │                                                    │
│         └── AI 助手 ──┬── AI 对话列表 ─── 对话详情              │
│                       │                                          │
│                       └── AI 聊天 ───── 流式响应                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```
