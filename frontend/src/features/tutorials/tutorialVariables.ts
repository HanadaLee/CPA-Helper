import type { UserApiKeySummary } from '@/shared/types/api'

export interface TutorialEndpoint { label: string; baseURL: string }
export interface TutorialContext {
  apiKeys: Pick<UserApiKeySummary, 'api_key' | 'api_key_hash' | 'description' | 'disabled'>[]
  endpoints: TutorialEndpoint[]
}

export const emptyTutorialContext: TutorialContext = { apiKeys: [], endpoints: [] }
export const tutorialVariables = ['api_key', 'api_base_url', 'responses_url', 'chat_completions_url', 'claude_messages_url'] as const
export const variablePattern = /\{\{\s*(api_key|api_base_url|api_endpoint|responses_url|chat_completions_url|claude_messages_url)\s*\}\}/g

export function variableRequirements(text: string) {
  const names = [...text.matchAll(new RegExp(variablePattern))].map((match) => match[1])
  return { key: names.includes('api_key'), endpoint: names.some((name) => name !== 'api_key') }
}

export function resolveTutorialVariables(text: string, key: string, baseURL: string) {
  const base = baseURL.replace(/\/$/, '')
  const values: Record<string, string> = {
    api_key: key, api_base_url: base, api_endpoint: base,
    responses_url: `${base}/responses`,
    chat_completions_url: `${base}/chat/completions`,
    claude_messages_url: `${base}/messages`,
  }
  return text.replace(new RegExp(variablePattern), (_, name: string) => values[name] ?? '')
}
