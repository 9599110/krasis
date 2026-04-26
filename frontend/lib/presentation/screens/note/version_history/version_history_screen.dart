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
        title: const Text('Version History'),
      ),
      body: state.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.red),
              const SizedBox(height: 8),
              Text('Failed to load: $e'),
            ],
          ),
        ),
        data: (versions) {
          if (versions.isEmpty) {
            return const Center(child: Text('No version history available'));
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
                    : 'Version ${v.version}'),
                subtitle: Text(
                  '${_formatDate(v.createdAt)}${v.changedBy != null ? ' by ${v.changedBy}' : ''}',
                ),
                trailing: OutlinedButton(
                  onPressed: () => _showRestoreDialog(context, ref, v.version),
                  child: const Text('Restore'),
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
        title: const Text('Restore Version'),
        content: Text(
          'Are you sure you want to restore version $version? '
          'Current changes will be saved as a new version.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              ref.read(versionListProvider(noteId).notifier)
                  .restoreVersion(version);
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('Restored to version $version')),
              );
            },
            child: const Text('Restore'),
          ),
        ],
      ),
    );
  }

  String _formatDate(DateTime dt) {
    final now = DateTime.now();
    final diff = now.difference(dt);
    if (diff.inMinutes < 1) return 'just now';
    if (diff.inHours < 1) return '${diff.inMinutes}m ago';
    if (diff.inDays < 1) return '${diff.inHours}h ago';
    if (diff.inDays < 7) return '${diff.inDays}d ago';
    return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')}';
  }
}
