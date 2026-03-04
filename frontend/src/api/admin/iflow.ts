/**
 * Admin iFlow API endpoints
 * Handles iFlow cookie-based authentication for administrators
 */

import { apiClient } from '../client'

export interface IFlowTokenInfo {
  api_key?: string
  cookie?: string
  expires_at?: number | string
  email?: string
  [key: string]: unknown
}

export async function authenticateWithCookie(
  cookie: string,
  proxyId?: number | null
): Promise<IFlowTokenInfo> {
  const payload: Record<string, any> = { cookie }
  if (proxyId) payload.proxy_id = proxyId
  const { data } = await apiClient.post<IFlowTokenInfo>('/admin/iflow/oauth/cookie-auth', payload)
  return data
}

export async function refreshAPIKey(
  cookie: string,
  proxyId?: number | null
): Promise<IFlowTokenInfo> {
  const payload: Record<string, any> = { cookie }
  if (proxyId) payload.proxy_id = proxyId
  const { data } = await apiClient.post<IFlowTokenInfo>('/admin/iflow/oauth/refresh-key', payload)
  return data
}

export default { authenticateWithCookie, refreshAPIKey }
