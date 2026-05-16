/**
 * High-security ECC key management and note encryption/decryption for Flutter.
 *
 * Equivalent to the TypeScript keyStore.ts — same key hierarchy:
 *   1. User password (never stored)
 *   2. PBKDF2(password + salt) → Key Encryption Key (KEK)
 *   3. KEK wraps ECDH private key via AES-GCM
 *   4. ECDH P-256 key pair for note encryption
 *   5. ECDH deriveBits → PBKDF2(salt') → per-note AES-256-GCM key
 *
 * Storage:
 *   - SharedPreferences stores: encryptedPrivateKey (base64), salt, iv, publicKey (base64)
 *   - In-memory during session: private key bytes (raw d scalar)
 *   - Note content stored as "salt:iv:ciphertext" (base64, colon-separated)
 */

import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:pointycastle/export.dart';
import 'package:pointycastle/asn1.dart';

// ─── Constants ──────────────────────────────────────────────────────────────

const String _keyPrefix = 'krasis_ecc_';
const String _storeEncryptedKey = '${_keyPrefix}encrypted_private';
const String _storeSalt = '${_keyPrefix}salt';
const String _storeIv = '${_keyPrefix}iv';
const String _storePublicKey = '${_keyPrefix}public_key';

const int _pbkdf2Iterations = 600000;
const int _notePbkdf2Iterations = 100000;

// ─── In-memory session key ──────────────────────────────────────────────────

Uint8List? _sessionPrivateKeyD;
Uint8List? _sessionPublicKeyRaw;

bool hasSessionKey() => _sessionPrivateKeyD != null;

Uint8List? getSessionPublicKey() => _sessionPublicKeyRaw;

void clearSessionKey() {
  _sessionPrivateKeyD = null;
  _sessionPublicKeyRaw = null;
}

// ─── Key generation ─────────────────────────────────────────────────────────

/// Generate an ECDH P-256 key pair, wrap the private key with a
/// password-derived AES-GCM key, and persist to SharedPreferences.
///
/// Returns the raw public key bytes (uncompressed point, 65 bytes).
Future<Uint8List> generateKeyPair(String password) async {
  // 1. Generate P-256 key pair via pointycastle
  final keyGen = ECKeyGenerator()
    ..init(ParametersWithRandom(
      ECKeyGeneratorParameters(ECCurve_secp256r1()),
      _secureRandom(),
    ));
  final keyPair = keyGen.generateKeyPair();
  final privKey = keyPair.privateKey as ECPrivateKey;
  final pubKey = keyPair.publicKey as ECPublicKey;

  // 2. Export raw private scalar d (32 bytes)
  final dBytes = _bigIntTo32Bytes(privKey.d!);

  // 3. Export uncompressed public point (04||x||y, 65 bytes)
  final pubBytes = _encodeUncompressedPoint(pubKey.Q!);

  // 4. Derive KEK from password
  final salt = _randomBytes(32);
  final kekIv = _randomBytes(12);
  final kek = _deriveKeyFromPassword(password, salt);

  // 5. Wrap private key with KEK (AES-256-GCM)
  final encryptedPrivateKey = _aes256GcmEncrypt(kek, kekIv, dBytes);

  // 6. Persist to SharedPreferences
  final prefs = await SharedPreferences.getInstance();
  await prefs.setString(_storeEncryptedKey, base64Encode(encryptedPrivateKey));
  await prefs.setString(_storeSalt, base64Encode(salt));
  await prefs.setString(_storeIv, base64Encode(kekIv));
  await prefs.setString(_storePublicKey, base64Encode(pubBytes));

  // 7. Set session keys
  _sessionPrivateKeyD = dBytes;
  _sessionPublicKeyRaw = pubBytes;

  return pubBytes;
}

// ─── Key unlock / import ────────────────────────────────────────────────────

