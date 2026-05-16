import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import '../../../core/network/api_client.dart';
import '../../../data/api/files_api.dart' show FolderFile;
import '../../providers/auth_provider.dart';

/// Provider for the photos list
final photosProvider = StateNotifierProvider<PhotosNotifier, AsyncValue<List<FolderFile>>>((ref) {
  return PhotosNotifier(ref.watch(apiClientProvider));
});

class PhotosNotifier extends StateNotifier<AsyncValue<List<FolderFile>>> {
  final ApiClient _api;
  bool _initialized = false;

  PhotosNotifier(this._api) : super(const AsyncValue.loading());

  Future<void> loadPhotos() async {
    _initialized = true;
    state = const AsyncValue.loading();
    try {
      final res = await _api.get('/files/images', queryParameters: {'page': 1, 'size': 100});
      final data = res.data?['data'];
      final items = (data?['items'] as List?)?.map((e) => FolderFile.fromJson(e as Map<String, dynamic>)).toList() ?? [];
      state = AsyncValue.data(items);
    } catch (e) {
      state = AsyncValue.error(e, StackTrace.current);
    }
  }

  Future<void> uploadPhoto(File file, {void Function(String err)? onError}) async {
    try {
      final fileName = file.path.split('/').last;

      // Step 1: Get presigned upload URL
      final presignRes = await _api.get('/files/presign', queryParameters: {
        'file_name': fileName,
        'file_type': 'image/jpeg',
      });
      final pd = presignRes.data?['data'] as Map<String, dynamic>;
      final fileId = pd['file_id'] as String;
      final uploadUrl = pd['upload_url'] as String;

      // Step 2: Upload file directly to MinIO
      final bytes = await file.readAsBytes();
      final uploadDio = Dio(BaseOptions(
        connectTimeout: const Duration(seconds: 60),
        receiveTimeout: const Duration(seconds: 60),
      ));
      final uploadRes = await uploadDio.put(
        uploadUrl,
        data: Stream.fromIterable([bytes]),
        options: Options(
          headers: {
            'Content-Type': 'image/jpeg',
            'Content-Length': bytes.length.toString(),
          },
        ),
      );
      if (uploadRes.statusCode != 200) {
        throw Exception('上传到存储失败 (${uploadRes.statusCode})');
      }

      // Step 3: Confirm upload
      await _api.post('/files/confirm', data: {'file_id': fileId});

      // Refresh list
      await loadPhotos();
    } catch (e) {
      onError?.call(e.toString());
    }
  }

  Future<void> deletePhoto(String fileId) async {
    try {
      await _api.delete('/files/$fileId');
      await loadPhotos();
    } catch (e) {
      rethrow;
    }
  }
}

class PhotosScreen extends ConsumerStatefulWidget {
  const PhotosScreen({super.key});

  @override
  ConsumerState<PhotosScreen> createState() => _PhotosScreenState();
}

