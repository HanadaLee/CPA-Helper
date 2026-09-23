import { apiClient } from '@/shared/api/apiClient'
import type {
  LiteLLMProxySettings,
  LiteLLMProxySettingsPayload,
  ModelPrice,
  ModelPriceCatalogResponse,
  ModelPricePayload,
  ModelPriceSyncResponse,
  PricingHolidayCalendar,
} from '@/shared/types/api'

export function listModelPrices(): Promise<ModelPrice[]> {
  return apiClient.get<ModelPrice[]>('/model-prices')
}

export function listModelPriceCatalog(): Promise<ModelPriceCatalogResponse> {
  return apiClient.get<ModelPriceCatalogResponse>('/model-prices/catalog')
}

export function createModelPrice(payload: ModelPricePayload): Promise<ModelPrice> {
  return apiClient.post<ModelPrice>('/model-prices', payload)
}

export function updateModelPrice(id: number, payload: ModelPricePayload): Promise<ModelPrice> {
  return apiClient.put<ModelPrice>(`/model-prices/${id}`, payload)
}

export function deleteModelPrice(id: number): Promise<void> {
  return apiClient.delete(`/model-prices/${id}`)
}

export function syncLitellmModelPrices(): Promise<ModelPriceSyncResponse> {
  return apiClient.post<ModelPriceSyncResponse>('/model-prices/sync/litellm')
}

export function getLiteLLMProxySettings(): Promise<LiteLLMProxySettings> {
  return apiClient.get<LiteLLMProxySettings>('/model-prices/litellm-proxy')
}

export function updateLiteLLMProxySettings(
  payload: LiteLLMProxySettingsPayload,
): Promise<LiteLLMProxySettings> {
  return apiClient.put<LiteLLMProxySettings>('/model-prices/litellm-proxy', payload)
}

export function getPricingHolidayCalendar(year: number): Promise<PricingHolidayCalendar> {
  return apiClient.get<PricingHolidayCalendar>(`/model-prices/holiday-calendar?year=${year}`)
}

export function updatePricingHolidayCalendar(year: number, dates: string[]): Promise<PricingHolidayCalendar> {
  return apiClient.put<PricingHolidayCalendar>('/model-prices/holiday-calendar', { year, dates })
}