/// Unlock the stored private key using the user's password.
/// On success, the private key is held in memory for the session.
Future<bool> unlockKey(String password) async {
  final prefs = await SharedPreferences.getInstance();
  final encryptedB64 = prefs.getString(_storeEncryptedKey);
  final saltB64 = prefs.getString(_storeSalt);
  final ivB64 = prefs.getString(_storeIv);
  final pubB64 = prefs.getString(_storePublicKey);

  if (encryptedB64 == null || saltB64 == null || ivB64 == null || pubB64 == null) {
    return false;
  }

  try {
    final encryptedPrivateKey = base64Decode(encryptedB64);
    final salt = base64Decode(saltB64);
    final iv = base64Decode(ivB64);
    final pubBytesRaw = base64Decode(pubB64);

    // Normalize keys: handle both raw format (Flutter-native) and
    // SPKI/PKCS#8 DER format (from Web SubtleCrypto cross-platform sync)
    final pubBytes = _normalizePublicKey(pubBytesRaw);

    // Derive KEK from password + stored salt
    final kek = _deriveKeyFromPassword(password, salt);

    // Decrypt private key
    final dBytesRaw = _aes256GcmDecrypt(kek, iv, encryptedPrivateKey);

    // Normalize private key if it came from Web (PKCS#8 DER → raw d)
    final dBytes = _normalizePrivateKey(dBytesRaw);

    _sessionPrivateKeyD = dBytes;
    _sessionPublicKeyRaw = pubBytes;

    return true;
  } catch (_) {
    // Wrong password or corrupt data
    return false;
  }
}

/// Check if keys exist in storage.
Future<bool> hasStoredKey() async {
  final prefs = await SharedPreferences.getInstance();
  return prefs.containsKey(_storeEncryptedKey);
}

/// Delete stored keys and clear session.
Future<void> deleteStoredKey() async {
  clearSessionKey();
  final prefs = await SharedPreferences.getInstance();
  await prefs.remove(_storeEncryptedKey);
  await prefs.remove(_storeSalt);
  await prefs.remove(_storeIv);
  await prefs.remove(_storePublicKey);
}

// ─── Display helpers ────────────────────────────────────────────────────────

String publicKeyToPem(Uint8List publicKeyRaw) {
  final b64 = base64Encode(publicKeyRaw);
  final lines = _chunkString(b64, 64);
  return '-----BEGIN PUBLIC KEY-----\n${lines.join('\n')}\n-----END PUBLIC KEY-----';
}

String publicKeyToHex(Uint8List publicKeyRaw) {
  return publicKeyRaw.map((b) => b.toRadixString(16).padLeft(2, '0')).join('');
}

// ─── Cross-platform key export / import ─────────────────────────────────────

/// Export stored key package as a portable JSON string (base64-encoded values).
/// Returns null if no keys exist.
Future<String?> exportKeyPackage() async {
  final prefs = await SharedPreferences.getInstance();
  final encryptedB64 = prefs.getString(_storeEncryptedKey);
  final saltB64 = prefs.getString(_storeSalt);
  final ivB64 = prefs.getString(_storeIv);
  final pubB64 = prefs.getString(_storePublicKey);

  if (encryptedB64 == null || saltB64 == null || ivB64 == null || pubB64 == null) {
    return null;
  }

  final package = {
    'encryptedPrivateKey': encryptedB64,
    'salt': saltB64,
    'iv': ivB64,
    'publicKeyRaw': pubB64,
  };

  return jsonEncode(package);
}

/// Import a key package exported from another platform (Web/Flutter).
/// Returns true on success.
Future<bool> importKeyPackage(String packageJson) async {
  try {
    final Map<String, dynamic> data = jsonDecode(packageJson);
    final encryptedB64 = data['encryptedPrivateKey'] as String?;
    final saltB64 = data['salt'] as String?;
    final ivB64 = data['iv'] as String?;
    final pubB64 = data['publicKeyRaw'] as String?;

    if (encryptedB64 == null || saltB64 == null || ivB64 == null || pubB64 == null) {
      return false;
    }

    // Validate base64
    base64Decode(encryptedB64);
    base64Decode(saltB64);
    base64Decode(ivB64);
    base64Decode(pubB64);

    // Delete any existing key first
    await deleteStoredKey();

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_storeEncryptedKey, encryptedB64);
    await prefs.setString(_storeSalt, saltB64);
    await prefs.setString(_storeIv, ivB64);
    await prefs.setString(_storePublicKey, pubB64);

    return true;
  } catch (_) {
    return false;
  }
}

// ─── Server sync (cross-platform key synchronization) ───────────────────────

/// Type alias for a simple API caller used by server sync functions.
/// The Flutter app sets this before calling sync functions.
typedef SyncApiCaller = Future<Map<String, dynamic>> Function(
  String method,
  String path, {
  Map<String, dynamic>? data,
});

