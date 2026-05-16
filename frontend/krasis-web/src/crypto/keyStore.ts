/**
 * High-security ECC key management and note encryption/decryption.
 *
 * Key hierarchy:
 *   1. User password (never stored)
 *   2. PBKDF2(password + salt) → Key Encryption Key (KEK)
 *   3. KEK wraps ECDH private key via AES-GCM
 *   4. ECDH P-256 key pair for note encryption
 *   5. ECDH deriveBits → PBKDF2(salt') → per-note AES-256-GCM key
 *
 * Storage:
 *   - IndexedDB stores: encryptedPrivateKey, salt, iv, publicKey (raw)
 *   - In-memory during session: decrypted CryptoKey (private)
 *   - Note content stored as "salt:iv:ciphertext" (base64, colon-separated)
 */

// ─── IndexedDB helpers ──────────────────────────────────────────────────────

const DB_NAME = 'krasis-crypto'
const DB_VERSION = 1
const STORE_NAME = 'keys'

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION)
    req.onupgradeneeded = () => {
      const db = req.result
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: 'id' })
      }
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

interface KeyRecord {
  id: string
  encryptedPrivateKey: ArrayBuffer
  salt: Uint8Array    // PBKDF2 salt for KEK
  iv: Uint8Array      // AES-GCM iv for private key wrapping
  publicKeyRaw: Uint8Array
}

async function storeKeyRecord(record: KeyRecord): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, 'readwrite')
    tx.objectStore(STORE_NAME).put(record)
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

async function getKeyRecord(): Promise<KeyRecord | null> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, 'readonly')
    const req = tx.objectStore(STORE_NAME).get('default')
    req.onsuccess = () => resolve(req.result ?? null)
    req.onerror = () => reject(req.error)
  })
}

async function deleteKeyRecord(): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, 'readwrite')
    tx.objectStore(STORE_NAME).delete('default')
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

// ─── In-memory session key ──────────────────────────────────────────────────

let sessionPrivateKey: CryptoKey | null = null
let sessionPublicKeyRaw: Uint8Array | null = null

export function hasSessionKey(): boolean {
  return sessionPrivateKey !== null
}

export function getSessionPublicKey(): Uint8Array | null {
  return sessionPublicKeyRaw
}

export function clearSessionKey(): void {
  sessionPrivateKey = null
  sessionPublicKeyRaw = null
}

// ─── Key generation ─────────────────────────────────────────────────────────

/**
 * Generate an ECDH P-256 key pair, wrap the private key with a
 * password-derived AES-GCM key, and persist to IndexedDB.
 *
 * Returns the raw public key (SPKI format) so it can be displayed/exported.
 */
export async function generateKeyPair(password: string): Promise<Uint8Array> {
  // 1. Generate ECDH P-256 key pair
  const keyPair = await crypto.subtle.generateKey(
    { name: 'ECDH', namedCurve: 'P-256' },
    true,
    ['deriveKey', 'deriveBits'],
  )

  // 2. Export private key raw
  const privateKeyRaw = await crypto.subtle.exportKey('pkcs8', keyPair.privateKey)
  const publicKeyRaw = await crypto.subtle.exportKey('spki', keyPair.publicKey)

  // 3. Derive KEK from password
  const salt = crypto.getRandomValues(new Uint8Array(32))
  const kekIv = crypto.getRandomValues(new Uint8Array(12))
  const kek = await deriveKeyFromPassword(password, salt, 'AES-GCM', ['encrypt', 'decrypt'])

  // 4. Wrap private key with KEK
  const encryptedPrivateKey = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: kekIv },
    kek,
    privateKeyRaw,
  )

  // 5. Store in IndexedDB
  await storeKeyRecord({
    id: 'default',
    encryptedPrivateKey,
    salt,
    iv: kekIv,
    publicKeyRaw: new Uint8Array(publicKeyRaw),
  })

  sessionPrivateKey = keyPair.privateKey
  sessionPublicKeyRaw = new Uint8Array(publicKeyRaw)

  return new Uint8Array(publicKeyRaw)
}

