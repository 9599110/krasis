# Flutter 前端详细设计

## 1. 项目结构

```
frontend/
├── lib/
│   ├── main.dart
│   ├── app.dart
│   ├── config/
│   │   ├── app_config.dart          # 应用配置
│   │   ├── theme.dart               # 主题配置
│   │   └── routes.dart              # 路由配置
│   ├── core/
│   │   ├── constants/
│   │   │   ├── api_constants.dart   # API 常量
│   │   │   └── app_constants.dart   # 应用常量
│   │   ├── errors/
│   │   │   ├── exceptions.dart       # 异常定义
│   │   │   └── failures.dart         # 失败类型
│   │   ├── network/
│   │   │   ├── api_client.dart       # HTTP 客户端
│   │   │   ├── api_interceptor.dart # 拦截器
│   │   │   └── ws_client.dart       # WebSocket 客户端
│   │   ├── storage/
│   │   │   ├── secure_storage.dart   # 安全存储
│   │   │   └── local_storage.dart    # 本地存储
│   │   └── utils/
│   │       ├── extensions.dart      # 扩展方法
│   │       └── validators.dart       # 验证器
│   ├── data/
│   │   ├── models/                  # 数据模型
│   │   │   ├── user_model.dart
│   │   │   ├── note_model.dart
│   │   │   ├── folder_model.dart
│   │   │   ├── file_model.dart
│   │   │   ├── ai_message_model.dart
│   │   │   └── session_model.dart
│   │   ├── repositories/            # 仓储实现
│   │   │   ├── auth_repository_impl.dart
│   │   │   ├── note_repository_impl.dart
│   │   │   ├── user_repository_impl.dart
│   │   │   ├── file_repository_impl.dart
│   │   │   └── ai_repository_impl.dart
│   │   └── datasources/
│   │       ├── remote/
│   │       │   ├── auth_api.dart
│   │       │   ├── note_api.dart
│   │       │   ├── user_api.dart
│   │       │   ├── file_api.dart
│   │       │   └── ai_api.dart
│   │       └── local/
│   │           └── cache_datasource.dart
│   ├── domain/
│   │   ├── entities/                # 领域实体
│   │   ├── repositories/            # 仓储接口
│   │   └── usecases/                # 用例
│   │       ├── auth/
│   │       ├── note/
│   │       ├── user/
│   │       ├── file/
│   │       └── ai/
│   ├── presentation/
│   │   ├── widgets/                 # 通用组件
│   │   │   ├── buttons/
│   │   │   ├── inputs/
│   │   │   ├── cards/
│   │   │   ├── dialogs/
│   │   │   └── loading/
│   │   ├── screens/                 # 页面
│   │   │   ├── auth/
│   │   │   │   ├── login_screen.dart
│   │   │   │   ├── oauth_callback_screen.dart
│   │   │   │   └── splash_screen.dart
│   │   │   ├── home/
│   │   │   │   ├── home_screen.dart
│   │   │   │   └── widgets/
│   │   │   ├── note/
│   │   │   │   ├── note_list_screen.dart
│   │   │   │   ├── note_editor_screen.dart
│   │   │   │   ├── note_detail_screen.dart
│   │   │   │   └── widgets/
│   │   │   ├── folder/
│   │   │   │   └── folder_screen.dart
│   │   │   ├── share/
│   │   │   │   └── share_view_screen.dart
│   │   │   ├── ai/
│   │   │   │   ├── ai_chat_screen.dart
│   │   │   │   └── widgets/
│   │   │   ├── profile/
│   │   │   │   ├── profile_screen.dart
│   │   │   │   ├── devices_screen.dart
│   │   │   │   └── settings_screen.dart
│   │   │   └── admin/
│   │   │       ├── admin_users_screen.dart
│   │   │       └── admin_dashboard_screen.dart
│   │   └── providers/               # Riverpod Providers
│   │       ├── auth_provider.dart
│   │       ├── note_provider.dart
│   │       ├── user_provider.dart
│   │       ├── file_provider.dart
│   │       ├── ai_provider.dart
│   │       └── settings_provider.dart
│   └── generated/                   # 代码生成
│       └── api/
├── assets/
│   ├── images/
│   ├── icons/
│   └── fonts/
├── ios/
├── android/
├── web/
├── linux/
├── macos/
├── windows/
├── pubspec.yaml
└── README.md
```