SyncApiCaller? _syncApiCaller;

/// Set the API caller used by server sync functions.
/// The app should provide a closure wrapping its authenticated ApiClient.
void setSyncApiCaller(SyncApiCaller caller) {
  _syncApiCaller = caller;
}

/// Upload the current local key package to the server for cross-platform sync.
Future<bool> syncToServer() async {
  final caller = _syncApiCaller;
  if (caller == null) return false;

  final prefs = await SharedPreferences.getInstance();
  final encryptedB64 = prefs.getString(_storeEncryptedKey);
  final saltB64 = prefs.getString(_storeSalt);
  final ivB64 = prefs.getString(_storeIv);
  final pubB64 = prefs.getString(_storePublicKey);

  if (encryptedB64 == null || saltB64 == null || ivB64 == null || pubB64 == null) {
    return false;
  }

  try {
    await caller('PUT', '/keys', data: {
      'encrypted_private_key': encryptedB64,
      'salt': saltB64,
      'iv': ivB64,
      'public_key_raw': pubB64,
    });
    return true;
  } catch (_) {
    return false;
  }
}

/// Download the key package from the server and store locally.
/// When [force] is true, overwrites any existing local keys with server keys.
/// When [force] is false (default), skips if local keys already exist.
Future<bool> syncFromServer({bool force = false}) async {
  final caller = _syncApiCaller;
  if (caller == null) return false;

  // By default, don't overwrite existing local keys
  if (!force) {
    final exists = await hasStoredKey();
    if (exists) return false;
  }

  try {
    final result = await caller('GET', '/keys');
    final data = result['data'] as Map<String, dynamic>?;
    if (data == null) return false;

    final encryptedB64 = data['encrypted_private_key'] as String?;
    final saltB64 = data['salt'] as String?;
    final ivB64 = data['iv'] as String?;
    final pubB64 = data['public_key_raw'] as String?;

    if (encryptedB64 == null || saltB64 == null || ivB64 == null || pubB64 == null) {
      return false;
    }

    // Validate base64
    base64Decode(encryptedB64);
    base64Decode(saltB64);
    base64Decode(ivB64);
    base64Decode(pubB64);

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_storeEncryptedKey, encryptedB64);
    await prefs.setString(_storeSalt, saltB64);
    await prefs.setString(_storeIv, ivB64);
    await prefs.setString(_storePublicKey, pubB64);

    return true;
  } catch (_) {
    return false;
  }
}

/// Try to sync keys from server: if no local keys exist, try to download
/// from server. If local keys exist, upload to server so other devices can
/// download them. Never overwrite local keys with server keys in auto-sync.
Future<void> autoSyncKeys() async {
  try {
    final exists = await hasStoredKey();
    if (exists) {
      // Local keys exist — upload to server to seed/update cross-device sync
      await syncToServer();
    } else {
      // No local keys — download from server (if available)
      await syncFromServer();
    }
  } catch (_) {
    // Best-effort
  }
}

/// Delete keys from server.
Future<void> deleteKeysFromServer() async {
  final caller = _syncApiCaller;
  if (caller == null) return;

  try {
    await caller('DELETE', '/keys');
  } catch (_) {
    // Best-effort
  }
}

/// Get compact key package JSON for QR code encoding.
/// Returns null if no keys.
Future<String?> getQRKeyPackage() async {
  final prefs = await SharedPreferences.getInstance();
  final encryptedB64 = prefs.getString(_storeEncryptedKey);
  final saltB64 = prefs.getString(_storeSalt);
  final ivB64 = prefs.getString(_storeIv);
  final pubB64 = prefs.getString(_storePublicKey);

  if (encryptedB64 == null || saltB64 == null || ivB64 == null || pubB64 == null) {
    return null;
  }

  final pkg = {
    'v': 1,
    't': DateTime.now().millisecondsSinceEpoch ~/ 1000,
    'ek': encryptedB64,
    's': saltB64,
    'i': ivB64,
    'pk': pubB64,
  };

  return jsonEncode(pkg);
}