class _PhotosScreenState extends ConsumerState<PhotosScreen> {
  final _picker = ImagePicker();
  bool _uploading = false;
  String? _previewUrl;
  FolderFile? _previewFile;
  bool _previewLoading = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(photosProvider.notifier).loadPhotos();
    });
  }

  Future<void> _pickAndUpload() async {
    final picked = await _picker.pickMultiImage(
      limit: 10,
      imageQuality: 90,
    );
    if (picked.isEmpty) return;

    for (final xFile in picked) {
      try {
        final file = File(xFile.path);
        await ref.read(photosProvider.notifier).uploadPhoto(file);
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('上传失败: $e')),
          );
        }
      }
    }
  }

  Future<void> _handleDelete(FolderFile photo) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除照片'),
        content: Text('确定要删除「${photo.fileName}」吗？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (confirm == true) {
      try {
        await ref.read(photosProvider.notifier).deletePhoto(photo.id);
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('已删除')),
          );
        }
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('删除失败: $e')),
          );
        }
      }
    }
  }

  Future<void> _handlePreview(FolderFile photo) async {
    setState(() {
      _previewFile = photo;
      _previewLoading = true;
    });
    try {
      final res = await ref.read(apiClientProvider).get('/files/${photo.id}/url');
      final url = res.data?['data']?['url'] as String?;
      setState(() {
        _previewUrl = url;
        _previewLoading = false;
      });
    } catch (e) {
      setState(() => _previewLoading = false);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('加载预览失败')),
        );
      }
    }
  }

  Future<void> _handleDownload(FolderFile photo) async {
    try {
      final res = await ref.read(apiClientProvider).get('/files/${photo.id}/download');
      final url = res.data?['data']?['url'] as String?;
      if (url != null) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('下载链接已获取')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('获取下载链接失败: $e')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(photosProvider);
    final theme = Theme.of(context);

    return Scaffold(
      body: Column(
        children: [
          // Header
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  '照片',
                  style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold),
                ),
                FilledButton.icon(
                  onPressed: _uploading ? null : _pickAndUpload,
                  icon: _uploading
                      ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                      : const Icon(Icons.add_a_photo, size: 18),
                  label: Text(_uploading ? '上传中...' : '上传照片'),
                ),
              ],
            ),
          ),

          // Content
          Expanded(
            child: state.when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (e, _) => Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.error_outline, size: 48, color: Colors.red),
                    const SizedBox(height: 8),
                    Text('加载失败: $e'),
                    const SizedBox(height: 16),
                    OutlinedButton(
                      onPressed: () => ref.read(photosProvider.notifier).loadPhotos(),
                      child: const Text('重试'),
                    ),
                  ],
                ),
              ),
              data: (photos) {
                if (photos.isEmpty) {
                  return Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.photo_library_outlined, size: 64, color: Colors.grey.shade400),
                        const SizedBox(height: 12),
                        Text('暂无照片', style: TextStyle(color: Colors.grey.shade500, fontSize: 16)),
                        const SizedBox(height: 8),
                        Text('点击上方「上传照片」按钮添加照片',
                            style: TextStyle(color: Colors.grey.shade400, fontSize: 13)),
                      ],
                    ),
                  );
                }

                return RefreshIndicator(
                  onRefresh: () => ref.read(photosProvider.notifier).loadPhotos(),
                  child: GridView.builder(
                    padding: const EdgeInsets.all(12),
                    gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                      crossAxisCount: 4,
                      crossAxisSpacing: 8,
                      mainAxisSpacing: 8,
                      childAspectRatio: 0.85,
                    ),
                    itemCount: photos.length,
                    itemBuilder: (context, index) {
                      final photo = photos[index];
                      return _PhotoCard(
                        photo: photo,
                        onTap: () => _handlePreview(photo),
                        onDownload: () => _handleDownload(photo),
                        onDelete: () => _handleDelete(photo),
                      );
                    },
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class _PhotoCard extends StatelessWidget {
  final FolderFile photo;
  final VoidCallback onTap;
  final VoidCallback onDownload;
  final VoidCallback onDelete;

  const _PhotoCard({
    required this.photo,
    required this.onTap,
    required this.onDownload,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      clipBehavior: Clip.antiAlias,
      elevation: 1,
      child: InkWell(
        onTap: onTap,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: photo.thumbnailUrl != null
                  ? Image.network(
                      photo.thumbnailUrl!,
                      fit: BoxFit.cover,
                      errorBuilder: (_, __, ___) => _placeholder(),
                    )
                  : _placeholder(),
            ),
            Padding(
              padding: const EdgeInsets.all(6),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    photo.fileName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 11),
                  ),
                  const SizedBox(height: 2),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(
                        photo.fileSize,
                        style: TextStyle(fontSize: 10, color: Colors.grey.shade500),
                      ),
                      Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          InkWell(
                            onTap: onDownload,
                            child: Icon(Icons.download, size: 14, color: Colors.grey.shade600),
                          ),
                          const SizedBox(width: 4),
                          InkWell(
                            onTap: onDelete,
                            child: Icon(Icons.delete_outline, size: 14, color: Colors.red.shade300),
                          ),
                        ],
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _placeholder() {
    return Container(
      color: Colors.grey.shade100,
      child: Center(
        child: Icon(Icons.image, size: 32, color: Colors.grey.shade400),
      ),
    );
  }
}