## 2. 技术选型

| 类别 | 技术 | 版本 |
|------|------|------|
| 框架 | Flutter | 3.16+ |
| 状态管理 | Riverpod | 2.x |
| 网络请求 | Dio | 5.x |
| WebSocket | web_socket_channel | 2.x |
| 路由 | go_router | 14.x |
| 本地存储 | shared_preferences, flutter_secure_storage | Latest |
| 富文本编辑器 | flutter_quill | 10.x |
| 图片处理 | cached_network_image | 3.x |
| 安全存储 | flutter_secure_storage | 9.x |
| 依赖注入 | riverpod_annotation | 2.x |
| 代码生成 | freezed, json_serializable | Latest |

## 3. 核心模块设计

### 3.1 网络层

```dart
// lib/core/network/api_client.dart

class ApiClient {
  final Dio _dio;
  final SecureStorage _secureStorage;
  
  ApiClient({
    required Dio dio,
    required SecureStorage secureStorage,
  }) : _dio = dio,
       _secureStorage = secureStorage {
    _dio.interceptors.addAll([
      AuthInterceptor(secureStorage),
      LoggingInterceptor(),
      RetryInterceptor(dio),
    ]);
  }
  
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) => _dio.get<T>(path, queryParameters: queryParameters);
  
  Future<Response<T>> post<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) => _dio.post<T>(path, data: data, queryParameters: queryParameters);
  
  Future<Response<T>> put<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) => _dio.put<T>(path, data: data, queryParameters: queryParameters);
  
  Future<Response<T>> delete<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) => _dio.delete<T>(path, queryParameters: queryParameters);
}

// lib/core/network/api_interceptor.dart

class AuthInterceptor extends Interceptor {
  final SecureStorage _secureStorage;
  
  AuthInterceptor(this._secureStorage);
  
  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = await _secureStorage.getAccessToken();
    
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    
    handler.next(options);
  }
  
  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    if (err.response?.statusCode == 401) {
      // Token 过期，触发刷新或重新登录
      EventBus.emit(AuthEvents.tokenExpired);
    }
    handler.next(err);
  }
}

class RetryInterceptor extends Interceptor {
  final Dio _dio;
  static const _maxRetries = 3;
  
  RetryInterceptor(this._dio);
  
  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final extra = err.requestOptions.extra;
    final retryCount = extra['retryCount'] ?? 0;
    
    if (_shouldRetry(err) && retryCount < _maxRetries) {
      extra['retryCount'] = retryCount + 1;
      err.requestOptions.extra = extra;
      
      await Future.delayed(Duration(seconds: retryCount + 1));
      try {
        final response = await _dio.fetch(err.requestOptions);
        handler.resolve(response);
        return;
      } catch (e) {
        // 继续处理错误
      }
    }
    
    handler.next(err);
  }
  
  bool _shouldRetry(DioException err) {
    return err.type == DioExceptionType.connectionTimeout ||
           err.type == DioExceptionType.receiveTimeout ||
           (err.response?.statusCode != null &&
            err.response!.statusCode! >= 500);
  }
}
```

### 3.2 WebSocket 客户端

