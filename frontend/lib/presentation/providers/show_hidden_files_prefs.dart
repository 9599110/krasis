/// Preference provider controlling whether hidden encrypted files are
/// visible in file listings or hidden from the frontend.
///
/// When `true`, encrypted files marked as hidden on the server will
/// also be fetched and shown in folder file listings.
/// When `false` (default), hidden files are not synced to the client.
library show_hidden_files_prefs;

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _prefKey = 'show_hidden_files';

/// Riverpod provider for the "show hidden files" toggle state.
final showHiddenFilesProvider =
    StateNotifierProvider<ShowHiddenFilesNotifier, bool>((ref) {
  return ShowHiddenFilesNotifier();
});

class ShowHiddenFilesNotifier extends StateNotifier<bool> {
  ShowHiddenFilesNotifier() : super(false) {
    Future.microtask(() => _load());
  }

  Future<void> _load() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      state = prefs.getBool(_prefKey) ?? false;
    } catch (_) {
      state = false;
    }
  }

  Future<void> toggle(bool value) async {
    state = value;
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setBool(_prefKey, value);
    } catch (_) {
      // persist best-effort
    }
  }
}
