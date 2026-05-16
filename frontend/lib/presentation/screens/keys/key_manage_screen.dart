import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:dio/dio.dart' show Response;
import '../../../core/crypto/key_store.dart' as key_store;
import '../../providers/auth_provider.dart';

class KeyManageScreen extends ConsumerStatefulWidget {
  const KeyManageScreen({super.key});

  @override
  ConsumerState<KeyManageScreen> createState() => _KeyManageScreenState();
}

class _KeyManageScreenState extends ConsumerState<KeyManageScreen> {
  bool _keyExists = false;
  bool _keyUnlocked = false;
  bool _loading = false;
  String _publicKeyHex = '';
  String _publicKeyPem = '';

  // Sync status
  bool _syncing = false;

  // QR display
  String _qrData = '';

  // QR scan
  bool _scanProcessing = false;
  String _scanError = '';

  // Generate dialog
  final _genPasswordController = TextEditingController();
  final _genPasswordConfirmController = TextEditingController();
  bool _showGenerateDialog = false;

  // Unlock dialog
  final _unlockPasswordController = TextEditingController();
  bool _showUnlockDialog = false;

  @override
  void initState() {
    super.initState();
    _initSyncApiCaller();
    // Kick off async init chain: refresh then auto sync
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _initialSync();
    });
  }

  Future<void> _initialSync() async {
    await _refreshStatus();
    await _doAutoSync();
  }

  void _initSyncApiCaller() {
    key_store.setSyncApiCaller((String method, String path,
        {Map<String, dynamic>? data}) async {
      final api = ref.read(apiClientProvider);
      Response<Map<String, dynamic>> response;
      switch (method) {
        case 'GET':
          response = await api.get(path);
          break;
        case 'PUT':
          response = await api.put(path, data: data);
          break;
        case 'DELETE':
          response = await api.delete(path);
          break;
        default:
          throw UnsupportedError('Unsupported method: $method');
      }
      return response.data ?? {};
    });
  }

  Future<void> _doAutoSync() async {
    setState(() => _syncing = true);
    await key_store.autoSyncKeys();
    if (mounted) setState(() => _syncing = false);
    await _refreshStatus();
  }

  Future<void> _syncToServer() async {
    setState(() => _syncing = true);
    await key_store.syncToServer();
    if (mounted) {
      setState(() => _syncing = false);
      _showSnackBar('密钥已同步至云端');
    }
  }

  Future<void> _syncFromServer() async {
    setState(() => _syncing = true);
    final ok = await key_store.syncFromServer(force: true);
    if (mounted) {
      setState(() => _syncing = false);
      if (ok) {
        _showSnackBar('密钥已从云端同步到本地');
        await _refreshStatus();
      } else {
        _showSnackBar('云端无可用密钥或同步失败');
      }
    }
  }

  @override
  void dispose() {
    _genPasswordController.dispose();
    _genPasswordConfirmController.dispose();
    _unlockPasswordController.dispose();
    super.dispose();
  }

  Future<void> _refreshStatus() async {
    final exists = await key_store.hasStoredKey();
    final unlocked = key_store.hasSessionKey();
    String hex = '';
    String pem = '';
    if (unlocked) {
      final pub = key_store.getSessionPublicKey();
      if (pub != null) {
        hex = key_store.publicKeyToHex(pub);
        pem = key_store.publicKeyToPem(pub);
      }
    }
    if (mounted) {
      setState(() {
        _keyExists = exists;
        _keyUnlocked = unlocked;
        _publicKeyHex = hex;
        _publicKeyPem = pem;
      });
    }
  }

  Future<void> _handleGenerate() async {
    final password = _genPasswordController.text;
    final confirm = _genPasswordConfirmController.text;

    if (password.length < 6) {
      _showSnackBar('密码至少 6 位');
      return;
    }
    if (password != confirm) {
      _showSnackBar('两次输入的密码不一致');
      return;
    }

    setState(() => _loading = true);
    try {
      await key_store.generateKeyPair(password);
      _showSnackBar('ECC 密钥对已生成并安全存储');
      _genPasswordController.clear();
      _genPasswordConfirmController.clear();
      setState(() => _showGenerateDialog = false);
      await _refreshStatus();
      await key_store.syncToServer();
    } catch (e) {
      _showSnackBar('密钥生成失败: $e');
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _handleUnlock() async {
    final password = _unlockPasswordController.text;
    if (password.isEmpty) {
      _showSnackBar('请输入密码');
      return;
    }

    setState(() => _loading = true);
    try {
      final ok = await key_store.unlockKey(password);
      if (ok) {
        _showSnackBar('密钥已解锁，当前会话可用');
        _unlockPasswordController.clear();
        setState(() => _showUnlockDialog = false);
        await _refreshStatus();
        await key_store.syncToServer();
      } else {
        _showSnackBar('密码错误，无法解锁密钥');
      }
    } catch (e) {
      _showSnackBar('解锁失败: $e');
    } finally {
      setState(() => _loading = false);
    }
  }

  void _handleLock() {
    key_store.clearSessionKey();
    setState(() {
      _keyUnlocked = false;
      _publicKeyHex = '';
      _publicKeyPem = '';
    });
    _showSnackBar('密钥已锁定');
  }

  Future<void> _handleDeleteKey() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除密钥'),
        content: const Text(
          '确定要删除存储的 ECC 密钥吗？已加密的笔记将无法解密！同时将从云端移除同步的密钥。请确保已备份。',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('删除', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );

    if (confirmed == true) {
      try {
        await key_store.deleteStoredKey();
        await key_store.deleteKeysFromServer();
        _showSnackBar('密钥已删除（本地 + 云端）');
        await _refreshStatus();
      } catch (e) {
        _showSnackBar('删除失败: $e');
      }
    }
  }

  void _copyPublicKey() {
    Clipboard.setData(ClipboardData(text: _publicKeyPem));
    _showSnackBar('公钥已复制到剪贴板');
  }

  Future<void> _showQRCode() async {
    final json = await key_store.getQRKeyPackage();
    if (json == null) {
      _showSnackBar('未找到密钥');
      return;
    }
    if (!mounted) return;
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        contentPadding: const EdgeInsets.all(24),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text(
              '私钥二维码（离线分享）',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 4),
            Text(
              '另一设备扫描此二维码即可导入私钥',
              style: TextStyle(fontSize: 12, color: Colors.grey.shade600),
            ),
            const SizedBox(height: 20),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.grey.shade200),
              ),
              child: QrImageView(
                data: json,
                version: QrVersions.auto,
                size: 260,
                eyeStyle: QrEyeStyle(
                  eyeShape: QrEyeShape.square,
                  color: Colors.black,
                ),
                dataModuleStyle: const QrDataModuleStyle(
                  dataModuleShape: QrDataModuleShape.square,
                  color: Colors.black,
                ),
              ),
            ),
            const SizedBox(height: 16),
            Text(
              '注意：二维码包含加密私钥，请确保在可信环境中展示',
              style: TextStyle(fontSize: 11, color: Colors.grey.shade500),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('关闭'),
            ),
          ],
        ),
      ),
    );
  }

  void _onDetectQR(BarcodeCapture capture) {
    if (_scanProcessing) return;

    final barcode = capture.barcodes.firstOrNull;
    if (barcode == null || barcode.rawValue == null) return;

    _scanProcessing = true;
    setState(() => _scanError = '');

    _importFromQRJson(barcode.rawValue!);
  }

  Future<void> _importFromQRJson(String jsonStr) async {
    try {
      final ok = await key_store.importQRKeyPackage(jsonStr);
      if (ok) {
        _showSnackBar('二维码导入密钥成功');
        if (mounted) Navigator.pop(context);
        await key_store.syncToServer();
        await _refreshStatus();
      } else {
        setState(() {
          _scanError = '二维码密钥包格式无效';
          _scanProcessing = false;
        });
      }
    } catch (e) {
      setState(() {
        _scanError = '导入失败: $e';
        _scanProcessing = false;
      });
    }
  }

  Future<void> _scanQRCode() async {
    setState(() {
      _scanProcessing = false;
      _scanError = '';
    });
    await Navigator.push<void>(
      context,
      MaterialPageRoute(
        builder: (ctx) => Scaffold(
          appBar: AppBar(
            title: const Text('扫码导入私钥'),
            leading: IconButton(
              icon: const Icon(Icons.close),
              onPressed: () => Navigator.pop(ctx),
            ),
          ),
          body: _buildScannerBody(ctx),
        ),
      ),
    );
  }

  Widget _buildScannerBody(BuildContext ctx) {
    return Column(
      children: [
        Expanded(
          child: MobileScanner(
            onDetect: (capture) {
              if (_scanProcessing) return;
              final barcode = capture.barcodes.firstOrNull;
              if (barcode == null || barcode.rawValue == null) return;
              _scanProcessing = true;
              setState(() => _scanError = '');
              _importFromQRJson(barcode.rawValue!);
            },
          ),
        ),
        if (_scanError.isNotEmpty)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(
              _scanError,
              style: const TextStyle(color: Colors.red),
            ),
          ),
        if (_scanProcessing)
          const Padding(
            padding: EdgeInsets.all(16),
            child: CircularProgressIndicator(),
          ),
      ],
    );
  }

  void _showSnackBar(String message) {
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(message)),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('ECC 加密密钥管理'),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(
            '使用 ECDH P-256 + AES-256-GCM 对笔记进行端到端加密。'
            '密钥自动同步到云端，可在不同设备使用同一密钥。',
            style: TextStyle(color: Colors.grey.shade600, fontSize: 13),
          ),
          const SizedBox(height: 20),
          _StatusCard(
            keyExists: _keyExists,
            keyUnlocked: _keyUnlocked,
            publicKeyHex: _publicKeyHex,
            isSyncing: _syncing,
            onGenerate: () => setState(() => _showGenerateDialog = true),
            onUnlock: () => setState(() => _showUnlockDialog = true),
            onLock: _handleLock,
            onSyncFromServer: _syncFromServer,
          ),
          const SizedBox(height: 16),
          if (_keyUnlocked) ...[
            _KeyDetailsCard(
              publicKeyHex: _publicKeyHex,
              publicKeyPem: _publicKeyPem,
              onCopy: _copyPublicKey,
            ),
            const SizedBox(height: 16),
          ],
          if (_keyExists) ...[
            _SyncCard(
              keyUnlocked: _keyUnlocked,
              isSyncing: _syncing,
              onSync: _syncToServer,
              onSyncFromServer: _syncFromServer,
            ),
            const SizedBox(height: 16),
          ],
          if (_keyExists) ...[
            _QRSharingCard(
              onShowQR: _showQRCode,
              onScan: _scanQRCode,
            ),
            const SizedBox(height: 16),
          ],
          if (_keyExists) _DangerCard(onDelete: _handleDeleteKey),
        ],
      ),
      floatingActionButton: !_keyExists
          ? FloatingActionButton.extended(
              heroTag: 'key_gen',
              onPressed: () => setState(() => _showGenerateDialog = true),
              icon: const Icon(Icons.vpn_key),
              label: const Text('生成密钥'),
            )
          : null,
      floatingActionButtonLocation: FloatingActionButtonLocation.centerFloat,
    );
  }
}

