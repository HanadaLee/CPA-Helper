import { expect, test, type Page } from '@playwright/test'

const adminRoutes = [
  ['/admin/usage', /用量分析|Usage Analytics/i],
  ['/admin/records', /请求明细|Request Records/i],
  ['/admin/users', /用户管理|Users/i],
  ['/admin/pricing', /模型价格|Model Prices/i],
  ['/admin/upstreams', /上游管理|Upstreams/i],
  ['/admin/account-mgmt', /账号管理|Account Management/i],
  ['/admin/cpamc', 'CPAMC'],
  ['/admin/settings', /系统设置|System Settings/i],
] as const

const accountRoutes = [
  ['/account/usage', /我的用量|My Usage/i],
  ['/account/records', /我的明细|My Records/i],
  ['/account/keys', /API 密钥|API Keys/i],
  ['/account/models', /可用模型|Available Models/i],
  ['/account/settings', /账户设置|Account Settings/i],
] as const

async function setupOrLogin(page: Page) {
  await page.goto('/login')
  const inputs = page.locator('input')
  await expect(inputs.first()).toBeVisible()
  await inputs.nth(0).fill('admin')
  await inputs.nth(1).fill('test-password')
  if ((await inputs.count()) >= 3) {
    await inputs.nth(2).fill('管理员')
  }
  await page.locator('button[type="submit"]').click()
  await page.waitForURL(/\/admin\/usage|\/change-credentials/)
  if (page.url().includes('/change-credentials')) {
    const passwordInputs = page.locator('input[type="password"]')
    await passwordInputs.nth(0).fill('test-password')
    await passwordInputs.nth(1).fill('new-test-password')
    await page.locator('button').filter({ hasText: /保存|Save/ }).click()
    await page.waitForURL(/\/admin\/usage/)
  }
}

async function expectTableInsidePanel(page: Page, tableSelector: string, panelSelector: string) {
  const tableBox = await page.locator(tableSelector).boundingBox()
  const panelBox = await page.locator(panelSelector).boundingBox()
  expect(tableBox).not.toBeNull()
  expect(panelBox).not.toBeNull()
  expect((tableBox?.y ?? 0) + (tableBox?.height ?? 0)).toBeLessThanOrEqual(
    (panelBox?.y ?? 0) + (panelBox?.height ?? 0) + 1,
  )
  await expect.poll(() =>
    page.locator('.content-scroll').evaluate((element) => element.scrollHeight <= element.clientHeight + 1),
  ).toBe(true)
}