/// Import keys from a compact QR package JSON.
/// Returns true on success.
Future<bool> importQRKeyPackage(String jsonStr) async {
  try {
    final data = jsonDecode(jsonStr) as Map<String, dynamic>;
    final ek = data['ek'] as String?;
    final s = data['s'] as String?;
    final i = data['i'] as String?;
    final pk = data['pk'] as String?;

    if (ek == null || s == null || i == null || pk == null) return false;

    // Validate base64
    base64Decode(ek);
    base64Decode(s);
    base64Decode(i);
    base64Decode(pk);

    // Delete existing keys
    await deleteStoredKey();

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_storeEncryptedKey, ek);
    await prefs.setString(_storeSalt, s);
    await prefs.setString(_storeIv, i);
    await prefs.setString(_storePublicKey, pk);

    return true;
  } catch (_) {
    return false;
  }
}

// ─── Note encryption / decryption ───────────────────────────────────────────

/// Encrypt note content.
/// Content is stored as: base64(salt):base64(iv):base64(ciphertext)
/// Returns the encrypted string to be stored as the note's content on the server.
Future<String> encryptNoteContent(String content) async {
  if (_sessionPrivateKeyD == null || _sessionPublicKeyRaw == null) {
    throw Exception('Encryption key not unlocked. Please unlock your key first.');
  }

  // 1. ECDH derive shared secret (static DH using own private + own public)
  final sharedSecret = _ecdhDerive(_sessionPrivateKeyD!, _sessionPublicKeyRaw!);

  // 2. Derive per-note AES key using PBKDF2 + random salt
  final noteSalt = _randomBytes(16);
  final noteKey = _deriveKeyFromSecret(sharedSecret, noteSalt);
  final noteIv = _randomBytes(12);

  // 3. Encrypt content
  final contentBytes = Uint8List.fromList(utf8.encode(content));
  final ciphertext = _aes256GcmEncrypt(noteKey, noteIv, contentBytes);

  // 4. Format: base64(salt):base64(iv):base64(ciphertext)
  return '${base64Encode(noteSalt)}:${base64Encode(noteIv)}:${base64Encode(ciphertext)}';
}

/// Decrypt note content that was encrypted with encryptNoteContent().
Future<String> decryptNoteContent(String encrypted) async {
  if (_sessionPrivateKeyD == null || _sessionPublicKeyRaw == null) {
    throw Exception('Encryption key not unlocked. Please unlock your key first.');
  }

  // 1. Parse parts
  final parts = encrypted.split(':');
  if (parts.length != 3) {
    throw FormatException('Invalid encrypted note format');
  }

  final noteSalt = base64Decode(parts[0]);
  final noteIv = base64Decode(parts[1]);
  final ciphertext = base64Decode(parts[2]);

  // 2. ECDH derive shared secret
  final sharedSecret = _ecdhDerive(_sessionPrivateKeyD!, _sessionPublicKeyRaw!);

  // 3. Derive per-note AES key
  final noteKey = _deriveKeyFromSecret(sharedSecret, noteSalt);

  // 4. Decrypt
  final plaintext = _aes256GcmDecrypt(noteKey, noteIv, ciphertext);

  return utf8.decode(plaintext);
}

/// Check if a string is encrypted content (format: base64(salt):base64(iv):base64(ciphertext)).
bool isEncryptedContent(String content) {
  if (content.isEmpty) return false;
  final parts = content.split(':');
  if (parts.length != 3) return false;
  try {
    base64Decode(parts[0]);
    base64Decode(parts[1]);
    base64Decode(parts[2]);
    return true;
  } catch (_) {
    return false;
  }
}

// ─── Internal crypto helpers ────────────────────────────────────────────────

/// Build a secure random instance for pointycastle.
SecureRandom _secureRandom() {
  final random = Random.secure();
  final seed = Uint8List.fromList(List<int>.generate(32, (_) => random.nextInt(256)));
  final fortuna = FortunaRandom();
  fortuna.seed(KeyParameter(seed));
  return fortuna;
}

/// Generate cryptographically random bytes.
Uint8List _randomBytes(int length) {
  final random = Random.secure();
  final bytes = Uint8List(length);
  for (int i = 0; i < length; i++) bytes[i] = random.nextInt(256);
  return bytes;
}

