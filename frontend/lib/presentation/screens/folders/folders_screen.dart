import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../data/models/note_model.dart';
import '../../providers/auth_provider.dart';
import '../../providers/note_provider.dart';

class FoldersScreen extends ConsumerStatefulWidget {
  const FoldersScreen({super.key});

  @override
  ConsumerState<FoldersScreen> createState() => _FoldersScreenState();
}

class _FoldersScreenState extends ConsumerState<FoldersScreen> {
  Future<void> _showCreateFolder([FolderModel? existing]) async {
    final nameController = TextEditingController(text: existing?.name ?? '');
    final color = existing?.color ?? '#4CAF50';

    final result = await showDialog<Map<String, String?>>(
      context: context,
      builder: (ctx) => _FolderDialog(
        controller: nameController,
        initialColor: color,
        isEdit: existing != null,
      ),
    );

    if (result == null) return;

    final api = ref.read(apiClientProvider);
    try {
      if (existing != null) {
        await api.put('/folders/${existing.id}', data: {
          'name': result['name'],
          'color': result['color'],
          'parent_id': existing.parentId,
        });
      } else {
        await api.post('/folders', data: {
          'name': result['name'],
          'color': result['color'],
        });
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(existing != null ? '文件夹已更新' : '文件夹已创建')),
        );
      }
      ref.read(folderListProvider.notifier).loadFolders();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('操作失败: $e')),
        );
      }
    }
  }

  Future<void> _deleteFolder(FolderModel folder) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除文件夹'),
        content: Text('确定要删除文件夹"${folder.name}"吗？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('删除')),
        ],
      ),
    );

    if (confirmed != true) return;

    try {
      final api = ref.read(apiClientProvider);
      await api.delete('/folders/${folder.id}');
      ref.read(folderListProvider.notifier).loadFolders();
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('文件夹已删除')),
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

  @override
  Widget build(BuildContext context) {
    final folders = ref.watch(folderListProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: const Text('文件夹管理'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            onPressed: () => _showCreateFolder(),
            tooltip: '新建文件夹',
          ),
        ],
      ),
      body: folders.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text('加载失败: $e')),
        data: (items) {
          if (items.isEmpty) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.folder_outlined, size: 64, color: Colors.grey.shade300),
                  const SizedBox(height: 16),
                  Text('暂无文件夹', style: TextStyle(color: Colors.grey.shade500)),
                  const SizedBox(height: 16),
                  FilledButton.icon(
                    onPressed: () => _showCreateFolder(),
                    icon: const Icon(Icons.add),
                    label: const Text('新建文件夹'),
                  ),
                ],
              ),
            );
          }

          return ListView.separated(
            padding: const EdgeInsets.all(16),
            itemCount: items.length,
            separatorBuilder: (_, __) => const SizedBox(height: 8),
            itemBuilder: (context, index) {
              final folder = items[index];
              final color = _parseColor(folder.color);

              return Card(
                child: ListTile(
                  leading: Icon(Icons.folder, color: color),
                  title: Text(folder.name),
                  subtitle: Text(
                    '创建时间: ${_formatDate(folder.createdAt)}',
                    style: TextStyle(color: Colors.grey.shade600, fontSize: 12),
                  ),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      IconButton(
                        icon: const Icon(Icons.edit_outlined),
                        onPressed: () => _showCreateFolder(folder),
                      ),
                      IconButton(
                        icon: const Icon(Icons.delete_outline),
                        color: Colors.red,
                        onPressed: () => _deleteFolder(folder),
                      ),
                    ],
                  ),
                ),
              );
            },
          );
        },
      ),
    );
  }

  String _formatDate(DateTime dt) =>
      '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')}';

  Color _parseColor(String? hex) {
    if (hex == null || hex.isEmpty) return Colors.blue;
    final cleaned = hex.replaceAll('#', '');
    return Color(int.parse('FF$cleaned', radix: 16));
  }
}

class _FolderDialog extends StatefulWidget {
  final TextEditingController controller;
  final String initialColor;
  final bool isEdit;

  const _FolderDialog({
    required this.controller,
    required this.initialColor,
    required this.isEdit,
  });

  @override
  State<_FolderDialog> createState() => _FolderDialogState();
}

class _FolderDialogState extends State<_FolderDialog> {
  late String _selectedColor;

  static const _colors = ['#4CAF50', '#2196F3', '#FF9800', '#9C27B0', '#F44336', '#607D8B'];

  @override
  void initState() {
    super.initState();
    _selectedColor = widget.initialColor;
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(widget.isEdit ? '编辑文件夹' : '新建文件夹'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          TextField(
            controller: widget.controller,
            autofocus: true,
            decoration: const InputDecoration(
              labelText: '文件夹名称',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          const Text('颜色', style: TextStyle(fontSize: 13, color: Colors.grey)),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            children: _colors.map((c) {
              final selected = c == _selectedColor;
              return GestureDetector(
                onTap: () => setState(() => _selectedColor = c),
                child: Container(
                  width: 36,
                  height: 36,
                  decoration: BoxDecoration(
                    color: Color(int.parse('FF${c.replaceAll('#', '')}', radix: 16)),
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: selected ? Colors.white : Colors.transparent,
                      width: 3,
                    ),
                    boxShadow: [
                      if (selected)
                        BoxShadow(
                          color: Colors.black26,
                          blurRadius: 4,
                          offset: const Offset(0, 2),
                        ),
                    ],
                  ),
                  child: selected ? const Icon(Icons.check, color: Colors.white, size: 18) : null,
                ),
              );
            }).toList(),
          ),
        ],
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context), child: const Text('取消')),
        FilledButton(
          onPressed: widget.controller.text.trim().isEmpty
              ? null
              : () => Navigator.pop(context, {'name': widget.controller.text.trim(), 'color': _selectedColor}),
          child: Text(widget.isEdit ? '保存' : '创建'),
        ),
      ],
    );
  }
}