test('all migrated routes render and core controls remain interactive', async ({ page }) => {
  const consoleErrors: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error' && !message.text().startsWith('Failed to load resource:')) {
      consoleErrors.push(message.text())
    }
  })
  page.on('pageerror', (error) => consoleErrors.push(error.message))

  await setupOrLogin(page)
  await expect(page.getByText(/管理中心|Admin Center/, { exact: true })).toBeVisible()

  for (const [route, title] of adminRoutes) {
    await page.goto(route)
    await expect(page.locator('.desktop-location')).toContainText(title)
    await expect(page.locator('[data-page-title]')).toBeVisible()
    await expect(page.locator('.desktop-location')).toHaveCSS('visibility', 'hidden')
  }

  for (const [route, title] of accountRoutes) {
    await page.goto(route)
    await expect(page.locator('.desktop-location')).toContainText(title)
    await expect(page.locator('[data-page-title]')).toBeVisible()
  }

  await page.goto('/admin/account-mgmt')
  const accountTypeFilter = page.getByRole('combobox', { name: /账号类型|Account Type/ })
  await accountTypeFilter.click()
  const accountTypeSearch = page.getByPlaceholder(/账号类型|Account Type/)
  await expect(accountTypeSearch).toBeVisible()
  await accountTypeSearch.fill('missing-account-type')
  await accountTypeSearch.press('Escape')
  await expect(page.locator('.account-table-footer .n-pagination')).toBeVisible()
  await expect(page.locator('.account-table-footer').getByRole('combobox')).toBeVisible()

  await page.goto('/admin/pricing')
  const statusFilter = page.getByRole('combobox').nth(1)
  await expect(statusFilter).toHaveAccessibleName(/全部状态|All statuses/)
  await statusFilter.click()
  const firstStatusOption = page.getByRole('option').first()
  const firstStatusLabel = await firstStatusOption.innerText()
  await firstStatusOption.click()
  await expect(statusFilter).toContainText(firstStatusLabel)
  await page.getByRole('button', { name: 'Clear selection' }).click()
  await expectTableInsidePanel(page, '.price-table', '.price-table-panel')
  await page.getByRole('button', { name: /新增价格|Add price/ }).click()
  const priceDialog = page.getByRole('dialog', { name: /新增价格|Add price/ })
  const priceCancelButton = priceDialog.getByRole('button', { name: /取消|Cancel/ })
  await expect(priceCancelButton).toHaveCSS('cursor', 'pointer')
  await expect(priceCancelButton).toHaveCSS('border-top-width', '1px')
  await priceCancelButton.click()

  await page.goto('/admin/upstreams')
  const upstreamSearch = page.getByPlaceholder(/搜索密钥、地址或前缀|Search keys, URLs, or prefixes/)
  await expect(upstreamSearch.locator('..')).toHaveAttribute('data-slot', 'input-group')
  await expect(upstreamSearch.locator('..').locator('svg.lucide-search')).toBeVisible()
  const upstreamCreateButton = page.getByRole('button', { name: /新建|New/ })
  const upstreamAlignment = await page.locator('.provider-panel__toolbar').evaluate((toolbar) => {
    const search = toolbar.querySelector<HTMLElement>('[data-slot="input-group"]')
    const create = toolbar.querySelector<HTMLElement>('.provider-create-button')
    if (!search || !create) return null
    const searchBox = search.getBoundingClientRect()
    const createBox = create.getBoundingClientRect()
    return {
      yDifference: Math.abs(searchBox.y - createBox.y),
      heightDifference: Math.abs(searchBox.height - createBox.height),
    }
  })
  expect(upstreamAlignment).not.toBeNull()
  expect(upstreamAlignment?.yDifference).toBeLessThanOrEqual(1)
  expect(upstreamAlignment?.heightDifference).toBeLessThanOrEqual(1)
  await expect(upstreamCreateButton).toHaveCSS('cursor', 'pointer')
  await upstreamCreateButton.click()
  const upstreamDrawer = page.locator('[data-slot="sheet-content"]')
  await expect(upstreamDrawer).toBeVisible()
  await expect(upstreamDrawer).toHaveCSS('width', '720px')
  await page.getByRole('button', { name: /取消|Cancel/ }).click()

  await page.goto('/admin/users')
  await page.getByRole('button', { name: /打开 .*操作菜单|Open actions for/ }).first().click()
  await expect(page.getByRole('menuitem', { name: /编辑|Edit/ })).toBeVisible()
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: /增加用户|Add user/ }).click()
  await expect(page.getByText(/增加用户|Add user/, { exact: true }).last()).toBeVisible()
  const adminSwitch = page.locator('button[role="switch"]').first()
  await adminSwitch.click()
  await expect(adminSwitch).toHaveAttribute('data-state', 'checked')
  await page.getByRole('button', { name: /取消|Cancel/ }).click()

  await page.goto('/admin/records')
  await expect(page.locator('input[type="datetime-local"]')).toHaveCount(0)
  await expect(page.locator('.n-date-range-start')).toBeVisible()
  await expect(page.locator('.n-date-range-end')).toBeVisible()
  await page.locator('.n-date-range-trigger').click()
  await expect(page.locator('[data-slot="range-calendar"]')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.locator('table')).toBeVisible()
  await expect(page.getByRole('columnheader', { name: /^(来源|Source)$/ })).toHaveCount(0)
  await expect(page.getByRole('columnheader', { name: /^(思考|Reasoning)$/ })).toHaveCount(0)
  await expect(page.getByRole('columnheader', { name: /^(接口|Endpoint)$/ })).toHaveCount(0)
  await expect(page.getByRole('columnheader', { name: /^(费用|Cost)$/ })).toHaveCount(0)
  await expect(page.locator('.records-table .n-data-table-th').first()).toHaveCSS('border-top-left-radius', /[1-9]/)
  await expectTableInsidePanel(page, '.records-table', '.records-table-panel')

  await page.goto('/admin/settings')
  const brandingSection = page.locator('.settings-section').filter({ hasText: /界面品牌|Interface branding/ })
  const brandingInputs = brandingSection.locator('input')
  await expect(brandingInputs).toHaveCount(4)
  await expect(brandingInputs.nth(0)).toHaveValue('CPA-Helper')
  await brandingInputs.nth(0).fill('测试边缘网关')
  await brandingInputs.nth(1).fill('Test Edge Gateway')
  await page.getByRole('button', { name: /保存设置|Save settings/ }).click()
  await expect(page.locator('.brand-copy strong')).toHaveText(/测试边缘网关|Test Edge Gateway/)
  await brandingInputs.nth(0).fill('CPA-Helper')
  await brandingInputs.nth(1).fill('CPA-Helper')
  await page.getByRole('button', { name: /保存设置|Save settings/ }).click()
  await expect(page.locator('.brand-copy strong')).toHaveText('CPA-Helper')
  await page.getByRole('button', { name: /追加 Endpoint|Add endpoint/ }).click()
  const extraEndpointInputs = page.locator('.extra-endpoint-row input')
  const extraEndpointURLBox = await extraEndpointInputs.nth(0).boundingBox()
  const extraEndpointDescriptionBox = await extraEndpointInputs.nth(1).boundingBox()
  expect(extraEndpointURLBox).not.toBeNull()
  expect(extraEndpointDescriptionBox).not.toBeNull()
  expect(Math.abs((extraEndpointURLBox?.width ?? 0) - (extraEndpointDescriptionBox?.width ?? 0))).toBeLessThanOrEqual(1)
  const switches = page.locator('button[role="switch"]')
  await expect(switches.first()).toBeVisible()
  await page.locator('.content-scroll').evaluate((element) => element.scrollTo({ top: element.scrollHeight }))
  await expect(page.locator('.desktop-location')).toHaveCSS('visibility', 'visible')

  await page.goto('/account/keys')
  await expect(page.locator('.desktop-location')).toContainText(/API 密钥|API Keys/i)
  await page.getByRole('button', { name: /新建|New API key/i }).click()
  const apiKeyDialog = page.getByRole('dialog', { name: /新建 API 密钥|New API key/i })
  await expect(apiKeyDialog).toBeVisible()
  const editorInputBox = await apiKeyDialog.getByRole('textbox').boundingBox()
  const editorSaveBox = await apiKeyDialog.getByRole('button', { name: /创建|Create/ }).boundingBox()
  expect(editorInputBox).not.toBeNull()
  expect(editorSaveBox).not.toBeNull()
  expect((editorSaveBox?.y ?? 0) - ((editorInputBox?.y ?? 0) + (editorInputBox?.height ?? 0))).toBeGreaterThanOrEqual(8)
  await page.getByRole('button', { name: /取消|Cancel/ }).click()
  await expect(page.locator('.api-endpoint-type-options .n-radio-button').first()).toHaveCSS('justify-content', 'center')
  await expect(page.locator('.api-endpoint-type-options .n-radio-button').first()).toHaveCSS('cursor', 'pointer')
  const endpointSwitchBox = await page.locator('.api-endpoint-panel .request-endpoint-switch').boundingBox()
  const endpointOptionsBox = await page.locator('.api-endpoint-type-options').boundingBox()
  expect(endpointSwitchBox).not.toBeNull()
  expect(endpointOptionsBox).not.toBeNull()
  expect((endpointOptionsBox?.x ?? 0) + (endpointOptionsBox?.width ?? 0)).toBeLessThan(
    (endpointSwitchBox?.x ?? 0) + (endpointSwitchBox?.width ?? 0) - 12,
  )

  expect(consoleErrors).toEqual([])
})

