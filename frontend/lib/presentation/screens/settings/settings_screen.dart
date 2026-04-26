import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../providers/auth_provider.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: const Text('设置'),
      ),
      body: ListView(
        children: [
          _Section(
            title: '外观',
            children: [
              _ThemeToggleTile(),
            ],
          ),
          const Divider(height: 1),
          _Section(
            title: '存储',
            children: [
              ListTile(
                leading: const Icon(Icons.storage_outlined),
                title: const Text('缓存管理'),
                trailing: OutlinedButton(
                  onPressed: () {},
                  child: const Text('清除缓存'),
                ),
              ),
            ],
          ),
          const Divider(height: 1),
          _Section(
            title: '关于',
            children: [
              ListTile(
                leading: const Icon(Icons.info_outline),
                title: const Text('版本'),
                trailing: const Text('v0.1.0'),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _ThemeToggleTile extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;

    return ListTile(
      leading: Icon(isDark ? Icons.dark_mode : Icons.light_mode_outlined),
      title: const Text('深色模式'),
      trailing: Switch(
        value: isDark,
        onChanged: (value) {
          ref.read(themeModeProvider.notifier).state =
              value ? ThemeMode.dark : ThemeMode.light;
        },
      ),
    );
  }
}

class _Section extends StatelessWidget {
  final String title;
  final List<Widget> children;

  const _Section({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
          child: Text(
            title,
            style: TextStyle(
              color: Colors.grey.shade500,
              fontSize: 13,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
        ...children,
      ],
    );
  }
}