```dart
// lib/core/network/ws_client.dart

class WsClient {
  WebSocketChannel? _channel;
  final _messageController = StreamController<WsMessage>.broadcast();
  final _connectionController = StreamController<ConnectionState>.broadcast();
  
  String? _currentNoteId;
  String? _token;
  
  Stream<WsMessage> get messages => _messageController.stream;
  Stream<ConnectionState> get connectionState => _connectionController.stream;
  
  Future<void> connect({
    required String noteId,
    required String token,
  }) async {
    _currentNoteId = noteId;
    _token = token;
    
    final uri = Uri.parse(
      '${AppConfig.wsBaseUrl}/ws/collab?note_id=$noteId&token=$token',
    );
    
    _channel = WebSocketChannel.connect(uri);
    
    _channel!.stream.listen(
      _onMessage,
      onError: _onError,
      onDone: _onDone,
    );
    
    _connectionController.add(ConnectionState.connected);
  }
  
  void _onMessage(dynamic data) {
    try {
      final json = jsonDecode(data as String);
      final message = WsMessage.fromJson(json);
      _messageController.add(message);
    } catch (e) {
      Logger.e('WebSocket message parse error: $e');
    }
  }
  
  void send(WsMessage message) {
    _channel?.sink.add(jsonEncode(message.toJson()));
  }
  
  void sendSync(List<int> update) {
    send(WsMessage(
      type: 'sync',
      payload: {
        'update': base64Encode(update),
      },
    ));
  }
  
  void sendAwareness(AwarenessState state) {
    send(WsMessage(
      type: 'awareness',
      payload: state.toJson(),
    ));
  }
  
  void disconnect() {
    _channel?.sink.close();
    _channel = null;
    _currentNoteId = null;
    _connectionController.add(ConnectionState.disconnected);
  }
}
```

### 3.3 状态管理 (Riverpod)

```dart
// lib/presentation/providers/auth_provider.dart

@riverpod
class AuthNotifier extends _$AuthNotifier {
  @override
  FutureOr<AuthState> build() async {
    final secureStorage = ref.read(secureStorageProvider);
    final token = await secureStorage.getAccessToken();
    
    if (token == null) {
      return const AuthState.unauthenticated();
    }
    
    try {
      final apiClient = ref.read(apiClientProvider);
      final response = await apiClient.get('/auth/me');
      final user = UserModel.fromJson(response.data['data']);
      return AuthState.authenticated(user);
    } catch (e) {
      await secureStorage.clearAll();
      return const AuthState.unauthenticated();
    }
  }
  
  Future<void> loginWithOAuth(OAuthProvider provider) async {
    state = const AuthState.loading();
    
    final authUrl = _getOAuthUrl(provider);
    final result = await WebAuthProvider.authenticate(
      url: authUrl,
      callbackUrlScheme: 'notekeeper',
    );
    
    if (result != null) {
      final code = Uri.parse(result).queryParameters['code'];
      if (code != null) {
        await _handleOAuthCallback(provider, code);
      }
    }
  }
  
  Future<void> _handleOAuthCallback(OAuthProvider provider, String code) async {
    try {
      final apiClient = ref.read(apiClientProvider);
      final response = await apiClient.post(
        '/auth/${provider.name}/callback',
        data: {'code': code},
      );
      
      final token = response.data['data']['access_token'];
      final user = UserModel.fromJson(response.data['data']['user']);
      
      await ref.read(secureStorageProvider).saveAccessToken(token);
      state = AuthState.authenticated(user);
    } catch (e) {
      state = AuthState.error(e.toString());
    }
  }
  
  Future<void> logout() async {
    try {
      final apiClient = ref.read(apiClientProvider);
      await apiClient.post('/auth/logout');
    } finally {
      await ref.read(secureStorageProvider).clearAll();
      state = const AuthState.unauthenticated();
    }
  }
}

enum OAuthProvider { github, google }

sealed class AuthState {
  const AuthState();
  const factory AuthState.authenticated(UserModel user) = AuthState.authenticated;
  const factory AuthState.unauthenticated() = AuthState.unauthenticated;
  const factory AuthState.loading() = AuthState.loading;
  const factory AuthState.error(String message) = AuthState.error;
}
```

### 3.4 笔记模块

