import 'dart:async';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:flutter_quill/flutter_quill.dart' as quill;
import 'package:markdown/markdown.dart' as md;
import 'package:markdown_quill/markdown_quill.dart';
import 'package:record/record.dart';
import 'package:dio/dio.dart';
import 'package:path_provider/path_provider.dart';
import '../../providers/note_provider.dart';
import '../../../core/errors/exceptions.dart';
import '../../providers/auth_provider.dart';

class NoteEditorScreen extends ConsumerStatefulWidget {
  final String noteId;

  const NoteEditorScreen({super.key, required this.noteId});

  @override
  ConsumerState<NoteEditorScreen> createState() => _NoteEditorScreenState();
}

class _NoteEditorScreenState extends ConsumerState<NoteEditorScreen> {
  late TextEditingController _titleController;
  late quill.QuillController _quillController;
  final FocusNode _contentFocusNode = FocusNode();
  final ScrollController _editScrollController = ScrollController();
  bool _hasChanges = false;
  bool _isSaving = false;
  int _currentVersion = 0;
  final _recorder = Record();
  bool _isRecording = false;

  static const int _titleMaxLen = 80;

  String _deriveTitleFromText(String text) {
    final t = text.replaceAll('\u0000', '').trim();
    if (t.isEmpty) return '无标题笔记';
    final lines = t.split(RegExp(r'\r?\n')).map((e) => e.trim()).where((e) => e.isNotEmpty).toList();
    final firstLine = lines.isNotEmpty ? lines.first : t;
    final sentence = firstLine.split(RegExp(r'[。！？!?]')).first.trim();
    final normalized = sentence.replaceAll(RegExp(r'\s+'), ' ').trim();
    return normalized.length > _titleMaxLen ? normalized.substring(0, _titleMaxLen) : normalized;
  }

  @override
  void initState() {
    super.initState();
    _titleController = TextEditingController();
    _titleController.addListener(_markAsChanged);

    _quillController = quill.QuillController(
      document: quill.Document(),
      selection: const TextSelection.collapsed(offset: 0),
    );
    _quillController.readOnly = false;
    _quillController.addListener(_markAsChanged);

    if (widget.noteId != 'new') {
      _loadNote();
    }
  }

  @override
  void dispose() {
    _titleController.dispose();
    _quillController.dispose();
    _contentFocusNode.dispose();
    _editScrollController.dispose();
    _recorder.dispose();
    super.dispose();
  }

  void _markAsChanged() {
    if (!_hasChanges) setState(() => _hasChanges = true);
  }

  Future<void> _loadNote() async {
    final note = await ref.read(noteEditorProvider(widget.noteId).notifier).load();
    if (note != null && mounted) {
      _titleController.text = note.title;
      final mdDoc = md.Document(encodeHtml: false, extensionSet: md.ExtensionSet.gitHubFlavored);
      final delta = MarkdownToDelta(markdownDocument: mdDoc).convert(note.content);
      _quillController = quill.QuillController(
        document: quill.Document.fromDelta(delta),
        selection: const TextSelection.collapsed(offset: 0),
      );
      _quillController.readOnly = false;
      _quillController.addListener(_markAsChanged);
      _currentVersion = note.version;
      setState(() {});
    }
  }