// ─── Key unlock / import ────────────────────────────────────────────────────

/**
 * Unlock the stored private key using the user's password.
 * On success, the private key is held in memory for the session.
 */
export async function unlockKey(password: string): Promise<boolean> {
  const record = await getKeyRecord()
  if (!record) return false

  try {
    // Derive KEK from password + stored salt
    const kek = await deriveKeyFromPassword(password, record.salt, 'AES-GCM', ['decrypt'])

    // Decrypt private key raw bytes
    const privateKeyRaw = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: record.iv },
      kek,
      record.encryptedPrivateKey,
    )

    // Import back as CryptoKey
    sessionPrivateKey = await crypto.subtle.importKey(
      'pkcs8',
      privateKeyRaw,
      { name: 'ECDH', namedCurve: 'P-256' },
      false,
      ['deriveBits'],
    )

    sessionPublicKeyRaw = record.publicKeyRaw

    return true
  } catch {
    // Wrong password or corrupt data
    return false
  }
}

/**
 * Check if a key exists in storage (i.e. user has generated keys).
 */
export async function hasStoredKey(): Promise<boolean> {
  const record = await getKeyRecord()
  return record !== null
}

/**
 * Delete stored keys and clear session.
 */
export async function deleteStoredKey(): Promise<void> {
  clearSessionKey()
  await deleteKeyRecord()
}

// ─── Cross-platform key export / import ─────────────────────────────────────

/**
 * Export stored key package as a portable JSON string (base64-encoded values).
 * Returns null if no keys exist.
 */
export async function exportKeyPackage(): Promise<string | null> {
  const record = await getKeyRecord()
  if (!record) return null

  const pkg = {
    encryptedPrivateKey: arrayBufferToBase64(record.encryptedPrivateKey),
    salt: uint8ArrayToBase64(record.salt),
    iv: uint8ArrayToBase64(record.iv),
    publicKeyRaw: uint8ArrayToBase64(record.publicKeyRaw),
  }

  return JSON.stringify(pkg)
}

/**
 * Import a key package exported from another platform (Web/Flutter).
 * Returns true on success.
 */
export async function importKeyPackage(packageJson: string): Promise<boolean> {
  try {
    const data = JSON.parse(packageJson)
    const { encryptedPrivateKey, salt, iv, publicKeyRaw } = data as Record<string, string>

    if (!encryptedPrivateKey || !salt || !iv || !publicKeyRaw) {
      return false
    }

    // Validate base64
    atob(encryptedPrivateKey)
    atob(salt)
    atob(iv)
    atob(publicKeyRaw)

    // Delete any existing key first
    clearSessionKey()
    await deleteKeyRecord()

    await storeKeyRecord({
      id: 'default',
      encryptedPrivateKey: base64ToArrayBuffer(encryptedPrivateKey),
      salt: base64ToUint8Array(salt),
      iv: base64ToUint8Array(iv),
      publicKeyRaw: base64ToUint8Array(publicKeyRaw),
    })

    return true
  } catch {
    return false
  }
}

// ─── Server sync (cross-platform key synchronization) ───────────────────────

/**
 * Upload the current local key package to the server for cross-platform sync.
 * Called automatically after key generation/unlock/import.
 */
export async function syncToServer(): Promise<boolean> {
  const record = await getKeyRecord()
  if (!record) return false

  try {
    const { syncKeyToServer } = await import('../api/keys')
    await syncKeyToServer({
      encrypted_private_key: arrayBufferToBase64(record.encryptedPrivateKey),
      salt: uint8ArrayToBase64(record.salt),
      iv: uint8ArrayToBase64(record.iv),
      public_key_raw: uint8ArrayToBase64(record.publicKeyRaw),
    })
    return true
  } catch {
    return false
  }
}

/**
 * Download the key package from the server and store locally.
 * Skips if local keys already exist to prevent accidental overwrite.
 * Returns true if keys were imported.
 */
