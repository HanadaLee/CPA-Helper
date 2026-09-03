import { apiClient } from '@/shared/api/apiClient'
import type {
  ApiKeyCreatePayload,
  ApiKeyUpdatePayload,
  ModelRequestGuide,
  ModelRequestTestPayload,
  ModelRequestTestResponse,
  UserApiKeySummary,
} from '@/shared/types/api'

function toApiKeyList(value: unknown): UserApiKeySummary[] {
  return Array.isArray(value) ? (value as UserApiKeySummary[]) : []
}

export function listApiKeys(): Promise<UserApiKeySummary[]> {
  return apiClient.get<unknown>('/api-keys').then(toApiKeyList)
}

export function createApiKey(payload: ApiKeyCreatePayload): Promise<UserApiKeySummary> {
  return apiClient.post<UserApiKeySummary>('/api-keys', payload)
}

export function updateApiKey(
  apiKeyHash: string,
  payload: ApiKeyUpdatePayload,
): Promise<UserApiKeySummary> {
  return apiClient.put<UserApiKeySummary>(`/api-keys/${apiKeyHash}`, payload)
}

export function disableApiKey(apiKeyHash: string): Promise<UserApiKeySummary> {
  return apiClient.post<UserApiKeySummary>(`/api-keys/${apiKeyHash}/disable`, {})
}

export function enableApiKey(apiKeyHash: string): Promise<UserApiKeySummary> {
  return apiClient.post<UserApiKeySummary>(`/api-keys/${apiKeyHash}/enable`, {})
}

export function deleteApiKey(apiKeyHash: string): Promise<void> {
  return apiClient.delete(`/api-keys/${apiKeyHash}`)
}

export function getModelRequestGuide(): Promise<ModelRequestGuide> {
  return apiClient.get<ModelRequestGuide>('/account/model-request')
}

export function testModelRequest(
  payload: ModelRequestTestPayload,
): Promise<ModelRequestTestResponse> {
  return apiClient.post<ModelRequestTestResponse>('/account/model-request/test', payload)
}
