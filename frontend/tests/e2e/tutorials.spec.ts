import { expect, test, type Page } from '@playwright/test'

async function login(page: Page) {
  await page.request.post('/api/auth/setup', { data: { username: 'admin', password: 'test-password', nickname: 'Admin' } })
  const response = await page.request.post('/api/auth/login', { data: { username: 'admin', password: 'test-password' } })
  expect(response.ok()).toBeTruthy()
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: {
      writeText: async (text: string) => { (window as Window & { copiedText?: string }).copiedText = text },
    } })
  })
}

async function clipboard(page: Page) {
  return page.evaluate(() => (window as Window & { copiedText?: string }).copiedText ?? '')
}

async function mockKeys(page: Page, count: number) {
  await page.route('**/api/api-keys', (route) => route.fulfill({ json: Array.from({ length: count }, (_, index) => ({
    api_key_hash: `test-hash-${index}`, api_key: `sk-e2e-test-${index}`, description: `Test key ${index + 1}`, disabled: false,
    user_id: 1, user_name: 'Admin', records: 0, success_records: 0, failed_records: 0, total_tokens: 0,
    created_at: '2026-09-21T01:00:00Z', updated_at: '2026-09-21T01:00:00Z',
    today_records: 0, today_success_records: 0, today_failed_records: 0, today_input_tokens: 0, today_output_tokens: 0, today_cached_tokens: 0,
  })) }))
}

async function mockEndpoints(page: Page, extra = false) {
  await page.route('**/api/account/model-request', (route) => route.fulfill({ json: {
    model_request_url: 'https://gateway.example', openai_base_url: 'https://gateway.example/v1',
    chat_completions_url: 'https://gateway.example/v1/chat/completions',
    extra_endpoints: extra ? [{ url: 'https://backup.example/prefix', description: 'Backup endpoint' }] : [],
  } }))
}

async function mockModelTutorial(page: Page, markdown: string) {
  await page.route('**/api/tutorials', (route) => route.fulfill({ json: [{
    id: 101, title: '模型选择', title_en: 'Model selection', markdown, markdown_en: '', sort_order: 0, published: true,
  }] }))
}

function model(id: string, keyIndexes = [0]) {
  return { id, name: `Display name for ${id}`, sources: keyIndexes.map((index) => ({ api_key_hash: `test-hash-${index}` })) }
}

const illustratedTutorial = `# Illustrated tutorial

![Install **screen**](https://images.example.test/install.png "Install step")

![Reference screenshot][screen]

![](/tutorial-test-image.svg)

[![Linked screenshot](https://images.example.test/linked.webp)](https://guide.example.test/)

[screen]: https://images.example.test/reference.svg "Reference image"

<div class="tutorial-html-layout">
<details open><summary>HTML instructions</summary><p><strong>Formatted HTML</strong><br><em>HTML emphasis</em><img src="https://images.example.test/html-image" alt="HTML screenshot" width="320"><span>{{api_base_url}}</span></p></details>
</div>

1. First step
2. Second step

> A useful note with ~~old text~~.

| Field | Value |
| --- | --- |
| Example | Enabled |

\`\`\`html
<img src="example.png" alt="A & B">
{{api_key}}
\`\`\`
`

async function mockTutorialImages(page: Page) {
  const image = '<svg xmlns="http://www.w3.org/2000/svg" width="1800" height="450"><rect width="1800" height="450" fill="#dcfce7"/><text x="60" y="240" font-size="70" fill="#166534">Tutorial image</text></svg>'
  for (const url of ['https://images.example.test/**', '**/tutorial-test-image.svg']) {
    await page.route(url, (route) => route.fulfill({ contentType: 'image/svg+xml', body: image }))
  }
}

