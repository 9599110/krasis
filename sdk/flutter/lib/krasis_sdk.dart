library krasis_sdk;

export 'src/client.dart';
export 'src/error.dart';
export 'src/types.dart';
export 'src/auth.dart';
export 'src/notes.dart';
export 'src/search.dart';
export 'src/ai.dart';
export 'src/collab.dart';
export 'providers.dart';

import 'src/client.dart';
import 'src/auth.dart';
import 'src/notes.dart';
import 'src/search.dart';
import 'src/ai.dart';
import 'src/collab.dart';

class KrasisSDK {
  final KrasisClient client;

  late final AuthModule auth;
  late final UserModule users;
  late final NotesModule notes;
  late final FoldersModule folders;
  late final ShareModule share;
  late final SearchModule search;
  late final FileModule files;
  late final AIModule ai;
  CollabModule? _collab;

  KrasisSDK({
    required String baseUrl,
    String? token,
  }) : client = KrasisClient(apiBaseUrl: baseUrl, token: token) {
    auth = AuthModule(client);
    users = UserModule(client);
    notes = NotesModule(client);
    folders = FoldersModule(client);
    share = ShareModule(client);
    search = SearchModule(client);
    files = FileModule(client);
    ai = AIModule(client);
  }

  CollabModule get collab {
    _collab ??= CollabModule(
      wsBaseUrl: client.apiBaseUrl.replaceAll('http', 'ws'),
      token: client.token ?? '',
    );
    return _collab!;
  }

  Future<void> setToken(String token) => client.setToken(token);
  Future<void> clearToken() => client.clearToken();
  bool get isAuthenticated => client.isAuthenticated;

  void dispose() {
    _collab?.disconnect();
    client.dispose();
  }
}
