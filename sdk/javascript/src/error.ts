export class SDKError extends Error {
  code: number;
  status: number;
  data?: unknown;

  constructor(message: string, code: number, status: number, data?: unknown) {
    super(message);
    this.name = 'SDKError';
    this.code = code;
    this.status = status;
    this.data = data;
  }
}

export class VersionConflictError extends SDKError {
  currentVersion: number;
  note: unknown;

  constructor(currentVersion: number, note: unknown) {
    super('版本冲突', 1005, 409, { currentVersion, note });
    this.name = 'VersionConflictError';
    this.currentVersion = currentVersion;
    this.note = note;
  }
}

export class RateLimitError extends SDKError {
  retryAfter: number;

  constructor(retryAfter: number) {
    super('请求过于频繁', 2001, 429, { retryAfter });
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
  }
}