test('published tutorials render standard Markdown images and trusted HTML without breaking copy controls', async ({ page }) => {
  await login(page)
  await mockKeys(page, 1)
  await mockEndpoints(page)
  await mockTutorialImages(page)
  await mockModelTutorial(page, illustratedTutorial)
  await page.goto('/account/keys')
  const article = page.locator('[data-tutorial-guide] article')
  await expect(article.getByRole('heading', { name: 'Illustrated tutorial' })).toBeVisible()
  await expect(article.locator('img')).toHaveCount(5)
  for (const image of await article.locator('img').all()) {
    await expect(image).toBeVisible()
    await expect.poll(() => image.evaluate((element) => (element as HTMLImageElement).naturalWidth)).toBe(1800)
  }
  await expect(article.getByRole('img', { name: 'Install screen', exact: true })).toHaveAttribute('title', 'Install step')
  await expect(article.getByRole('img', { name: 'Reference screenshot', exact: true })).toHaveAttribute('title', 'Reference image')
  await expect(article.locator('img[alt=""]')).toHaveAttribute('src', '/tutorial-test-image.svg')
  await expect(article.getByRole('link', { name: 'Linked screenshot', exact: true })).toHaveAttribute('href', 'https://guide.example.test/')
  await expect(article.locator('details')).toHaveAttribute('open', '')
  await expect(article.locator('.tutorial-html-layout details p strong')).toHaveText('Formatted HTML')
  await expect(article.locator('.tutorial-html-layout details p em')).toHaveText('HTML emphasis')
  await expect(article.locator('ol > li')).toHaveText(['First step', 'Second step'])
  await expect(article.locator('blockquote s')).toHaveText('old text')
  await expect(article.locator('table tbody td')).toHaveText(['Example', 'Enabled'])
  await expect(article.locator('pre img')).toHaveCount(0)
  await expect(article.locator('pre')).toContainText('<img src="example.png" alt="A & B">')
  await article.getByRole('button', { name: 'Base URL', exact: true }).click()
  await expect.poll(() => clipboard(page)).toBe('https://gateway.example/v1')
  await article.getByRole('button', { name: 'Copy code', exact: true }).click()
  await expect.poll(() => clipboard(page)).toBe('<img src="example.png" alt="A & B">\nsk-e2e-test-0\n')
  await article.scrollIntoViewIfNeeded()
  await page.screenshot({ path: 'test-results/tutorial-images-desktop.png' })
  await page.setViewportSize({ width: 390, height: 844 })
  await article.getByRole('heading', { name: 'Illustrated tutorial' }).scrollIntoViewIfNeeded()
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy()
  const articleBox = await article.boundingBox()
  for (const image of await article.locator('img').all()) {
    const imageBox = await image.boundingBox()
    expect(imageBox!.width).toBeLessThanOrEqual(articleBox!.width)
  }
  await page.screenshot({ path: 'test-results/tutorial-images-mobile.png' })
})

