import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../providers/share_provider.dart';

class ShareScreen extends ConsumerWidget {
  final String noteId;

  const ShareScreen({super.key, required this.noteId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(shareProvider(noteId));
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Share Note'),
      ),
      body: state.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, st) => Center(child: Text('Error: $e')),
        data: (share) {
          if (share == null) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(Icons.share_outlined, size: 64, color: Colors.grey),
                  const SizedBox(height: 16),
                  const Text('This note is not shared yet'),
                  const SizedBox(height: 24),
                  FilledButton.icon(
                    onPressed: () => ref
                        .read(shareProvider(noteId).notifier)
                        .createShare(),
                    icon: const Icon(Icons.add_link),
                    label: const Text('Create Share Link'),
                  ),
                ],
              ),
            );
          }

          final shareUrl = '${_baseUrl(context)}/share/${share.token}';

          return Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Share Link',
                            style: theme.textTheme.titleSmall),
                        const SizedBox(height: 8),
                        Row(
                          children: [
                            Expanded(
                              child: Text(
                                shareUrl,
                                style: theme.textTheme.bodyMedium?.copyWith(
                                  color: theme.colorScheme.primary,
                                ),
                              ),
                            ),
                            IconButton(
                              icon: const Icon(Icons.copy),
                              onPressed: () {
                                Clipboard.setData(ClipboardData(text: shareUrl));
                                ScaffoldMessenger.of(context).showSnackBar(
                                  const SnackBar(
                                      content: Text('Link copied!')),
                                );
                              },
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Settings',
                            style: theme.textTheme.titleSmall),
                        const SizedBox(height: 12),
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            const Text('Permission'),
                            Text(share.permission,
                                style: theme.textTheme.bodyMedium?.copyWith(
                                  fontWeight: FontWeight.bold,
                                )),
                          ],
                        ),
                        const Divider(),
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            const Text('Status'),
                            Chip(
                              label: Text(
                                share.isActive ? 'Active' : 'Inactive',
                                style: const TextStyle(fontSize: 12),
                              ),
                              backgroundColor: share.isActive
                                  ? Colors.green.withOpacity(0.2)
                                  : Colors.grey.withOpacity(0.2),
                            ),
                          ],
                        ),
                        if (share.expiresAt != null) ...[
                          const Divider(),
                          Row(
                            mainAxisAlignment: MainAxisAlignment.spaceBetween,
                            children: [
                              const Text('Expires'),
                              Text(_formatDate(share.expiresAt!)),
                            ],
                          ),
                        ],
                        const Divider(),
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            const Text('Created'),
                            Text(_formatDate(share.createdAt)),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
                const Spacer(),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton.icon(
                    onPressed: () => _showRevokeDialog(context, ref),
                    icon: const Icon(Icons.link_off, color: Colors.red),
                    label: const Text('Revoke Share Link',
                        style: TextStyle(color: Colors.red)),
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }

  String _baseUrl(BuildContext context) {
    // Derive from current API base URL
    return 'https://krasis.app';
  }

  void _showRevokeDialog(BuildContext context, WidgetRef ref) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Revoke Share Link'),
        content: const Text(
          'The share link will no longer work. '
          'Anyone with the link will lose access.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () {
              ref.read(shareProvider(noteId).notifier).revokeShare();
              Navigator.pop(ctx);
            },
            child: const Text('Revoke'),
          ),
        ],
      ),
    );
  }

  String _formatDate(DateTime dt) {
    return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')} '
        '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
  }
}
