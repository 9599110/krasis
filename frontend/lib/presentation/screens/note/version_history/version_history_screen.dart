import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../providers/version_provider.dart';

class VersionHistoryScreen extends ConsumerWidget {
  final String noteId;

  const VersionHistoryScreen({super.key, required this.noteId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(versionListProvider(noteId));
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('版本历史'),
      ),
      body: state.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.red),
              const SizedBox(height: 8),
              Text('加载失败: $e'),
            ],
          ),
        ),
        data: (versions) {
          if (versions.isEmpty) {
            return const Center(child: Text('暂无版本历史'));
          }
          return ListView.separated(
            itemCount: versions.length,
            separatorBuilder: (_, __) => const Divider(height: 1),
            itemBuilder: (context, index) {
              final v = versions[index];
              return ListTile(
                leading: CircleAvatar(
                  backgroundColor: theme.colorScheme.primaryContainer,
                  child: Text(
                    'v${v.version}',
                    style: TextStyle(
                      color: theme.colorScheme.onPrimaryContainer,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
                title: Text(v.changeSummary?.isNotEmpty == true
                    ? v.changeSummary!
                    : '版本 ${v.version}'),
                subtitle: Text(
                  '${_formatDate(v.createdAt)}${v.changedBy != null ? ' 由 ${v.changedBy}' : ''}',
                ),
                trailing: OutlinedButton(
                  onPressed: () => _showRestoreDialog(context, ref, v.version),
                  child: const Text('恢复'),
                ),
              );
            },
          );
        },
      ),
    );
  }

  void _showRestoreDialog(BuildContext context, WidgetRef ref, int version) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('恢复版本'),
        content: Text(
          '确定要恢复到版本 $version 吗？\n当前版本会被备份为新版本。',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () {
              ref.read(versionListProvider(noteId).notifier)
                  .restoreVersion(version);
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('已恢复到版本 $version')),
              );
            },
            child: const Text('恢复'),
          ),
        ],
      ),
    );
  }

  String _formatDate(DateTime dt) {
    final now = DateTime.now();
    final diff = now.difference(dt);
    if (diff.inMinutes < 1) return '刚刚';
    if (diff.inHours < 1) return '${diff.inMinutes}分钟前';
    if (diff.inDays < 1) return '${diff.inHours}小时前';
    if (diff.inDays < 7) return '${diff.inDays}天前';
    return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')}';
  }
}