```dart
// lib/presentation/providers/note_provider.dart

@riverpod
class NoteListNotifier extends _$NoteListNotifier {
  @override
  Future<NoteListState> build() async {
    return const NoteListState();
  }
  
  Future<void> loadNotes({
    int page = 1,
    String? folderId,
    String? keyword,
  }) async {
    final currentState = state.valueOrNull ?? const NoteListState();
    state = const AsyncLoading();
    
    try {
      final apiClient = ref.read(apiClientProvider);
      final response = await apiClient.get('/notes', queryParameters: {
        'page': page,
        'size': 20,
        if (folderId != null) 'folder_id': folderId,
        if (keyword != null) 'keyword': keyword,
      });
      
      final notes = (response.data['data']['items'] as List)
          .map((e) => NoteModel.fromJson(e))
          .toList();
      
      final total = response.data['data']['total'] as int;
      
      if (page == 1) {
        state = AsyncData(NoteListState(
          notes: notes,
          page: page,
          hasMore: notes.length == 20,
        ));
      } else {
        state = AsyncData(currentState.copyWith(
          notes: [...currentState.notes, ...notes],
          page: page,
          hasMore: notes.length == 20,
        ));
      }
    } catch (e, st) {
      state = AsyncError(e, st);
    }
  }
  
  Future<void> refresh() async {
    await loadNotes(page: 1);
  }
}

@riverpod
class NoteEditorNotifier extends _$NoteEditorNotifier {
  @override
  Future<NoteModel?> build(String noteId) async {
    if (noteId == 'new') return null;
    
    final apiClient = ref.read(apiClientProvider);
    final response = await apiClient.get('/notes/$noteId');
    return NoteModel.fromJson(response.data['data']);
  }
  
  Future<NoteModel> createNote({
    required String title,
    required String content,
    String? folderId,
  }) async {
    final apiClient = ref.read(apiClientProvider);
    final response = await apiClient.post('/notes', data: {
      'title': title,
      'content': content,
      if (folderId != null) 'folder_id': folderId,
    });
    
    final note = NoteModel.fromJson(response.data['data']);
    
    // 刷新列表
    ref.invalidate(noteListNotifierProvider);
    
    return note;
  }
  
  Future<void> updateNote({
    required String noteId,
    required String title,
    required String content,
    required int version,
  }) async {
    try {
      final apiClient = ref.read(apiClientProvider);
      await apiClient.put('/notes/$noteId', data: {
        'title': title,
        'content': content,
        'version': version,
      }, options: Options(headers: {
        'If-Match': version.toString(),
      }));
    } on DioException catch (e) {
      if (e.response?.statusCode == 409) {
        // 版本冲突
        final currentNote = NoteModel.fromJson(
          e.response!.data['data']['note'],
        );
        throw VersionConflictException(
          currentVersion: currentNote.version,
          note: currentNote,
        );
      }
      rethrow;
    }
  }
  
  Future<void> deleteNote(String noteId) async {
    final apiClient = ref.read(apiClientProvider);
    await apiClient.delete('/notes/$noteId');
    ref.invalidate(noteListNotifierProvider);
  }
}
```

### 3.5 AI 对话模块

```dart
// lib/presentation/providers/ai_provider.dart

@riverpod
class AIChatNotifier extends _$AIChatNotifier {
  WsClient? _wsClient;
  
  @override
  Future<AIChatState> build(String conversationId) async {
    if (conversationId.isEmpty) {
      return const AIChatState(messages: []);
    }
    
    final apiClient = ref.read(apiClientProvider);
    final response = await apiClient.get(
      '/ai/conversations/$conversationId/messages',
    );
    
    final messages = (response.data['data'] as List)
        .map((e) => AIMessageModel.fromJson(e))
        .toList();
    
    return AIChatState(messages: messages);
  }
  
  Stream<String> ask(String question) async* {
    final controller = StreamController<String>();
    
    try {
      final secureStorage = ref.read(secureStorageProvider);
      final token = await secureStorage.getAccessToken();
      
      _wsClient = WsClient();
      
      _wsClient!.messages.listen((message) {
        if (message.type == 'token') {
          final token = message.payload['token'] as String;
          controller.add(token);
        } else if (message.type == 'done') {
          controller.close();
        } else if (message.type == 'reference') {
          // 处理引用
        }
      });
      
      _wsClient!.connect(
        noteId: 'ai_session',
        token: token!,
      );
      
      // 发送问题
      final apiClient = ref.read(apiClientProvider);
      await apiClient.post('/ai/ask', data: {
        'question': question,
        'conversation_id': conversationId.isEmpty ? null : conversationId,
        'stream': true,
      });
      
      yield* controller.stream;
    } finally {
      _wsClient?.disconnect();
      _wsClient = null;
    }
  }
  
  void cancelAsk() {
    _wsClient?.disconnect();
    _wsClient = null;
  }
}
```