export async function syncFromServer(): Promise<boolean> {
  // Don't overwrite existing local keys
  const local = await getKeyRecord()
  if (local) return false

  try {
    const { fetchKeyFromServer } = await import('../api/keys')
    const pkg = await fetchKeyFromServer()
    if (!pkg) return false

    // Validate base64
    atob(pkg.encrypted_private_key)
    atob(pkg.salt)
    atob(pkg.iv)
    atob(pkg.public_key_raw)

    await storeKeyRecord({
      id: 'default',
      encryptedPrivateKey: base64ToArrayBuffer(pkg.encrypted_private_key),
      salt: base64ToUint8Array(pkg.salt),
      iv: base64ToUint8Array(pkg.iv),
      publicKeyRaw: base64ToUint8Array(pkg.public_key_raw),
    })

    return true
  } catch {
    return false
  }
}

/**
 * Try to sync keys from server; if local keys exist, upload to server for
 * cross-device sync. If no local keys, download from server.
 * Never overwrite local keys with server keys in auto-sync mode.
 */
export async function autoSyncKeys(): Promise<void> {
  try {
    const local = await getKeyRecord()
    if (local) {
      // Upload local keys to server so other devices can download them
      await syncToServer()
    } else {
      // No local keys — download from server if available
      await syncFromServer()
    }
  } catch {
    // Silently fail — sync is best-effort
  }
}

/**
 * Get compact key package for QR code encoding.
 * Returns the compact JSON to encode into QR, or null if no keys.
 */
export async function getQRKeyPackage(): Promise<string | null> {
  const record = await getKeyRecord()
  if (!record) return null

  const pkg = {
    v: 1,
    t: Math.floor(Date.now() / 1000),
    ek: arrayBufferToBase64(record.encryptedPrivateKey),
    s: uint8ArrayToBase64(record.salt),
    i: uint8ArrayToBase64(record.iv),
    pk: uint8ArrayToBase64(record.publicKeyRaw),
  }

  return JSON.stringify(pkg)
}

// ─── Note encryption / decryption ───────────────────────────────────────────

const NOTE_KEY_ALGO = 'AES-GCM'
const NOTE_KEY_LENGTH = 256

/**
 * Encrypt note content.
 * Content is stored as: base64(salt):base64(iv):base64(ciphertext)
 * Returns the encrypted string to be stored as the note's content on the server.
 */
export async function encryptNoteContent(content: string): Promise<string> {
  if (!sessionPrivateKey) {
    throw new Error('Encryption key not unlocked. Please unlock your key first.')
  }

  // 1. ECDH derive shared secret (own public → own private = static DH)
  const publicKey = await crypto.subtle.importKey(
    'spki',
    sessionPublicKeyRaw!,
    { name: 'ECDH', namedCurve: 'P-256' },
    false,
    [],
  )

  const sharedSecret = await crypto.subtle.deriveBits(
    { name: 'ECDH', public: publicKey },
    sessionPrivateKey,
    256,
  )

  // 2. Derive per-note AES key using PBKDF2 + random salt
  const noteSalt = crypto.getRandomValues(new Uint8Array(16))
  const noteKey = await deriveKeyFromSecret(sharedSecret, noteSalt, NOTE_KEY_LENGTH, NOTE_KEY_ALGO, ['encrypt'])

  // 3. Encrypt content
  const iv = crypto.getRandomValues(new Uint8Array(12))
  const encoder = new TextEncoder()
  const ciphertext = await crypto.subtle.encrypt(
    { name: NOTE_KEY_ALGO, iv },
    noteKey,
    encoder.encode(content),
  )

  // 4. Format: base64(salt):base64(iv):base64(ciphertext)
  return `${uint8ArrayToBase64(noteSalt)}:${uint8ArrayToBase64(iv)}:${arrayBufferToBase64(ciphertext)}`
}

/**
 * Decrypt note content that was encrypted with encryptNoteContent().
 */
