/**
 * Admin Kimi API endpoints
 * Handles Kimi OAuth device flow for administrators
 */

import { apiClient } from '../client'

export interface KimiDeviceFlowResponse {
  session_id: string
  user_code: string
  verification_url: string
  verification_url_complete: string
  expires_in: number
  interval: number
}

export interface KimiDeviceFlowRequest {
  proxy_id?: number
}

export interface KimiTokenInfo {
  access_token?: string
  refresh_token?: string
  token_type?: string
  scope?: string
  expires_in?: number
  expires_at?: number | string
  device_id?: string
  [key: string]: unknown
}

export async function initiateDeviceFlow(
  payload: KimiDeviceFlowRequest
): Promise<KimiDeviceFlowResponse> {
  const { data } = await apiClient.post<KimiDeviceFlowResponse>(
    '/admin/kimi/oauth/device-flow',
    payload
  )
  return data
}

export async function pollToken(sessionId: string): Promise<KimiTokenInfo> {
  const { data } = await apiClient.post<KimiTokenInfo>('/admin/kimi/oauth/poll-token', {
    session_id: sessionId
  })
  return data
}

export async function refreshKimiToken(
  refreshToken: string,
  proxyId?: number | null
): Promise<KimiTokenInfo> {
  const payload: Record<string, any> = { refresh_token: refreshToken }
  if (proxyId) payload.proxy_id = proxyId
  const { data } = await apiClient.post<KimiTokenInfo>('/admin/kimi/oauth/refresh-token', payload)
  return data
}

export default { initiateDeviceFlow, pollToken, refreshKimiToken }
