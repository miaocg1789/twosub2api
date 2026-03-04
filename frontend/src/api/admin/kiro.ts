/**
 * Admin Kiro API endpoints
 * Handles Kiro (AWS CodeWhisperer) refresh token authentication
 */

import { apiClient } from '../client'

export interface KiroRefreshTokenRequest {
  refresh_token: string
  auth_type?: string
  client_id?: string
  client_secret?: string
  region?: string
  proxy_id?: number | null
}

export interface KiroTokenInfo {
  access_token?: string
  refresh_token?: string
  expires_at?: number | string
  auth_type?: string
  [key: string]: unknown
}

export async function refreshKiroToken(
  payload: KiroRefreshTokenRequest
): Promise<KiroTokenInfo> {
  const { data } = await apiClient.post<KiroTokenInfo>(
    '/admin/kiro/oauth/refresh-token',
    payload
  )
  return data
}

export default { refreshKiroToken }