// ─── Status Card ────────────────────────────────────────────────────────────

class _StatusCard extends StatelessWidget {
  final bool keyExists;
  final bool keyUnlocked;
  final String publicKeyHex;
  final bool isSyncing;
  final VoidCallback onGenerate;
  final VoidCallback onUnlock;
  final VoidCallback onLock;
  final VoidCallback onSyncFromServer;

  const _StatusCard({
    required this.keyExists,
    required this.keyUnlocked,
    required this.publicKeyHex,
    required this.isSyncing,
    required this.onGenerate,
    required this.onUnlock,
    required this.onLock,
    required this.onSyncFromServer,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Row(
          children: [
            Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                color: keyUnlocked
                    ? Colors.green.shade50
                    : keyExists
                        ? Colors.orange.shade50
                        : Colors.grey.shade100,
                borderRadius: BorderRadius.circular(24),
              ),
              child: Icon(
                keyUnlocked
                    ? Icons.lock
                    : keyExists
                        ? Icons.lock_outline
                        : Icons.no_encryption_outlined,
                color: keyUnlocked
                    ? Colors.green
                    : keyExists
                        ? Colors.orange
                        : Colors.grey,
                size: 24,
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    keyUnlocked
                        ? '密钥已解锁'
                        : keyExists
                            ? '密钥已存储，未解锁'
                            : '未创建密钥',
                    style: theme.textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    keyUnlocked
                        ? '当前会话中私钥可用，可加密/解密笔记'
                        : keyExists
                            ? '请输入密码解锁密钥以加密笔记'
                            : '首次使用请生成 ECC 密钥对',
                    style: TextStyle(
                      color: Colors.grey.shade600,
                      fontSize: 12,
                    ),
                  ),
                  if (keyUnlocked && publicKeyHex.isNotEmpty) ...[
                    const SizedBox(height: 6),
                    Text(
                      '公钥指纹：${publicKeyHex.substring(0, 32)}...',
                      style: TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 11,
                        color: Colors.grey.shade500,
                      ),
                    ),
                  ],
                  if (isSyncing) ...[
                    const SizedBox(height: 4),
                    Row(
                      children: [
                        SizedBox(
                          width: 12,
                          height: 12,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.blue.shade400,
                          ),
                        ),
                        const SizedBox(width: 6),
                        Text(
                          '同步中...',
                          style: TextStyle(
                            fontSize: 11,
                            color: Colors.blue.shade400,
                          ),
                        ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
            if (!keyExists)
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  FilledButton.tonalIcon(
                    onPressed: onGenerate,
                    icon: const Icon(Icons.add, size: 18),
                    label: const Text('生成'),
                  ),
                  const SizedBox(width: 8),
                  OutlinedButton.icon(
                    onPressed: isSyncing ? null : onSyncFromServer,
                    icon: isSyncing
                        ? SizedBox(
                            width: 14,
                            height: 14,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.blue.shade400,
                            ),
                          )
                        : const Icon(Icons.cloud_download, size: 16),
                    label: Text(isSyncing ? '同步中...' : '从云端下载'),
                    style: OutlinedButton.styleFrom(
                      visualDensity: VisualDensity.compact,
                    ),
                  ),
                ],
              )
            else if (!keyUnlocked)
              FilledButton.tonalIcon(
                onPressed: onUnlock,
                icon: const Icon(Icons.lock_open, size: 18),
                label: const Text('解锁'),
              )
            else
              OutlinedButton.icon(
                onPressed: onLock,
                icon: const Icon(Icons.lock_outline, size: 18),
                label: const Text('锁定'),
              ),
          ],
        ),
      ),
    );
  }
}