  Future<void> _saveNote() async {
    var title = _titleController.text.trim();
    if (title.isEmpty) {
      title = _deriveTitleFromText(_quillController.document.toPlainText());
      _titleController.text = title;
    }

    setState(() => _isSaving = true);

    try {
      // Requirement: UI is WYSIWYG, but we store Markdown.
      // Pasted markdown should be stored as-is (treated as plain text in editor).
      final contentMarkdown = DeltaToMarkdown().convert(_quillController.document.toDelta());
      if (widget.noteId == 'new') {
        final note = await ref.read(noteEditorProvider('new').notifier).createNote(
              title: title,
              content: contentMarkdown,
            );
        if (mounted) {
          context.replace('/notes/note/${note.id}');
        }
      } else {
        await ref.read(noteEditorProvider(widget.noteId).notifier).updateNote(
              title: title,
              content: contentMarkdown,
              version: _currentVersion,
            );
        _currentVersion++;
        setState(() => _hasChanges = false);
      }

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('保存成功')),
        );
      }
    } on VersionConflictException {
      _showConflictDialog();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('保存失败: $e')),
        );
      }
    } finally {
      setState(() => _isSaving = false);
    }
  }

  void _showConflictDialog() {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('版本冲突'),
        content: const Text('此笔记已被其他人修改，是否覆盖服务器版本？'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('取消'),
          ),
          TextButton(
            onPressed: () {
              Navigator.pop(ctx);
              _loadNote();
            },
            child: const Text('使用服务器版本'),
          ),
          ElevatedButton(
            onPressed: () async {
              Navigator.pop(ctx);
              await _saveNote();
            },
            child: const Text('覆盖'),
          ),
        ],
      ),
    );
  }

  String _buildVoiceFileName() {
    final d = DateTime.now();
    String pad2(int n) => n.toString().padLeft(2, '0');
    final yyyy = d.year.toString();
    final mm = pad2(d.month);
    final dd = pad2(d.day);
    final hh = pad2(d.hour);
    final mi = pad2(d.minute);
    final ss = pad2(d.second);
    return '${yyyy}${mm}${dd}_${hh}${mi}${ss}.m4a';
  }

  Future<void> _toggleVoiceInput() async {
    try {
      if (_isRecording) {
        final path = await _recorder.stop();
        if (mounted) setState(() => _isRecording = false);
        if (path == null || path.isEmpty) return;

        // Presign upload
        final api = ref.read(apiClientProvider);
        final fileName = _buildVoiceFileName();
        final noteId = widget.noteId == 'new' ? null : widget.noteId;
        final presignRes = await api.get(
          '/files/presign',
          queryParameters: {
            'file_name': fileName,
            'file_type': 'audio',
            if (noteId != null) 'note_id': noteId,
          },
        );
        final data = presignRes.data?['data'] as Map<String, dynamic>;
        final fileId = data['file_id'] as String;
        final uploadUrl = data['upload_url'] as String;

        // Upload to presigned URL (no auth headers)
        final dio = Dio();
        final bytes = await File(path).readAsBytes();
        await dio.put(
          uploadUrl,
          data: Stream.fromIterable([bytes]),
          options: Options(
            headers: {
              'Content-Type': 'audio/mp4',
              'Content-Length': bytes.length,
            },
          ),
        );

        // Confirm
        await api.post(
          '/files/confirm',
          data: {
            'file_id': fileId,
            if (noteId != null) 'note_id': noteId,
          },
        );

        // Insert reference text into editor
        _quillController.document.insert(
          _quillController.selection.baseOffset,
          '\n[语音 $fileName](file:$fileId)\n',
        );
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('语音已上传')),
          );
        }
        return;
      }

      final ok = await _recorder.hasPermission();
      if (!ok) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('没有麦克风权限')),
          );
        }
        return;
      }

      final dir = await getTemporaryDirectory();
      final fullPath = '${dir.path}/${_buildVoiceFileName()}';
      await _recorder.start(
        path: fullPath,
        encoder: AudioEncoder.aacLc,
        bitRate: 128000,
        samplingRate: 44100,
      );
      if (mounted) {
        setState(() => _isRecording = true);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('开始录音，再次点击停止')),
        );
      }
    }
    on PlatformException catch (e) {
      if (mounted) {
        setState(() => _isRecording = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('语音功能不可用: ${e.code}')),
        );
      }
    } catch (e) {
      if (mounted) {
        setState(() => _isRecording = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('语音功能异常: $e')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final noteState = ref.watch(noteEditorProvider(widget.noteId));

    return Scaffold(
      appBar: AppBar(
        title: TextField(
          controller: _titleController,
          style: Theme.of(context).textTheme.titleLarge,
          decoration: const InputDecoration(
            hintText: '笔记标题',
            border: InputBorder.none,
            isDense: true,
          ),
        ),
        actions: [
          IconButton(
            icon: Icon(_isRecording ? Icons.stop_circle : Icons.mic),
            tooltip: _isRecording ? '停止语音输入' : '语音输入',
            onPressed: () async {
              try {
                await _toggleVoiceInput();
              } catch (e) {
                if (mounted) {
                  setState(() => _isRecording = false);
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text('语音功能异常: $e')),
                  );
                }
              }
            },
          ),
          if (widget.noteId != 'new') ...[
            IconButton(
              icon: const Icon(Icons.share),
              tooltip: '分享',
              onPressed: () => context.push('/notes/note/${widget.noteId}/share'),
            ),
            IconButton(
              icon: const Icon(Icons.history),
              tooltip: '版本历史',
              onPressed: () => context.push('/notes/note/${widget.noteId}/versions'),
            ),
          ],
          if (_hasChanges || _isSaving)
            IconButton(
              icon: _isSaving
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.save),
              onPressed: _isSaving ? null : _saveNote,
            ),
        ],
      ),
      body: noteState.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => widget.noteId == 'new'
            ? _buildEditorBody()
            : Center(child: Text('加载失败: $e')),
        data: (_) => _buildEditorBody(),
      ),
    );
  }

  Widget _buildEditorBody() {
    return _buildEditPane();
  }

  Widget _buildEditPane() {
    return Column(
      children: [
        quill.QuillSimpleToolbar(
          controller: _quillController,
          config: const quill.QuillSimpleToolbarConfig(
            multiRowsDisplay: false,
            showCodeBlock: true,
            showInlineCode: true,
            showListBullets: true,
            showListNumbers: true,
            showUndo: true,
            showRedo: true,
            showLink: true,
            showFontFamily: false,
            showFontSize: false,
            showSearchButton: false,
            showSubscript: false,
            showSuperscript: false,
          ),
        ),
        const Divider(height: 1),
        Expanded(
          child: SingleChildScrollView(
            controller: _editScrollController,
            padding: const EdgeInsets.all(16),
            child: quill.QuillEditor.basic(
              controller: _quillController,
              config: quill.QuillEditorConfig(
                placeholder: '开始写作...',
              ),
            ),
          ),
        ),
      ],
    );
  }
}
