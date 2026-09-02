import { apiClient } from '@/shared/api/apiClient'

export const upstreamSectionNames = [
  'gemini-api-key',
  'codex-api-key',
  'xai-api-key',
  'claude-api-key',
  'vertex-api-key',
  'openai-compatibility',
] as const

export type UpstreamSection = (typeof upstreamSectionNames)[number]
export type UpstreamItem = Record<string, unknown>

export interface DiscoveredUpstreamModel {
  name: string
  alias?: string
  display_name?: string
}

export interface DiscoverUpstreamModelsRequest {
  section: UpstreamSection
  base_url: string
  api_key?: string
  auth_index?: string
  headers?: Record<string, string>
}

interface UpstreamSectionsResponse {
  sections: Partial<Record<UpstreamSection, UpstreamItem[]>>
}

export async function listUpstreamSections(): Promise<Record<UpstreamSection, UpstreamItem[]>> {
  const response = await apiClient.get<UpstreamSectionsResponse>('/upstreams')
  return Object.fromEntries(
    upstreamSectionNames.map((name) => [name, Array.isArray(response.sections[name]) ? response.sections[name] : []]),
  ) as Record<UpstreamSection, UpstreamItem[]>
}

export function replaceUpstreamSection(section: UpstreamSection, items: UpstreamItem[]) {
  return apiClient.put<void>(`/upstreams/${section}`, items)
}

export async function discoverUpstreamModels(payload: DiscoverUpstreamModelsRequest): Promise<DiscoveredUpstreamModel[]> {
  const response = await apiClient.post<{ models: DiscoveredUpstreamModel[] }>('/upstreams/models', payload)
  return Array.isArray(response.models) ? response.models : []
}