## 4. 页面设计

### 4.1 登录页面

```dart
// lib/presentation/screens/auth/login_screen.dart

class LoginScreen extends ConsumerWidget {
  const LoginScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Logo
              Image.asset(
                'assets/images/logo.png',
                height: 80,
              ),
              const SizedBox(height: 16),
              
              // 标题
              Text(
                'NoteKeeper',
                style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              
              Text(
                '智能笔记，让知识触手可及',
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: Colors.grey,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 48),
              
              // GitHub 登录
              _OAuthButton(
                provider: 'GitHub',
                icon: Icons.code,
                color: Colors.black87,
                onPressed: () => ref.read(authNotifierProvider.notifier)
                    .loginWithOAuth(OAuthProvider.github),
              ),
              const SizedBox(height: 16),
              
              // Google 登录
              _OAuthButton(
                provider: 'Google',
                icon: Icons.g_mobiledata,
                color: Colors.blue,
                onPressed: () => ref.read(authNotifierProvider.notifier)
                    .loginWithOAuth(OAuthProvider.google),
              ),
              const SizedBox(height: 32),
              
              // 分割线
              const Row(
                children: [
                  Expanded(child: Divider()),
                  Padding(
                    padding: EdgeInsets.symmetric(horizontal: 16),
                    child: Text('或'),
                  ),
                  Expanded(child: Divider()),
                ],
              ),
              const SizedBox(height: 16),
              
              // 邮箱登录
              OutlinedButton.icon(
                onPressed: () => context.push('/login/email'),
                icon: const Icon(Icons.email_outlined),
                label: const Text('邮箱登录'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
```

### 4.2 笔记列表页面

```dart
// lib/presentation/screens/note/note_list_screen.dart

class NoteListScreen extends ConsumerStatefulWidget {
  const NoteListScreen({super.key});

  @override
  ConsumerState<NoteListScreen> createState() => _NoteListScreenState();
}

class _NoteListScreenState extends ConsumerState<NoteListScreen> {
  final _searchController = TextEditingController();
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    Future.microtask(() {
      ref.read(noteListNotifierProvider.notifier).loadNotes();
    });
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 200) {
      final state = ref.read(noteListNotifierProvider).valueOrNull;
      if (state != null && state.hasMore) {
        ref.read(noteListNotifierProvider.notifier).loadNotes(
          page: state.page + 1,
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final noteListState = ref.watch(noteListNotifierProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('笔记'),
        actions: [
          IconButton(
            icon: const Icon(Icons.search),
            onPressed: () => _showSearch(context),
          ),
          IconButton(
            icon: const Icon(Icons.grid_view),
            onPressed: () => ref.read(noteListViewModeProvider.notifier).toggle(),
          ),
        ],
      ),
      body: noteListState.when(
        data: (state) => _buildNoteList(state),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _buildError(e),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => context.push('/note/new'),
        child: const Icon(Icons.add),
      ),
    );
  }

  Widget _buildNoteList(NoteListState state) {
    if (state.notes.isEmpty) {
      return _buildEmptyState();
    }

    final viewMode = ref.watch(noteListViewModeProvider);

    return RefreshIndicator(
      onRefresh: () => ref.read(noteListNotifierProvider.notifier).refresh(),
      child: viewMode == ViewMode.grid
          ? _buildGridView(state.notes)
          : _buildListView(state.notes),
    );
  }

  Widget _buildListView(List<NoteModel> notes) {
    return ListView.builder(
      controller: _scrollController,
      itemCount: notes.length,
      itemBuilder: (context, index) {
        final note = notes[index];
        return NoteCard(
          note: note,
          onTap: () => context.push('/note/${note.id}'),
          onDelete: () => _deleteNote(note),
          onShare: () => _shareNote(note),
        );
      },
    );
  }

  Widget _buildGridView(List<NoteModel> notes) {
    return GridView.builder(
      controller: _scrollController,
      padding: const EdgeInsets.all(16),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        childAspectRatio: 1.2,
        crossAxisSpacing: 12,
        mainAxisSpacing: 12,
      ),
      itemCount: notes.length,
      itemBuilder: (context, index) {
        final note = notes[index];
        return NoteGridCard(
          note: note,
          onTap: () => context.push('/note/${note.id}'),
        );
      },
    );
  }
}
```

