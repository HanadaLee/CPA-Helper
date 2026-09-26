import { apiClient } from '@/shared/api/apiClient'
import type {
  QuotaCard,
  QuotaCardPayload,
  QuotaCharge,
  QuotaPage,
  QuotaReset,
  QuotaTargets,
  UserQuotaStatus,
} from '@/shared/types/api'

export function listQuotaCards(
  admin: boolean,
  page = 1,
  userId?: number,
  kind?: string,
  status?: string,
) {
  const params = new URLSearchParams({ page: String(page), page_size: '20' })
  if (userId) params.set('user_id', String(userId))
  if (kind) params.set('kind', kind)
  if (status) params.set('status', status)
  return apiClient.get<QuotaPage<QuotaCard>>(
    `${admin ? '/quota' : '/account/quota'}/cards?${params}`,
  )
}

export function issueQuotaCards(payload: QuotaCardPayload) {
  return apiClient.post<{ issued: number; batch_id: string }>('/quota/cards/issue', payload)
}

export function editQuotaCard(id: number, payload: QuotaCardPayload) {
  return apiClient.put<void>(`/quota/cards/${id}`, payload)
}

export function revokeQuotaCards(payload: QuotaTargets & { card_ids?: number[]; kind?: string }) {
  return apiClient.post<{ revoked: number }>('/quota/cards/revoke', payload)
}

export function resetQuotas(payload: QuotaTargets) {
  return apiClient.post<{ reset: number }>('/quota/reset', payload)
}

export function useResetCard(id: number) {
  return apiClient.post<UserQuotaStatus>(`/account/quota/cards/${id}/use`)
}

export function listQuotaCharges(page = 1) {
  return apiClient.get<QuotaPage<QuotaCharge>>(`/account/quota/history?page=${page}&page_size=20`)
}

export function listQuotaResets(page = 1) {
  return apiClient.get<QuotaPage<QuotaReset>>(`/account/quota/resets?page=${page}&page_size=20`)
}
