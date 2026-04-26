import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/network/api_client.dart';
import 'auth_provider.dart';

final searchProvider = StateNotifierProvider<SearchNotifier, AsyncValue<List<SearchResult>>>((ref) {
  return SearchNotifier(ref.watch(apiClientProvider));
});

class SearchResult {
  final String type;
  final String id;
  final String title;
  final String highlights;
  final double score;
  final DateTime updatedAt;

  SearchResult({
    required this.type,
    required this.id,
    required this.title,
    this.highlights = '',
    this.score = 0,
    required this.updatedAt,
  });

  factory SearchResult.fromJson(Map<String, dynamic> json) => SearchResult(
        type: json['type'] as String,
        id: json['id'] as String,
        title: json['title'] as String,
        highlights: json['highlights'] as String? ?? '',
        score: (json['score'] as num?)?.toDouble() ?? 0,
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );
}

class SearchNotifier extends StateNotifier<AsyncValue<List<SearchResult>>> {
  final ApiClient _api;

  SearchNotifier(this._api) : super(const AsyncValue.data([]));

  Future<void> search(String query, {int page = 1, int size = 20}) async {
    if (query.trim().isEmpty) {
      state = const AsyncValue.data([]);
      return;
    }
    state = const AsyncValue.loading();
    try {
      final response = await _api.get('/search', queryParameters: {
        'q': query,
        'page': page,
        'size': size,
      });
      final data = response.data!['data'] as Map<String, dynamic>;
      final items = (data['items'] as List)
          .map((e) => SearchResult.fromJson(e as Map<String, dynamic>))
          .toList();
      state = AsyncValue.data(items);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }
}