### 4.3 笔记编辑器

```dart
// lib/presentation/screens/note/note_editor_screen.dart

class NoteEditorScreen extends ConsumerStatefulWidget {
  final String? noteId;

  const NoteEditorScreen({super.key, this.noteId});

  @override
  ConsumerState<NoteEditorScreen> createState() => _NoteEditorScreenState();
}

class _NoteEditorScreenState extends ConsumerState<NoteEditorScreen> {
  late QuillController _quillController;
  late TextEditingController _titleController;
  final FocusNode _editorFocusNode = FocusNode();
  bool _isLoading = false;
  bool _hasChanges = false;
  int _currentVersion = 0;

  @override
  void initState() {
    super.initState();
    _titleController = TextEditingController();
    _quillController = QuillController.basic();
    
    _setupListeners();
    
    if (widget.noteId != null && widget.noteId != 'new') {
      _loadNote();
    }
  }

  void _setupListeners() {
    _titleController.addListener(_markAsChanged);
    _quillController.addListener(_markAsChanged);
  }

  void _markAsChanged() {
    if (!_hasChanges) {
      setState(() => _hasChanges = true);
    }
  }

  Future<void> _loadNote() async {
    setState(() => _isLoading = true);
    
    try {
      final note = await ref.read(noteEditorNotifierProvider(widget.noteId!).future);
      if (note != null && mounted) {
        _titleController.text = note.title;
        _currentVersion = note.version;
        
        if (note.content.isNotEmpty) {
          _quillController = QuillController(
            document: Document.fromJson(jsonDecode(note.content)),
            selection: const TextSelection.collapsed(offset: 0),
          );
        }
        setState(() {});
      }
    } finally {
      setState(() => _isLoading = false);
    }
  }

  Future<void> _saveNote() async {
    final title = _titleController.text.trim();
    if (title.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('标题不能为空')),
      );
      return;
    }

    setState(() => _isLoading = true);

    try {
      final content = jsonEncode(_quillController.document.toDelta().toJson());
      
      if (widget.noteId == 'new') {
        final note = await ref.read(noteEditorNotifierProvider('new').notifier)
            .createNote(title: title, content: content);
        
        if (mounted) {
          context.replace('/note/${note.id}');
        }
      } else {
        await ref.read(noteEditorNotifierProvider(widget.noteId!).notifier)
            .updateNote(
              noteId: widget.noteId!,
              title: title,
              content: content,
              version: _currentVersion,
            );
      }

      setState(() => _hasChanges = false);
      
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('保存成功')),
        );
      }
    } on VersionConflictException catch (e) {
      _showConflictDialog(e);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('保存失败: $e')),
        );
      }
    } finally {
      setState(() => _isLoading = false);
    }
  }

  void _showConflictDialog(VersionConflictException e) {
    showDialog(
      context: context,
      builder: (context) => ConflictDialog(
        currentVersion: e.currentVersion,
        note: e.note,
        onKeepLocal: () async {
          Navigator.pop(context);
          await ref.read(noteEditorNotifierProvider(widget.noteId!).notifier)
              .updateNote(
                noteId: widget.noteId!,
                title: _titleController.text,
                content: jsonEncode(_quillController.document.toDelta().toJson()),
                version: e.currentVersion,
              );
        },
        onKeepServer: () {
          Navigator.pop(context);
          _loadNote();
        },
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: !_hasChanges,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        
        final shouldPop = await showDialog<bool>(
          context: context,
          builder: (context) => AlertDialog(
            title: const Text('有未保存的更改'),
            content: const Text('是否保存更改？'),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('放弃'),
              ),
              TextButton(
                onPressed: () {
                  Navigator.pop(context, false);
                  _saveNote();
                },
                child: const Text('保存'),
              ),
            ],
          ),
        );
        
        if (shouldPop == true && context.mounted) {
          Navigator.pop(context);
        }
      },
      child: Scaffold(
        appBar: AppBar(
          title: _buildTitleField(),
          actions: [
            if (_hasChanges)
              IconButton(
                icon: const Icon(Icons.save),
                onPressed: _saveNote,
              ),
            IconButton(
              icon: const Icon(Icons.more_vert),
              onPressed: _showMoreMenu,
            ),
          ],
        ),
        body: _isLoading
            ? const Center(child: CircularProgressIndicator())
            : Column(
                children: [
                  _buildToolbar(),
                  Expanded(child: _buildEditor()),
                ],
              ),
      ),
    );
  }

  Widget _buildTitleField() {
    return TextField(
      controller: _titleController,
      style: Theme.of(context).textTheme.titleLarge,
      decoration: const InputDecoration(
        hintText: '笔记标题',
        border: InputBorder.none,
      ),
    );
  }

  Widget _buildToolbar() {
    return Container(
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(color: Colors.grey.shade300),
        ),
      ),
      child: QuillSimpleToolbar(
        controller: _quillController,
        config: QuillSimpleToolbarConfig(
          showAlignmentButtons: true,
          showBackgroundColorButton: false,
          showClearFormat: false,
          showColorButton: false,
          showFontFamily: false,
          showFontSize: false,
          showInlineCode: false,
          showSearchButton: false,
          showSubscript: false,
          showSuperscript: false,
          showStrikeThrough: false,
          multiRowsDisplay: false,
        ),
      ),
    );
  }

  Widget _buildEditor() {
    return QuillEditor.basic(
      controller: _quillController,
      focusNode: _editorFocusNode,
      config: QuillEditorConfig(
        placeholder: '开始写作...',
        padding: const EdgeInsets.all(16),
        autoFocus: widget.noteId == 'new',
      ),
    );
  }
}
```