// ─── Key Details Card ───────────────────────────────────────────────────────

class _KeyDetailsCard extends StatelessWidget {
  final String publicKeyHex;
  final String publicKeyPem;
  final VoidCallback onCopy;

  const _KeyDetailsCard({
    required this.publicKeyHex,
    required this.publicKeyPem,
    required this.onCopy,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '公钥信息',
              style: Theme.of(context).textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
            ),
            const SizedBox(height: 16),
            _DetailRow(label: '算法', value: 'ECDH P-256 (secp256r1)'),
            const SizedBox(height: 12),
            _DetailRow(label: '指纹', value: publicKeyHex, isCode: true),
            const SizedBox(height: 12),
            _DetailRow(
                label: '公钥 (PEM)',
                value: publicKeyPem,
                isCode: true,
                multiLine: true),
            const SizedBox(height: 8),
            Align(
              alignment: Alignment.centerRight,
              child: OutlinedButton.icon(
                onPressed: onCopy,
                icon: const Icon(Icons.copy, size: 16),
                label: const Text('复制公钥'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DetailRow extends StatelessWidget {
  final String label;
  final String value;
  final bool isCode;
  final bool multiLine;

  const _DetailRow({
    required this.label,
    required this.value,
    this.isCode = false,
    this.multiLine = false,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 80,
          child: Text(
            label,
            style: TextStyle(
              fontSize: 13,
              color: Colors.grey.shade500,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: isCode
              ? Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Colors.grey.shade50,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: SelectableText(
                    value,
                    style: TextStyle(
                      fontFamily: 'monospace',
                      fontSize: multiLine ? 11 : 12,
                      height: multiLine ? 1.5 : 1.2,
                    ),
                  ),
                )
              : Text(value, style: const TextStyle(fontSize: 13)),
        ),
      ],
    );
  }
}

// ─── Sync Card ──────────────────────────────────────────────────────────────

class _SyncCard extends StatelessWidget {
  final bool keyUnlocked;
  final bool isSyncing;
  final VoidCallback onSync;
  final VoidCallback onSyncFromServer;

  const _SyncCard({
    required this.keyUnlocked,
    required this.isSyncing,
    required this.onSync,
    required this.onSyncFromServer,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      color: Colors.blue.shade50,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: Colors.blue.shade100),
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.cloud_sync, size: 20, color: Colors.blue.shade700),
                const SizedBox(width: 8),
                Text(
                  '云端同步',
                  style: TextStyle(
                    fontWeight: FontWeight.w600,
                    color: Colors.blue.shade700,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              '密钥经密码加密后存储到服务端，登录同一账号后自动同步到其他设备，无需手动导入导出。',
              style: TextStyle(fontSize: 13, color: Colors.grey.shade600),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                OutlinedButton.icon(
                  onPressed: isSyncing ? null : onSync,
                  icon: isSyncing
                      ? SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.blue.shade400,
                          ),
                        )
                      : Icon(Icons.cloud_upload, color: Colors.blue.shade600, size: 18),
                  label: Text(
                    isSyncing ? '同步中...' : '上传到云端',
                    style: TextStyle(color: Colors.blue.shade600),
                  ),
                  style: OutlinedButton.styleFrom(
                    side: BorderSide(color: Colors.blue.shade200),
                  ),
                ),
                const SizedBox(width: 12),
                OutlinedButton.icon(
                  onPressed: isSyncing ? null : onSyncFromServer,
                  icon: isSyncing
                      ? SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.blue.shade400,
                          ),
                        )
                      : Icon(Icons.cloud_download, color: Colors.blue.shade600, size: 18),
                  label: Text(
                    isSyncing ? '同步中...' : '从云端下载',
                    style: TextStyle(color: Colors.blue.shade600),
                  ),
                  style: OutlinedButton.styleFrom(
                    side: BorderSide(color: Colors.blue.shade200),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

// ─── QR Sharing Card ────────────────────────────────────────────────────────

class _QRSharingCard extends StatelessWidget {
  final VoidCallback onShowQR;
  final VoidCallback onScan;

  const _QRSharingCard({
    required this.onShowQR,
    required this.onScan,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      color: Colors.purple.shade50,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: Colors.purple.shade100),
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.qr_code, size: 20, color: Colors.purple.shade700),
                const SizedBox(width: 8),
                Text(
                  '二维码分享私钥（离线）',
                  style: TextStyle(
                    fontWeight: FontWeight.w600,
                    color: Colors.purple.shade700,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              '生成二维码后另一设备可扫码导入私钥。注意：二维码包含加密私钥，请确保在可信环境中展示。',
              style: TextStyle(fontSize: 13, color: Colors.grey.shade600),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                OutlinedButton.icon(
                  onPressed: onShowQR,
                  icon: Icon(Icons.qr_code,
                      color: Colors.purple.shade600, size: 18),
                  label: Text('显示二维码',
                      style: TextStyle(color: Colors.purple.shade600)),
                  style: OutlinedButton.styleFrom(
                    side: BorderSide(color: Colors.purple.shade200),
                  ),
                ),
                const SizedBox(width: 12),
                OutlinedButton.icon(
                  onPressed: onScan,
                  icon: Icon(Icons.qr_code_scanner,
                      color: Colors.purple.shade600, size: 18),
                  label: Text('扫码导入',
                      style: TextStyle(color: Colors.purple.shade600)),
                  style: OutlinedButton.styleFrom(
                    side: BorderSide(color: Colors.purple.shade200),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

// ─── Danger Card ────────────────────────────────────────────────────────────

class _DangerCard extends StatelessWidget {
  final VoidCallback onDelete;

  const _DangerCard({required this.onDelete});

  @override
  Widget build(BuildContext context) {
    return Card(
      color: Colors.red.shade50,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: Colors.red.shade100),
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '危险操作',
              style: TextStyle(
                fontWeight: FontWeight.w600,
                color: Colors.red.shade700,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              '删除密钥后，所有已加密的笔记将无法解密。同时将从云端移除同步的密钥。请确保已备份。',
              style: TextStyle(
                fontSize: 13,
                color: Colors.grey.shade600,
              ),
            ),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: onDelete,
              icon: const Icon(Icons.delete_forever, color: Colors.red),
              label: const Text('删除密钥（本地 + 云端）',
                  style: TextStyle(color: Colors.red)),
              style: OutlinedButton.styleFrom(
                side: BorderSide(color: Colors.red.shade300),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
