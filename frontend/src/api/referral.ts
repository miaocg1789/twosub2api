/**
 * Referral API endpoints
 * Handles referral/affiliate system for users
 */

import { apiClient } from './client'
import type { ReferralStats, ReferralInvitee, ReferralCommission } from '@/types'

/**
 * Get referral statistics
 */
export async function getReferralStats(): Promise<ReferralStats> {
  const { data } = await apiClient.get<ReferralStats>('/referral/stats')
  return data
}

/**
 * Get invited users list (paginated)
 */
export async function getInvitees(params: {
  page?: number
  page_size?: number
}): Promise<{ items: ReferralInvitee[]; total: number; page: number; page_size: number }> {
  const { data } = await apiClient.get('/referral/invitees', { params })
  return data as { items: ReferralInvitee[]; total: number; page: number; page_size: number }
}

/**
 * Get commission history (paginated)
 */
export async function getCommissions(params: {
  page?: number
  page_size?: number
}): Promise<{ items: ReferralCommission[]; total: number; page: number; page_size: number }> {
  const { data } = await apiClient.get('/referral/commissions', { params })
  return data as { items: ReferralCommission[]; total: number; page: number; page_size: number }
}

export const referralAPI = {
  getReferralStats,
  getInvitees,
  getCommissions
}

export default referralAPI