### 4.4 AI 聊天页面

```dart
// lib/presentation/screens/ai/ai_chat_screen.dart

class AIChatScreen extends ConsumerStatefulWidget {
  const AIChatScreen({super.key});

  @override
  ConsumerState<AIChatScreen> createState() => _AIChatScreenState();
}

class _AIChatScreenState extends ConsumerState<AIChatScreen> {
  final _inputController = TextEditingController();
  final _scrollController = ScrollController();
  bool _isAsking = false;
  String _currentResponse = '';

  @override
  void dispose() {
    _inputController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _sendMessage() async {
    final question = _inputController.text.trim();
    if (question.isEmpty) return;

    _inputController.clear();
    setState(() => _isAsking = true);

    // 添加用户消息
    final userMessage = AIMessageModel(
      id: uuid.v4(),
      role: 'user',
      content: question,
      createdAt: DateTime.now(),
    );

    setState(() => _currentResponse = '');

    try {
      final stream = ref.read(aiChatNotifierProvider('').notifier).ask(question);

      await for (final token in stream) {
        setState(() => _currentResponse += token);
      }

      // 保存完整对话
      // ...
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('提问失败: $e')),
        );
      }
    } finally {
      setState(() => _isAsking = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final chatState = ref.watch(aiChatNotifierProvider(''));

    return Scaffold(
      appBar: AppBar(
        title: const Text('AI 助手'),
        actions: [
          IconButton(
            icon: const Icon(Icons.history),
            onPressed: () => context.push('/ai/history'),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: chatState.when(
              data: (state) => _buildMessageList(state.messages),
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (e, _) => Center(child: Text('错误: $e')),
            ),
          ),
          if (_isAsking) _buildStreamingResponse(),
          _buildInputBar(),
        ],
      ),
    );
  }

  Widget _buildStreamingResponse() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.grey.shade100,
        border: Border(
          top: BorderSide(color: Colors.grey.shade300),
        ),
      ),
      child: Row(
        children: [
          const SizedBox(
            width: 20,
            height: 20,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              _currentResponse.isEmpty ? 'AI 正在思考...' : _currentResponse,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildInputBar() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.05),
            blurRadius: 10,
            offset: const Offset(0, -2),
          ),
        ],
      ),
      child: SafeArea(
        child: Row(
          children: [
            Expanded(
              child: TextField(
                controller: _inputController,
                decoration: InputDecoration(
                  hintText: '向 AI 提问关于你的笔记...',
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(24),
                  ),
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 20,
                    vertical: 12,
                  ),
                ),
                maxLines: 4,
                minLines: 1,
                onSubmitted: (_) => _sendMessage(),
              ),
            ),
            const SizedBox(width: 12),
            IconButton.filled(
              onPressed: _isAsking ? null : _sendMessage,
              icon: const Icon(Icons.send),
            ),
          ],
        ),
      ),
    );
  }
}
```

