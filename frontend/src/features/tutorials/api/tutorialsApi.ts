import { apiClient } from '@/shared/api/apiClient'

export interface Tutorial {
  id: number
  title: string
  title_en: string
  client: string
  platform: string
  markdown: string
  markdown_en: string
  sort_order: number
  published: boolean
  updated_at: string
}

export type TutorialInput = Omit<Tutorial, 'id' | 'updated_at'>

export const listTutorials = () => apiClient.get<Tutorial[]>('/tutorials')
export const manageTutorials = () => apiClient.get<Tutorial[]>('/settings/tutorials')
export const saveTutorial = (id: number | null, data: TutorialInput) => id
  ? apiClient.put<Tutorial>(`/settings/tutorials/${id}`, data)
  : apiClient.post<Tutorial>('/settings/tutorials', data)
export const deleteTutorial = (id: number) => apiClient.delete(`/settings/tutorials/${id}`)
