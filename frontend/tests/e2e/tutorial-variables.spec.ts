import { expect, test } from '@playwright/test'
import { resolveTutorialVariables, variableRequirements } from '../../src/features/tutorials/tutorialVariables'

test('a model ID variable does not require a key or endpoint', () => {
  expect(variableRequirements('{{ model_id }}')).toEqual({ key: false, endpoint: false, model: true })
  expect(variableRequirements('{{api_key}} {{responses_url}} {{model_id}}')).toEqual({ key: true, endpoint: true, model: true })
  expect(variableRequirements('{{api_endpoint}}')).toEqual({ key: false, endpoint: true, model: false })
  expect(variableRequirements('literal code')).toEqual({ key: false, endpoint: false, model: false })
})

test('model IDs are replaced consistently without altering literal dollar signs or unknown variables', () => {
  const text = 'model = "{{model_id}}"\nother = "{{ model_id }}"\n{{api_key}}\n{{api_base_url}}\n{{unknown}}'
  expect(resolveTutorialVariables(text, 'test-key', 'https://example.test/v1/', 'vendor/$model'))
    .toBe('model = "vendor/$model"\nother = "vendor/$model"\ntest-key\nhttps://example.test/v1\n{{unknown}}')
  expect(resolveTutorialVariables('{{api_endpoint}} {{responses_url}} {{chat_completions_url}} {{claude_messages_url}}', '', 'https://example.test/v1/'))
    .toBe('https://example.test/v1 https://example.test/v1/responses https://example.test/v1/chat/completions https://example.test/v1/messages')
})
