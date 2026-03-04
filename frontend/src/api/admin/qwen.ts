/**
 * Admin Qwen API endpoints
 * Handles Qwen OAuth device flow for administrators
 */

import { apiClient } from '../client'

export interface QwenDeviceFlowResponse {
  session_id: string
  user_code: string
  verification_url: string
  verification_url_complete: string
  expires_in: number
  interval: number
}

export interface QwenDeviceFlowRequest {
  proxy_id?: number
}

export interface QwenTokenInfo {
  access_token?: string
  refresh_token?: string
  token_type?: string
  expires_in?: number
  expires_at?: number | string
  resource_url?: string
  [key: string]: unknown
}

export async function initiateDeviceFlow(
  payload: QwenDeviceFlowRequest
): Promise<QwenDeviceFlowResponse> {
  const { data } = await apiClient.post<QwenDeviceFlowResponse>(
    '/admin/qwen/oauth/device-flow',
    payload
  )
  return data
}

export async function pollToken(sessionId: string): Promise<QwenTokenInfo> {
  const { data } = await apiClient.post<QwenTokenInfo>('/admin/qwen/oauth/poll-token', {
    session_id: sessionId
  })
  return data
}

export async function refreshQwenToken(
  refreshToken: string,
  proxyId?: number | null
): Promise<QwenTokenInfo> {
  const payload: Record<string, any> = { refresh_token: refreshToken }
  if (proxyId) payload.proxy_id = proxyId

  const { data } = await apiClient.post<QwenTokenInfo>('/admin/qwen/oauth/refresh-token', payload)
  return data
}

export default { initiateDeviceFlow, pollToken, refreshQwenToken }
