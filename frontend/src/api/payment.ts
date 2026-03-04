/**
 * Payment API endpoints
 * Handles online payment/topup for users
 */

import { apiClient } from './client'
import type { PaymentConfig, CreatePaymentRequest, CreatePaymentResponse } from '@/types'

/**
 * Get payment configuration (available methods, presets, etc.)
 */
export async function getPaymentConfig(): Promise<PaymentConfig> {
  const { data } = await apiClient.get<PaymentConfig>('/payment/config')
  return data
}

/**
 * Create a payment order
 */
export async function createOrder(request: CreatePaymentRequest): Promise<CreatePaymentResponse> {
  const { data } = await apiClient.post<CreatePaymentResponse>('/payment/create', request)
  return data
}

/**
 * Get order status by order ID (for polling)
 */
export async function getOrderStatus(orderId: number): Promise<{ status: string }> {
  const { data } = await apiClient.get<{ status: string }>(`/payment/orders/${orderId}/status`)
  return data
}

/**
 * Get user's payment order history
 */
export async function getOrders(params?: {
  page?: number
  page_size?: number
}): Promise<{ orders: any[]; total: number; page: number }> {
  const { data } = await apiClient.get<{ orders: any[]; total: number; page: number }>('/payment/orders', { params })
  return data
}

export const paymentAPI = {
  getPaymentConfig,
  createOrder,
  getOrderStatus,
  getOrders
}

export default paymentAPI