test('theme and mobile navigation survive the migration', async ({ page }) => {
  await setupOrLogin(page)
  await expect.poll(() =>
    page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--cpa-primary').trim()),
  ).toBe('#2563eb')
  await expect(page.getByRole('button', { name: /用量分析|Usage Analytics/ })).toHaveCSS('cursor', 'pointer')
  await expect(page.locator('a[href*="github.com/walkingddd/CPA-Helper"]')).toHaveCount(0)
  await page.getByRole('button', { name: /admin.*管理员|admin.*Admin/i }).click()
  await expect(page.getByRole('menuitem', { name: /退出登录|Sign out/ })).toBeVisible()
  await page.keyboard.press('Escape')

  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await expect(page.locator('div[data-state="collapsed"][data-side="left"]')).toBeVisible()
  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await page.getByRole('button', { name: /切换主题|Switch theme/ }).first().click()
  await page.getByRole('button', { name: /切换主题|Switch theme/ }).first().click()
  await expect(page.locator('html')).toHaveClass(/dark/)
  await expect.poll(() =>
    page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--cpa-primary').trim()),
  ).toBe('#60a5fa')

  await page.setViewportSize({ width: 390, height: 844 })
  await page.reload()
  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await expect(page.getByText('CPA-Helper', { exact: false }).last()).toBeVisible()
  await expect(page.getByRole('button', { name: /用量分析|Usage Analytics/ }).first()).toBeVisible()
  await page.getByRole('button', { name: /用户管理|Users/ }).click()
  await expect(page).toHaveURL(/\/admin\/users/)
  await expect(page.locator('[data-mobile="true"]')).toBeHidden()
})