test('tutorial editor preview renders the same images and trusted HTML as published articles', async ({ page }) => {
  await login(page)
  await mockTutorialImages(page)
  await page.goto('/admin/settings')
  await page.getByRole('tab', { name: 'Tutorial management', exact: true }).click()
  await page.locator('[data-tutorial-manager]').getByRole('button', { name: 'New tutorial' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.locator('#tutorial-title').fill('Image preview')
  await dialog.locator('#tutorial-body').fill(illustratedTutorial)
  await dialog.getByRole('tab', { name: 'Preview', exact: true }).click()
  await expect(dialog.locator('.tutorial-prose img')).toHaveCount(5)
  await expect(dialog.getByRole('img', { name: 'Install screen', exact: true })).toBeVisible()
  await expect.poll(() => dialog.getByRole('img', { name: 'Install screen', exact: true }).evaluate((element) => (element as HTMLImageElement).naturalWidth)).toBe(1800)
  await expect(dialog.locator('.tutorial-html-layout details p strong')).toHaveText('Formatted HTML')
  await expect(dialog.getByRole('button', { name: 'Base URL', exact: true })).toBeDisabled()
  await expect(dialog.getByRole('button', { name: 'Copy code', exact: true })).toBeDisabled()
  await page.screenshot({ path: 'test-results/tutorial-images-preview.png' })
  await dialog.getByRole('button', { name: 'Cancel', exact: true }).click()
  await expect(dialog).toBeHidden()
})

test('model ID loads on demand, copies the ID and deduplicates single choices', async ({ page }) => {
  await login(page)
  await mockKeys(page, 1)
  await mockEndpoints(page)
  await mockModelTutorial(page, '{{model_id}}\n\n```toml\nmodel = "{{model_id}}"\nfallback = "{{ model_id }}"\n```')
  let requests = 0
  let release!: () => void
  const gate = new Promise<void>((resolve) => { release = resolve })
  await page.route('**/api/account/models', async (route) => {
    requests++
    await gate
    await route.fulfill({ json: { models: [model('test-model-id'), model('test-model-id')] } })
  })
  await page.goto('/account/keys')
  const guide = page.locator('[data-tutorial-guide]')
  const button = guide.getByRole('button', { name: 'Model ID', exact: true }).first()
  await expect(button).toBeVisible()
  expect(requests).toBe(0)
  await button.click()
  await expect.poll(() => requests).toBe(1)
  await expect(button).toBeDisabled()
  expect(await clipboard(page)).toBe('')
  release()
  await expect.poll(() => clipboard(page)).toBe('test-model-id')
  await expect(page.locator('[data-slot="popover-content"]')).toHaveCount(0)
  await guide.getByRole('button', { name: 'Copy code', exact: true }).click()
  await expect.poll(() => clipboard(page)).toBe('model = "test-model-id"\nfallback = "test-model-id"\n')
  expect(requests).toBe(1)
})

test('model choices are searchable and code resolves key, endpoint and matching model', async ({ page }) => {
  await login(page)
  await mockKeys(page, 2)
  await mockEndpoints(page, true)
  await mockModelTutorial(page, '{{model_id}}\n\n```text\nKEY={{api_key}}\nURL={{api_base_url}}\nMODEL={{model_id}}\n```')
  const models = Array.from({ length: 7 }, (_, index) => model(`test-model-${index + 1}`, index < 5 ? [0] : [1]))
  await page.route('**/api/account/models', (route) => route.fulfill({ json: { models } }))
  await page.goto('/account/keys')
  const guide = page.locator('[data-tutorial-guide]')
  const popover = page.locator('[data-slot="popover-content"]')
  await guide.getByRole('button', { name: 'Model ID', exact: true }).first().click()
  await expect(popover.getByRole('button')).toHaveCount(7)
  await popover.getByRole('textbox', { name: 'Search options' }).fill('model-7')
  await expect(popover.getByRole('button')).toHaveCount(1)
  await page.screenshot({ path: 'test-results/tutorial-model-picker.png' })
  await popover.getByRole('button', { name: /test-model-7/ }).click()
  await expect.poll(() => clipboard(page)).toBe('test-model-7')
  await guide.getByRole('button', { name: 'Copy code', exact: true }).click()
  await popover.getByRole('button', { name: /Test key 2/ }).click()
  await popover.getByRole('button', { name: /Backup endpoint/ }).click()
  await expect(popover).toContainText('Select a model')
  await expect(popover.getByRole('button')).toHaveCount(2)
  await expect(popover).not.toContainText('test-model-1')
  await popover.getByRole('button', { name: /test-model-6/ }).click()
  await expect.poll(() => clipboard(page)).toBe('KEY=sk-e2e-test-1\nURL=https://backup.example/prefix/v1\nMODEL=test-model-6\n')
})

test('a key without models cannot be copied as a model-key pair', async ({ page }) => {
  await login(page)
  await mockKeys(page, 2)
  await mockEndpoints(page)
  await mockModelTutorial(page, '```text\n{{api_key}} {{model_id}}\n```')
  await page.route('**/api/account/models', (route) => route.fulfill({ json: { models: [model('only-key-2', [1])] } }))
  await page.goto('/account/keys')
  const copy = page.locator('[data-tutorial-guide]').getByRole('button', { name: 'Copy code', exact: true })
  const popover = page.locator('[data-slot="popover-content"]')
  await copy.click()
  await popover.getByRole('button', { name: /Test key 1/ }).click()
  await expect(page.locator('[data-sonner-toast]').filter({ hasText: 'No available models' })).toBeVisible()
  expect(await clipboard(page)).toBe('')
  await copy.click()
  await popover.getByRole('button', { name: /Test key 2/ }).click()
  await expect.poll(() => clipboard(page)).toBe('sk-e2e-test-1 only-key-2\n')
  await expect(popover).toHaveCount(0)
})

test('failed or empty model lists do not copy and refresh reloads available models', async ({ page }) => {
  await login(page)
  await mockKeys(page, 1)
  await mockEndpoints(page)
  await mockModelTutorial(page, '{{model_id}}')
  let fail = true
  let models: ReturnType<typeof model>[] = []
  await page.route('**/api/account/models', (route) => route.fulfill(fail
    ? { status: 503, json: { detail: { message: 'Models temporarily unavailable' } } }
    : { json: { models } }))
  await page.goto('/account/keys')
  const button = page.locator('[data-tutorial-guide]').getByRole('button', { name: 'Model ID', exact: true })
  await button.click()
  await expect(page.locator('[data-sonner-toast]').last()).toContainText('Models temporarily unavailable')
  expect(await clipboard(page)).toBe('')
  fail = false
  await button.click()
  await expect(page.locator('[data-sonner-toast]').filter({ hasText: 'No available models' })).toBeVisible()
  expect(await clipboard(page)).toBe('')
  models = [model('newly-available')]
  await page.locator('.page-toolbar').getByRole('button', { name: 'Refresh', exact: true }).click()
  await button.click()
  await expect.poll(() => clipboard(page)).toBe('newly-available')
})

test('tutorials show below API keys and single choices copy without popovers', async ({ page }) => {
  await login(page)
  await mockKeys(page, 1)
  await mockEndpoints(page)
  let release!: () => void
  const gate = new Promise<void>((resolve) => { release = resolve })
  await page.route('**/api/tutorials', async (route) => { await gate; await route.continue() })
  await page.goto('/account/keys')
  const guide = page.locator('[data-tutorial-guide]')
  await expect(guide.locator('[data-tutorial-loading]')).toBeVisible()
  release()
  await expect(guide.getByRole('tablist')).toHaveCount(1)
  await expect(guide.getByRole('tab', { name: 'Codex CLI Windows', exact: true })).toBeVisible()
  await guide.getByRole('tab', { name: 'Codex CLI Windows', exact: true }).click()
  await expect(guide.locator('article')).toContainText('1. Prepare')
  const panel = await page.locator('.api-key-panel-shell').boundingBox()
  const guideBox = await guide.boundingBox()
  expect(guideBox!.y).toBeGreaterThan(panel!.y + panel!.height)
  await guide.getByRole('button', { name: /基础 URL|Base URL/, exact: true }).click()
  await expect.poll(() => clipboard(page)).toBe('https://gateway.example/v1')
  await expect(page.locator('[data-slot="popover-content"]')).toHaveCount(0)
  await guide.getByRole('button', { name: /API 密钥|API key/, exact: true }).click()
  await expect.poll(() => clipboard(page)).toBe('sk-e2e-test-0')
  await expect(page.locator('[data-slot="popover-content"]')).toHaveCount(0)
  // No real key is inserted into the article DOM.
  await expect(guide).not.toContainText('sk-e2e-test-0')
  const config = guide.locator('.tutorial-code').filter({ hasText: 'model_provider' })
  await config.getByRole('button', { name: /复制代码|Copy code/ }).click()
  await expect.poll(() => clipboard(page)).toContain('base_url = "https://gateway.example/v1"')
  await expect.poll(() => clipboard(page)).not.toContain('{{api_base_url}}')
  await page.setViewportSize({ width: 390, height: 844 })
  await guide.scrollIntoViewIfNeeded()
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy()
  for (const label of await guide.getByRole('tab').locator('span').all()) {
    await expect.poll(() => label.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBeTruthy()
  }
  await page.screenshot({ path: 'test-results/tutorials-mobile.png' })
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.evaluate(() => localStorage.setItem('cpa-helper-language', 'zh'))
  await page.reload()
  await expect(guide.getByRole('tab').first()).toHaveText('Codex CLI Windows')
  await expect(guide.locator('article')).toContainText('准备密钥与模型')
  await guide.scrollIntoViewIfNeeded()
  await page.screenshot({ path: 'test-results/tutorials-desktop.png' })
})

test('multiple keys/endpoints prompt and code copy resolves both variables', async ({ page }) => {
  await login(page)
  await mockKeys(page, 2)
  await mockEndpoints(page, true)
  await page.route('**/api/tutorials', (route) => route.fulfill({ json: [{
    id: 101, title: '变量复制测试', title_en: 'Variable copy test',
    markdown: 'Key: {{api_key}}\n\nURL: {{api_base_url}}\n\n```text\nKEY={{api_key}}\nURL={{responses_url}}\n```\n\n[unsafe](javascript:alert(1))',
    markdown_en: '', sort_order: 0, published: true,
  }] }))
  await page.goto('/account/keys')
  const guide = page.locator('[data-tutorial-guide]')
  await expect(guide.locator('article')).toBeVisible()
  await guide.getByRole('button', { name: /API 密钥|API key/, exact: true }).first().click()
  const popover = page.locator('[data-slot="popover-content"]')
  await expect(popover).toBeVisible()
  await expect.poll(() => clipboard(page)).toBe('')
  await popover.getByRole('button', { name: /Test key 2/ }).click()
  await expect.poll(() => clipboard(page)).toBe('sk-e2e-test-1')
  await guide.getByRole('button', { name: /基础 URL|Base URL/, exact: true }).click()
  await expect(popover).toBeVisible()
  await popover.getByRole('button', { name: /Backup endpoint/ }).click()
  await expect.poll(() => clipboard(page)).toBe('https://backup.example/prefix/v1')
  await guide.getByRole('button', { name: /复制代码|Copy code/ }).click()
  await popover.getByRole('button', { name: /Test key 1/ }).click()
  await expect(popover).toContainText(/endpoint/i)
  await popover.getByRole('button', { name: /Backup endpoint/ }).click()
  await expect.poll(() => clipboard(page)).toBe('KEY=sk-e2e-test-0\nURL=https://backup.example/prefix/v1/responses\n')
  await expect(guide.locator('a[href^="javascript:"]')).toHaveCount(0)
  await expect(guide).not.toContainText('sk-e2e-test-0')
})

test('each tutorial title has its own tab and keeps selection by ID across refresh', async ({ page }) => {
  await login(page)
  const article = (id: number, title: string) => ({
    id, title, title_en: '',
    markdown: `Content ${id}`, markdown_en: '', published: true, sort_order: id,
  })
  let articles = [article(1, 'Codex Windows Desktop'), article(2, 'Codex macOS Desktop'), article(3, 'Claude Code')]
  await page.route('**/api/tutorials', (route) => route.fulfill({ json: articles }))
  await page.goto('/account/keys')
  const guide = page.locator('[data-tutorial-guide]')
  await expect(guide.getByRole('tablist')).toHaveCount(1)
  await expect(guide.getByRole('tab')).toHaveText(['Codex Windows Desktop', 'Codex macOS Desktop', 'Claude Code'])
  await expect(guide.locator('article')).toHaveCount(1)
  await expect(guide.locator('article')).toHaveText('Content 1')
  await guide.getByRole('tab').nth(1).click()
  await expect(guide.locator('article')).toHaveCount(1)
  await expect(guide.locator('article')).toHaveText('Content 2')
  await guide.getByRole('tab', { name: 'Claude Code', exact: true }).click()
  await expect(guide.locator('article')).toHaveCount(1)
  await expect(guide.locator('article')).toContainText('Content 3')
  articles[2] = { ...articles[2]!, title: 'Claude Code 中文', title_en: 'Claude Code English' }
  await page.locator('.page-toolbar').getByRole('button', { name: /^(刷新|Refresh)$/ }).click()
  await expect(guide.getByRole('tab', { name: 'Claude Code English', exact: true })).toHaveAttribute('data-state', 'active')
  await expect(guide.locator('article')).toHaveText('Content 3')
  articles = articles.slice(0, 2)
  await expect(page.locator('.page-toolbar').getByRole('button', { name: /^(刷新|Refresh)$/ })).toBeEnabled()
  await page.locator('.page-toolbar').getByRole('button', { name: /^(刷新|Refresh)$/ }).click()
  await expect(guide.getByRole('tab')).toHaveText(['Codex Windows Desktop', 'Codex macOS Desktop'])
  await expect(guide.locator('article')).toHaveCount(1)
  await expect(guide.locator('article')).toHaveText('Content 1')
  await expect(guide.getByRole('tab').first()).toHaveAttribute('data-state', 'active')
  articles = []
  await expect(page.locator('.page-toolbar').getByRole('button', { name: /^(刷新|Refresh)$/ })).toBeEnabled()
  await page.locator('.page-toolbar').getByRole('button', { name: /^(刷新|Refresh)$/ }).click()
  await expect(guide.getByRole('tab')).toHaveCount(0)
  await expect(guide.locator('article')).toHaveCount(0)
  await expect(guide).toContainText('No tutorials yet')
})

test('unavailable keys cannot be copied and a failed tutorial request can be retried', async ({ page }) => {
  await login(page)
  await mockEndpoints(page)
  await page.route('**/api/api-keys', (route) => route.fulfill({ json: [
    { api_key_hash: 'disabled', api_key: 'sk-disabled-secret', description: 'Disabled', disabled: true },
    { api_key_hash: 'missing', api_key: null, description: 'Missing', disabled: false },
  ] }))
  let fail = true
  await page.route('**/api/tutorials', async (route) => {
    if (fail) await route.fulfill({ status: 503, json: { detail: { message: 'Unavailable' } } })
    else await route.continue()
  })
  await page.goto('/account/keys')
  const guide = page.locator('[data-tutorial-guide]')
  await expect(guide.getByRole('alert')).toBeVisible()
  fail = false
  await guide.getByRole('button', { name: /重试|Retry/ }).click()
  await expect(guide.locator('article')).toBeVisible()
  await guide.getByRole('button', { name: /API 密钥|API key/, exact: true }).click()
  await expect(page.locator('[data-sonner-toast]').last()).toContainText(/暂无可用密钥|No available key/)
  expect(await clipboard(page)).toBe('')
  await expect(page.locator('[data-slot="popover-content"]')).toHaveCount(0)
  await guide.getByRole('button', { name: /基础 URL|Base URL/, exact: true }).click()
  await expect.poll(() => clipboard(page)).toBe('https://gateway.example/v1')
})

test('tutorial management saves drafts, inserts variables, previews, publishes and deletes', async ({ page }) => {
  await login(page)
  await page.goto('/admin/settings')
  await page.getByRole('tab', { name: /教程管理|Tutorial management/, exact: true }).click()
  const manager = page.locator('[data-tutorial-manager]')
  await expect(manager.getByRole('table')).toBeVisible()
  const tabBox = await page.getByRole('tablist').first().boundingBox()
  const managerBox = await manager.boundingBox()
  expect(managerBox!.y).toBeGreaterThan(tabBox!.y + tabBox!.height)
  await manager.getByRole('button', { name: /新建教程|New tutorial/ }).click()
  const dialog = page.getByRole('dialog')
  await dialog.locator('#tutorial-title').fill('E2E temporary tutorial')
  await expect(dialog.locator('#tutorial-platform, #tutorial-client, #tutorial-category')).toHaveCount(0)
  await dialog.locator('#tutorial-body').fill('## Example\n\nUse ')
  await dialog.getByRole('button', { name: 'api_key', exact: true }).click()
  await expect(dialog.locator('#tutorial-body')).toHaveValue('## Example\n\nUse {{api_key}}')
  await dialog.getByRole('button', { name: 'model_id', exact: true }).click()
  await expect(dialog.locator('#tutorial-body')).toHaveValue('## Example\n\nUse {{api_key}}{{model_id}}')
  await page.screenshot({ path: 'test-results/tutorials-editor.png' })
  await dialog.getByRole('tab', { name: /预览|Preview/ }).click()
  await expect(dialog.getByRole('heading', { name: 'Example' })).toBeVisible()
  await expect(dialog.getByRole('button', { name: /API 密钥|API key/, exact: true })).toBeDisabled()
  await expect(dialog.getByRole('button', { name: /模型 ID|Model ID/, exact: true })).toBeDisabled()
  await expect(dialog.locator('[data-slot="dialog-footer"]')).toHaveCSS('box-shadow', 'none')
  // A duplicate must keep the editor open and preserve the unsaved body.
  await dialog.locator('#tutorial-title').fill('Codex CLI Windows')
  await dialog.getByRole('button', { name: /保存教程|Save tutorial/ }).click()
  await expect(page.locator('[data-sonner-toast]').last()).toContainText(/教程标题重复|already uses this title/)
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('heading', { name: 'Example' })).toBeVisible()
  await dialog.locator('#tutorial-title').fill('E2E temporary tutorial')
  await dialog.getByRole('button', { name: /保存教程|Save tutorial/ }).click()
  await expect(dialog).toBeHidden()
  const row = manager.getByRole('row').filter({ hasText: 'E2E temporary tutorial' })
  await expect(row).toContainText(/草稿|Draft/)
  await expect(manager.getByRole('columnheader')).toHaveText(['Title', 'Order', 'Status', 'Actions'])
  const all = await (await page.request.get('/api/settings/tutorials')).json() as { id: number; title: string }[]
  const id = all.find((item) => item.title === 'E2E temporary tutorial')!.id
  try {
    expect(await (await page.request.get('/api/tutorials')).text()).not.toContain('E2E temporary tutorial')
    await row.getByRole('button', { name: /编辑|Edit/, exact: true }).click()
    await dialog.locator('#tutorial-title-en').fill('Codex CLI Windows')
    await dialog.getByRole('button', { name: /保存教程|Save tutorial/ }).click()
    await expect(page.locator('[data-sonner-toast]').last()).toContainText(/教程标题重复|already uses this title/)
    await expect(dialog).toBeVisible()
    await dialog.locator('#tutorial-title-en').fill('')
    await dialog.locator('#tutorial-published').click()
    await dialog.getByRole('button', { name: /保存教程|Save tutorial/ }).click()
    await expect(dialog).toBeHidden()
    await expect(row).toContainText(/已发布|Published/)
    expect(await (await page.request.get('/api/tutorials')).text()).toContain('E2E temporary tutorial')
    await page.screenshot({ path: 'test-results/tutorials-management.png' })
    await row.getByRole('button', { name: /删除教程|Delete tutorial/ }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: /^(删除|Delete)$/ }).click()
    await expect(row).toHaveCount(0)
  } finally { await page.request.delete(`/api/settings/tutorials/${id}`) }
})
