import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import '../providers/ai_provider.dart';
import '../../data/models/ai_message_model.dart';

class AIFloatingDialog extends ConsumerStatefulWidget {
  final VoidCallback onClose;
  const AIFloatingDialog({super.key, required this.onClose});

  @override
  ConsumerState<AIFloatingDialog> createState() => _AIFloatingDialogState();
}

class _AIFloatingDialogState extends ConsumerState<AIFloatingDialog> {
  final _inputController = TextEditingController();
  final _scrollController = ScrollController();
  bool _isMinimized = false;

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

    try {
      final stream = ref.read(aiChatProvider('').notifier).ask(question);
      await for (final _ in stream) {}
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('提问失败: $e')),
        );
      }
    }
  }

  void _close() => widget.onClose();

  @override
  Widget build(BuildContext context) {
    final chatState = ref.watch(aiChatProvider(''));
    final theme = Theme.of(context);
    const dialogWidth = 400.0;
    const dialogHeight = 600.0;

    if (_isMinimized) {
      return Positioned(
        right: 16,
        bottom: 16,
        child: Material(
          elevation: 8,
          borderRadius: BorderRadius.circular(28),
          child: InkWell(
            onTap: () => setState(() => _isMinimized = false),
            borderRadius: BorderRadius.circular(28),
            child: Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: theme.colorScheme.primary,
                borderRadius: BorderRadius.circular(28),
              ),
              child: Icon(Icons.smart_toy, color: theme.colorScheme.onPrimary),
            ),
          ),
        ),
      );
    }

    return Positioned(
      right: 16,
      bottom: 16,
      child: Material(
        elevation: 12,
        borderRadius: BorderRadius.circular(16),
        child: Container(
          width: dialogWidth,
          height: dialogHeight,
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: theme.colorScheme.outlineVariant, width: 0.5),
          ),
          child: Column(
            children: [
              // Header
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primaryContainer,
                  borderRadius: const BorderRadius.only(
                    topLeft: Radius.circular(16),
                    topRight: Radius.circular(16),
                  ),
                ),
                child: Row(
                  children: [
                    Icon(Icons.smart_toy, color: theme.colorScheme.onPrimaryContainer),
                    const SizedBox(width: 8),
                    Text(
                      'AI 助手',
                      style: TextStyle(
                        fontWeight: FontWeight.bold,
                        color: theme.colorScheme.onPrimaryContainer,
                      ),
                    ),
                    const Spacer(),
                    IconButton(
                      icon: const Icon(Icons.remove, size: 20),
                      tooltip: '最小化',
                      onPressed: () => setState(() => _isMinimized = true),
                      constraints: const BoxConstraints(),
                      padding: const EdgeInsets.all(4),
                    ),
                    const SizedBox(width: 4),
                    IconButton(
                      icon: const Icon(Icons.close, size: 20),
                      tooltip: '关闭',
                      onPressed: _close,
                      constraints: const BoxConstraints(),
                      padding: const EdgeInsets.all(4),
                    ),
                  ],
                ),
              ),
              // Messages
              Expanded(
                child: chatState.messages.isEmpty && chatState.streamingText.isEmpty
                    ? _buildWelcomeState()
                    : _buildMessageList(chatState),
              ),
              // Streaming indicator
              if (chatState.isStreaming) _buildStreamingIndicator(chatState),
              // Input bar
              _buildInputBar(),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildWelcomeState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.smart_toy_outlined,
            size: 48,
            color: Theme.of(context).colorScheme.primary,
          ),
          const SizedBox(height: 12),
          Text(
            'AI 助手',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
          ),
          const SizedBox(height: 4),
          Text(
            '向 AI 提问关于你的笔记内容',
            style: TextStyle(color: Colors.grey.shade500, fontSize: 13),
          ),
        ],
      ),
    );
  }

  Widget _buildMessageList(AIChatState chatState) {
    return ListView.builder(
      controller: _scrollController,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      itemCount: chatState.messages.length,
      itemBuilder: (context, index) {
        final msg = chatState.messages[index];
        return _MessageBubble(message: msg);
      },
    );
  }

  Widget _buildStreamingIndicator(AIChatState chatState) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.primaryContainer.withOpacity(0.3),
        border: Border(
          top: BorderSide(color: Theme.of(context).colorScheme.outlineVariant),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: chatState.streamingText.isEmpty
                ? Text('AI 正在思考...', style: TextStyle(color: Colors.grey.shade600, fontSize: 13))
                : MarkdownBody(
                    data: chatState.streamingText,
                    styleSheet: MarkdownStyleSheet(
                      p: Theme.of(context).textTheme.bodySmall,
                    ),
                  ),
          ),
        ],
      ),
    );
  }

  Widget _buildInputBar() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(
          top: BorderSide(color: Theme.of(context).colorScheme.outlineVariant),
        ),
      ),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _inputController,
              decoration: InputDecoration(
                hintText: '向 AI 提问...',
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(20),
                ),
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 8,
                ),
                isDense: true,
              ),
              maxLines: 3,
              minLines: 1,
              onSubmitted: (_) => _sendMessage(),
            ),
          ),
          const SizedBox(width: 8),
          IconButton.filled(
            onPressed: _sendMessage,
            icon: const Icon(Icons.send, size: 16),
            style: IconButton.styleFrom(
              minimumSize: const Size(36, 36),
            ),
          ),
        ],
      ),
    );
  }
}

class _MessageBubble extends StatelessWidget {
  final AIMessageModel message;

  const _MessageBubble({required this.message});

  @override
  Widget build(BuildContext context) {
    final isUser = message.role == 'user';

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        mainAxisAlignment: isUser ? MainAxisAlignment.end : MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!isUser) ...[
            CircleAvatar(
              radius: 14,
              backgroundColor: Theme.of(context).colorScheme.primaryContainer,
              child: Icon(Icons.smart_toy, size: 16,
                  color: Theme.of(context).colorScheme.onPrimaryContainer),
            ),
            const SizedBox(width: 6),
          ],
          Flexible(
            child: Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: isUser
                    ? Theme.of(context).colorScheme.primary
                    : Theme.of(context).colorScheme.surfaceVariant,
                borderRadius: BorderRadius.circular(12),
              ),
              child: isUser
                  ? Text(
                      message.content,
                      style: TextStyle(
                        color: isUser ? Theme.of(context).colorScheme.onPrimary : null,
                      ),
                    )
                  : MarkdownBody(
                      data: message.content,
                      styleSheet: MarkdownStyleSheet(
                        p: TextStyle(
                          color: isUser ? Theme.of(context).colorScheme.onPrimary : null,
                        ),
                      ),
                    ),
            ),
          ),
          if (isUser) const SizedBox(width: 6),
        ],
      ),
    );
  }
}
