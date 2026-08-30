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
