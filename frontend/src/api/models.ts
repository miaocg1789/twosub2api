/**
 * Model Pricing API endpoints
 * Fetches model pricing data from LiteLLM pricing repository
 */

import { apiClient } from './client'

export interface ModelPricingDisplay {
  id: string
  provider: string
  input_price: number
  output_price: number
  cache_read_price: number | null
  cache_create_price: number | null
  mode: string
}

export interface ModelPricingResponse {
  models: ModelPricingDisplay[]
  updated_at: string
}

/**
 * Get all model pricing data for display
 * @returns Model pricing list with last updated timestamp
 */
export async function getModelPricing(): Promise<ModelPricingResponse> {
  const { data } = await apiClient.get<ModelPricingResponse>('/settings/model-pricing')
  return data
}