## 5. 依赖注入配置

```dart
// lib/providers.dart

@Riverpod(keepAlive: true)
Dio dio(DioRef ref) {
  return Dio(BaseOptions(
    baseUrl: AppConfig.apiBaseUrl,
    connectTimeout: const Duration(seconds: 30),
    receiveTimeout: const Duration(seconds: 30),
  ));
}

@Riverpod(keepAlive: true)
SecureStorage secureStorage(SecureStorageRef ref) {
  return SecureStorage();
}

@Riverpod(keepAlive: true)
ApiClient apiClient(ApiClientRef ref) {
  return ApiClient(
    dio: ref.watch(dioProvider),
    secureStorage: ref.watch(secureStorageProvider),
  );
}

@Riverpod(keepAlive: true)
WsClient wsClient(WsClientRef ref) {
  return WsClient();
}
```

## 6. 路由配置

```dart
// lib/config/routes.dart

@Riverpod
GoRouter router(RouterRef ref) {
  final authState = ref.watch(authNotifierProvider);
  
  return GoRouter(
    initialLocation: '/',
    redirect: (context, state) {
      final isLoggedIn = authState.valueOrNull?.maybeMap(
        authenticated: (_) => true,
        orElse: () => false,
      ) ?? false;
      
      final isAuthRoute = state.matchedLocation.startsWith('/login') ||
                          state.matchedLocation == '/';
      
      if (!isLoggedIn && !isAuthRoute) {
        return '/login';
      }
      
      if (isLoggedIn && isAuthRoute) {
        return '/notes';
      }
      
      return null;
    },
    routes: [
      GoRoute(
        path: '/',
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/login/oauth/callback',
        builder: (context, state) => OAuthCallbackScreen(
          provider: state.uri.queryParameters['provider'] ?? 'github',
          code: state.uri.queryParameters['code'] ?? '',
        ),
      ),
      ShellRoute(
        builder: (context, state, child) => MainShell(child: child),
        routes: [
          GoRoute(
            path: '/notes',
            builder: (context, state) => const NoteListScreen(),
            routes: [
              GoRoute(
                path: ':noteId',
                builder: (context, state) => NoteEditorScreen(
                  noteId: state.pathParameters['noteId'],
                ),
              ),
            ],
          ),
          GoRoute(
            path: '/folders',
            builder: (context, state) => const FolderScreen(),
          ),
          GoRoute(
            path: '/ai',
            builder: (context, state) => const AIChatScreen(),
          ),
          GoRoute(
            path: '/profile',
            builder: (context, state) => const ProfileScreen(),
            routes: [
              GoRoute(
                path: 'devices',
                builder: (context, state) => const DevicesScreen(),
              ),
              GoRoute(
                path: 'settings',
                builder: (context, state) => const SettingsScreen(),
              ),
            ],
          ),
        ],
      ),
      GoRoute(
        path: '/share/:token',
        builder: (context, state) => ShareViewScreen(
          token: state.pathParameters['token']!,
        ),
      ),
    ],
  );
}
```
