import apiClient from './client'
import type { ApiResponse } from './types'

export interface KeyPackage {
  encrypted_private_key: string
  salt: string
  iv: string
  public_key_raw: string
  created_at?: string
  updated_at?: string | null
}

export interface QRKeyPackage {
  v: number
  t: number
  ek: string
  s: string
  i: string
  pk: string
}

/**
 * Upload/sync the encrypted key package to the server.
 */
export async function syncKeyToServer(pkg: {
  encrypted_private_key: string
  salt: string
  iv: string
  public_key_raw: string
}): Promise<void> {
  await apiClient.put<ApiResponse>('/keys', pkg)
}

/**
 * Fetch the user's encrypted key package from the server.
 */
export async function fetchKeyFromServer(): Promise<KeyPackage | null> {
  const res = await apiClient.get<ApiResponse<KeyPackage>>('/keys')
  return res.data.data ?? null
}

/**
 * Delete the user's key from the server.
 */
export async function deleteKeyFromServer(): Promise<void> {
  await apiClient.delete<ApiResponse>('/keys')
}

/**
 * Get the user's key package in compact format for QR code.
 */
export async function getQRKeyPackage(): Promise<QRKeyPackage | null> {
  const res = await apiClient.get<ApiResponse<QRKeyPackage>>('/keys/qr')
  return res.data.data ?? null
}