/// Encode an EC point as uncompressed 04||x||y (65 bytes).
Uint8List _encodeUncompressedPoint(ECPoint point) {
  final x = _bigIntTo32Bytes(point.x!.toBigInteger()!);
  final y = _bigIntTo32Bytes(point.y!.toBigInteger()!);
  final result = Uint8List(65);
  result[0] = 0x04;
  result.setRange(1, 33, x);
  result.setRange(33, 65, y);
  return result;
}

/// Convert a BigInt to 32-byte big-endian array.
Uint8List _bigIntTo32Bytes(BigInt n) {
  // BigInt in Dart doesn't have toByteArray, we convert manually
  int byteCount = (n.bitLength + 7) ~/ 8;
  final result = Uint8List(32);
  for (int i = 0; i < byteCount && i < 32; i++) {
    result[31 - i] = (n >> (8 * i)).toUnsigned(64).toInt() & 0xff;
  }
  return result;
}

/// ECDH key agreement: shared = privateD * publicPoint.
/// Returns X coordinate as 32 bytes.
Uint8List _ecdhDerive(Uint8List privateKeyD, Uint8List publicKeyRaw) {
  final curve = ECCurve_secp256r1();
  final d = _uint8ListToBigInt(privateKeyD);
  final x = _uint8ListToBigInt(publicKeyRaw.sublist(1, 33));
  final y = _uint8ListToBigInt(publicKeyRaw.sublist(33, 65));
  final point = curve.curve.createPoint(x, y);

  final shared = (point * d);
  return _bigIntTo32Bytes(shared!.x!.toBigInteger()!);
}

BigInt _uint8ListToBigInt(Uint8List bytes) {
  BigInt result = BigInt.zero;
  for (int i = 0; i < bytes.length; i++) {
    result = (result << 8) | BigInt.from(bytes[i]);
  }
  return result;
}

/// PBKDF2-HMAC-SHA256 key derivation (for KEK from password).
KeyParameter _deriveKeyFromPassword(String password, Uint8List salt) {
  final pbkdf2 = PBKDF2KeyDerivator(HMac(SHA256Digest(), 64));
  pbkdf2.init(Pbkdf2Parameters(salt, _pbkdf2Iterations, 32)); // 32 bytes = 256 bits
  final derived = pbkdf2.process(Uint8List.fromList(utf8.encode(password)));
  return KeyParameter(derived);
}

/// PBKDF2-HMAC-SHA256 key derivation (for per-note key from shared secret).
KeyParameter _deriveKeyFromSecret(Uint8List secret, Uint8List salt) {
  final pbkdf2 = PBKDF2KeyDerivator(HMac(SHA256Digest(), 64));
  pbkdf2.init(Pbkdf2Parameters(salt, _notePbkdf2Iterations, 32));
  final derived = pbkdf2.process(secret);
  return KeyParameter(derived);
}

/// AES-256-GCM encrypt. Returns ciphertext || tag.
Uint8List _aes256GcmEncrypt(KeyParameter key, Uint8List iv, Uint8List plaintext) {
  final cipher = GCMBlockCipher(AESEngine())
    ..init(
      true,
      AEADParameters(key, 128, iv, Uint8List(0)),
    );

  final out = Uint8List(cipher.getOutputSize(plaintext.length));
  final len1 = cipher.processBytes(plaintext, 0, plaintext.length, out, 0);
  final len2 = cipher.doFinal(out, len1);
  return Uint8List.sublistView(out, 0, len1 + len2);
}

/// AES-256-GCM decrypt.
Uint8List _aes256GcmDecrypt(KeyParameter key, Uint8List iv, Uint8List ciphertextWithTag) {
  final cipher = GCMBlockCipher(AESEngine())
    ..init(
      false,
      AEADParameters(key, 128, iv, Uint8List(0)),
    );

  final out = Uint8List(cipher.getOutputSize(ciphertextWithTag.length));
  final len1 = cipher.processBytes(ciphertextWithTag, 0, ciphertextWithTag.length, out, 0);
  final len2 = cipher.doFinal(out, len1);
  return Uint8List.sublistView(out, 0, len1 + len2);
}

// ─── Cross-platform key format normalization ────────────────────────────────
//
// Web SubtleCrypto exports keys in SPKI/PKCS#8 DER format, while Flutter
// stores raw uncompressed point (65 bytes) / raw d scalar (32 bytes).
// These helpers detect and normalize between the two formats.