export async function decryptNoteContent(encrypted: string): Promise<string> {
  if (!sessionPrivateKey) {
    throw new Error('Encryption key not unlocked. Please unlock your key first.')
  }

  // 1. Parse parts
  const parts = encrypted.split(':')
  if (parts.length !== 3) {
    throw new Error('Invalid encrypted note format')
  }
  const [saltB64, ivB64, ciphertextB64] = parts

  // 2. ECDH derive shared secret
  const publicKey = await crypto.subtle.importKey(
    'spki',
    sessionPublicKeyRaw!,
    { name: 'ECDH', namedCurve: 'P-256' },
    false,
    [],
  )

  const sharedSecret = await crypto.subtle.deriveBits(
    { name: 'ECDH', public: publicKey },
    sessionPrivateKey,
    256,
  )

  // 3. Derive same per-note AES key
  const noteSalt = base64ToUint8Array(saltB64)
  const noteKey = await deriveKeyFromSecret(sharedSecret, noteSalt, NOTE_KEY_LENGTH, NOTE_KEY_ALGO, ['decrypt'])

  // 4. Decrypt
  const iv = base64ToUint8Array(ivB64)
  const ciphertext = base64ToArrayBuffer(ciphertextB64)
  const plaintext = await crypto.subtle.decrypt(
    { name: NOTE_KEY_ALGO, iv },
    noteKey,
    ciphertext,
  )

  const decoder = new TextDecoder()
  return decoder.decode(plaintext)
}

export function isEncryptedContent(content: string): boolean {
  // Format: base64(salt):base64(iv):base64(ciphertext)
  // salt=16B → 24 base64 chars, iv=12B → 16 base64 chars
  if (!content || typeof content !== 'string') return false
  const parts = content.split(':')
  if (parts.length !== 3) return false
  const [salt, iv, ct] = parts
  if (salt.length < 20 || iv.length < 12 || ct.length < 10) return false
  // All parts should be valid base64
  try {
    base64ToUint8Array(salt)
    base64ToUint8Array(iv)
    base64ToArrayBuffer(ct)
    return true
  } catch {
    return false
  }
}

// ─── Utility functions ──────────────────────────────────────────────────────

const PW_ALGO = 'PBKDF2'
const PW_ITERATIONS = 600000 // OWASP recommended for PBKDF2-HMAC-SHA256

async function deriveKeyFromPassword(
  password: string,
  salt: Uint8Array,
  algo: string,
  usages: KeyUsage[],
): Promise<CryptoKey> {
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    PW_ALGO,
    false,
    ['deriveKey'],
  )

  return crypto.subtle.deriveKey(
    {
      name: PW_ALGO,
      salt,
      iterations: PW_ITERATIONS,
      hash: 'SHA-256',
    },
    keyMaterial,
    { name: algo, length: 256 },
    false,
    usages,
  )
}

async function deriveKeyFromSecret(
  secret: ArrayBuffer,
  salt: Uint8Array,
  keyLength: number,
  algo: string,
  usages: KeyUsage[],
): Promise<CryptoKey> {
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    secret,
    PW_ALGO,
    false,
    ['deriveKey'],
  )

  return crypto.subtle.deriveKey(
    {
      name: PW_ALGO,
      salt,
      iterations: 100000,
      hash: 'SHA-256',
    },
    keyMaterial,
    { name: algo, length: keyLength },
    false,
    usages,
  )
}

function uint8ArrayToBase64(arr: Uint8Array): string {
  return btoa(String.fromCharCode(...arr))
}

function arrayBufferToBase64(buf: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buf)))
}

function base64ToUint8Array(b64: string): Uint8Array {
  const binary = atob(b64)
  const arr = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    arr[i] = binary.charCodeAt(i)
  }
  return arr
}

function base64ToArrayBuffer(b64: string): ArrayBuffer {
  return base64ToUint8Array(b64).buffer
}

export function publicKeyToPem(publicKeyRaw: Uint8Array): string {
  const b64 = uint8ArrayToBase64(publicKeyRaw)
  const lines = b64.match(/.{1,64}/g) || []
  return `-----BEGIN PUBLIC KEY-----\n${lines.join('\n')}\n-----END PUBLIC KEY-----`
}

export function publicKeyToHex(publicKeyRaw: Uint8Array): string {
  return Array.from(publicKeyRaw)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}