/// DER length decoding: returns the length value.
int _derLengthDecode(Uint8List data, int pos) {
  if (data[pos] < 0x80) return data[pos];
  int numBytes = data[pos] & 0x7f;
  int length = 0;
  for (int i = 1; i <= numBytes; i++) {
    length = (length << 8) | data[pos + i];
  }
  return length;
}

/// Returns the number of bytes consumed by the DER length field starting at [pos].
int _derLengthSize(Uint8List data, int pos) {
  if (data[pos] < 0x80) return 1;
  return 1 + (data[pos] & 0x7f);
}

/// Skip a full DER TLV (tag + length + value) from position [pos].
/// Returns the position immediately after the value.
int _derSkipTLV(Uint8List data, int pos) {
  pos++; // skip tag
  if (data[pos] < 0x80) {
    // Short form: length fits in one byte
    int len = data[pos];
    return pos + 1 + len;
  } else {
    // Long form
    int numLenBytes = data[pos] & 0x7f;
    int len = 0;
    for (int j = 1; j <= numLenBytes; j++) {
      len = (len << 8) | data[pos + j];
    }
    return pos + 1 + numLenBytes + len;
  }
}

/// Normalize a public key from SPKI DER to raw uncompressed point (04||x||y).
/// If it's already raw format (starts with 0x04, 65 bytes), returns as-is.
Uint8List _normalizePublicKey(Uint8List raw) {
  // Already raw uncompressed EC point
  if (raw.length == 65 && raw[0] == 0x04) return raw;

  // Must be SPKI DER — extract the BIT STRING's EC point bytes
  int pos = 0;

  // Outer SEQUENCE
  if (raw[pos] != 0x30) return raw; // unknown format, return as-is
  pos++; // skip SEQUENCE tag
  pos += _derLengthSize(raw, pos); // skip SEQUENCE length

  // AlgorithmIdentifier SEQUENCE — skip entire TLV
  if (pos >= raw.length || raw[pos] != 0x30) return raw;
  pos = _derSkipTLV(raw, pos);

  // BIT STRING
  if (pos >= raw.length || raw[pos] != 0x03) return raw;
  pos++; // skip BIT STRING tag
  pos += _derLengthSize(raw, pos); // skip BIT STRING length

  // Unused bits byte
  if (pos >= raw.length || raw[pos] != 0x00) return raw; // must be 0 unused bits
  pos++;

  // Remaining bytes are the raw EC point (04||x||y, 65 bytes)
  if (pos + 65 > raw.length) return raw; // sanity check
  return raw.sublist(pos, pos + 65);
}

/// Normalize a private key from PKCS#8 DER to raw d scalar (32 bytes).
/// If it's already raw format (32 bytes), returns as-is.
Uint8List _normalizePrivateKey(Uint8List raw) {
  // Already raw d scalar
  if (raw.length == 32) return raw;

  // Must be PKCS#8 DER — extract the private key OCTET STRING
  int i = 0;
  if (raw[i] != 0x30) return raw; // unknown format, return as-is
  i++;
  i += _derLengthSize(raw, i);

  // Skip version INTEGER
  if (raw[i] != 0x02) return raw;
  i++;
  i += _derLengthSize(raw, i);

  // Skip AlgorithmIdentifier SEQUENCE
  if (raw[i] != 0x30) return raw;
  i++;
  i += _derLengthSize(raw, i);

  // OCTET STRING wrapping ECPrivateKey
  if (raw[i] != 0x04) return raw;
  i++;
  i += _derLengthSize(raw, i);

  // ECPrivateKey SEQUENCE
  if (raw[i] != 0x30) return raw;
  i++;
  i += _derLengthSize(raw, i);

  // Skip version INTEGER (1)
  if (raw[i] != 0x02) return raw;
  i++;
  i += _derLengthSize(raw, i);

  // OCTET STRING containing d
  if (raw[i] != 0x04) return raw;
  i++;
  int dLen = _derLengthDecode(raw, i);
  i += dLen >= 128 ? 3 : 1;

  return raw.sublist(i, i + dLen);
}

List<String> _chunkString(String str, int chunkSize) {
  final chunks = <String>[];
  for (int i = 0; i < str.length; i += chunkSize) {
    chunks.add(str.substring(i, (i + chunkSize > str.length) ? str.length : i + chunkSize));
  }
  return chunks;
}
