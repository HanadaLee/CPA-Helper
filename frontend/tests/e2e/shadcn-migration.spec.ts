import { expect, test, type Locator, type Page } from '@playwright/test'

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

async function expectPlainDialogFooter(dialog: Locator) {
  const footer = dialog.locator('[data-slot="dialog-footer"]')
  await expect(footer).toHaveCSS('border-top-width', '0px')
  await expect(footer).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)')
  await expect(footer).toHaveCSS('box-shadow', 'none')
}

test('initial navigation uses shell and dashboard skeletons instead of a blank screen', async ({ page }) => {
  await setupOrLogin(page)
  await expect(page.locator('[data-page-title]')).toBeVisible()

  const cdp = await page.context().newCDPSession(page)
  await cdp.send('Network.setCacheDisabled', { cacheDisabled: true })

  let releaseAuthRequest!: () => void
  let releaseUsageRequest!: () => void
  const authRequestGate = new Promise<void>((resolve) => {
    releaseAuthRequest = resolve
  })
  const usageRequestGate = new Promise<void>((resolve) => {
    releaseUsageRequest = resolve
  })
  let holdAuthRequest = true
  let holdUsageRequest = true

  await page.route('**/api/auth/me', async (route) => {
    if (holdAuthRequest) {
      holdAuthRequest = false
      await authRequestGate
    }
    await route.continue()
  })
  await page.route('**/api/usage/overview**', async (route) => {
    if (holdUsageRequest) {
      holdUsageRequest = false
      await usageRequestGate
    }
    await route.continue()
  })

  await page.goto('/admin/usage', { waitUntil: 'domcontentloaded' })

  const appSkeleton = page.locator('[data-app-loading="true"]')
  await expect(appSkeleton).toBeVisible()
  await expect.poll(() => appSkeleton.locator('[data-slot="skeleton"]').count()).toBeGreaterThan(12)
  const startupPanels = appSkeleton.locator('.startup-panels > .startup-panel')
  const startupTrendBox = await startupPanels.nth(0).boundingBox()
  const startupTokenBox = await startupPanels.nth(1).boundingBox()
  expect(startupTrendBox).not.toBeNull()
  expect(startupTokenBox).not.toBeNull()
  expect(
    Math.abs((startupTrendBox?.width ?? 0) - ((startupTokenBox?.width ?? 0) * 2 + 16)),
  ).toBeLessThanOrEqual(1)
  await expect(appSkeleton.locator('.startup-token-legend [data-slot="skeleton"]')).toHaveCount(4)

  releaseAuthRequest()
  await expect(appSkeleton).toBeHidden()
  await expect(page.locator('[data-page-title]')).toBeVisible()
  await expect(page.locator('.filter-panel')).toBeVisible()

  const dashboardSkeleton = page.locator('[data-usage-loading="true"]')
  await expect(dashboardSkeleton).toBeVisible()
  await expect.poll(() => dashboardSkeleton.locator('[data-slot="skeleton"]').count()).toBeGreaterThan(20)
  const skeletonTopPanels = dashboardSkeleton.locator('.usage-skeleton-top-grid > .usage-skeleton-panel')
  const skeletonBottomColumns = dashboardSkeleton.locator('.usage-skeleton-bottom-grid > .usage-skeleton-column')
  const skeletonTokenBox = await skeletonTopPanels.nth(1).boundingBox()
  const skeletonRankingBox = await skeletonBottomColumns.nth(2).boundingBox()
  expect(skeletonTokenBox).not.toBeNull()
  expect(skeletonRankingBox).not.toBeNull()
  expect(Math.abs((skeletonTokenBox?.x ?? 0) - (skeletonRankingBox?.x ?? 0))).toBeLessThanOrEqual(1)
  expect(Math.abs((skeletonTokenBox?.width ?? 0) - (skeletonRankingBox?.width ?? 0))).toBeLessThanOrEqual(1)

  releaseUsageRequest()
  await expect(dashboardSkeleton).toBeHidden()
  await expect(page.locator('.dashboard-metric-grid')).toBeVisible()
  const tokenPanelBox = await page.locator('.area-token').boundingBox()
  const rankingColumnBox = await page.locator('.dashboard-column-right').boundingBox()
  expect(tokenPanelBox).not.toBeNull()
  expect(rankingColumnBox).not.toBeNull()
  expect(Math.abs((tokenPanelBox?.x ?? 0) - (rankingColumnBox?.x ?? 0))).toBeLessThanOrEqual(1)
  expect(Math.abs((tokenPanelBox?.width ?? 0) - (rankingColumnBox?.width ?? 0))).toBeLessThanOrEqual(1)
})

test('sidebar navigation switches immediately and lets the destination render its loading state', async ({ page }) => {
  await setupOrLogin(page)
  await page.goto('/admin/usage')
  await expect(page.locator('[data-page-title]')).toContainText(/用量分析|Usage Analytics/i)

  let releaseUpstreamRequest!: () => void
  const upstreamRequestGate = new Promise<void>((resolve) => {
    releaseUpstreamRequest = resolve
  })
  let releaseAuthRequest!: () => void
  const authRequestGate = new Promise<void>((resolve) => {
    releaseAuthRequest = resolve
  })
  let authRequests = 0
  await page.route('**/api/auth/me', async (route) => {
    authRequests += 1
    await authRequestGate
    await route.continue()
  })
  await page.route('**/api/upstreams', async (route) => {
    await upstreamRequestGate
    await route.continue()
  })

  await page.getByRole('button', { name: /上游管理|Upstreams/ }).click()
  await expect(page).toHaveURL(/\/admin\/upstreams/)
  await expect(page.locator('[data-page-title]')).toContainText(/上游管理|Upstream Management/i)
  await expect(page.locator('[data-page-title]').filter({ hasText: /用量分析|Usage Analytics/i })).toHaveCount(0)
  await expect(page.locator('.provider-table-shell [data-slot="skeleton"]').first()).toBeVisible()
  await expect.poll(() => authRequests).toBeGreaterThan(0)

  releaseAuthRequest()
  releaseUpstreamRequest()
  await expect(page.locator('.provider-table-shell [data-slot="skeleton"]')).toHaveCount(0)
})

test('upstream model discovery merges selected models into the editor', async ({ page }) => {
  await setupOrLogin(page)
  await page.route('**/api/upstreams', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        sections: {
          'gemini-api-key': [],
          'codex-api-key': [],
          'xai-api-key': [],
          'claude-api-key': [],
          'vertex-api-key': [],
          'openai-compatibility': [],
        },
      }),
    })
  })
  let discoveryPayload: Record<string, unknown> | null = null
  await page.route('**/api/upstreams/models', async (route) => {
    discoveryPayload = route.request().postDataJSON() as Record<string, unknown>
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        models: [
          { name: 'gemini-2.5-flash', display_name: 'Gemini 2.5 Flash' },
          { name: 'gemini-2.5-pro', display_name: 'Gemini 2.5 Pro' },
        ],
      }),
    })
  })

  await page.goto('/admin/upstreams')
  await page.getByRole('button', { name: /新建|New/ }).click()
  const upstreamDrawer = page.locator('[data-slot="sheet-content"]')
  await upstreamDrawer.locator('#upstream-api-key').fill('gemini-test-key')
  await upstreamDrawer.getByRole('button', { name: /获取模型|Fetch models/ }).click()

  const discoveryDialog = page.getByRole('dialog', { name: /获取模型|Fetch models/ })
  await expect(discoveryDialog).toBeVisible()
  await expect.poll(() => discoveryPayload?.section).toBe('gemini-api-key')
  await expect.poll(() => discoveryPayload?.api_key).toBe('gemini-test-key')
  const modelRow = discoveryDialog.locator('.model-discovery-row').filter({ hasText: 'gemini-2.5-pro' })
  await modelRow.locator('[role="checkbox"]').click()
  await discoveryDialog.getByRole('button', { name: /添加所选模型|Add selected models/ }).click()

  await expect(discoveryDialog).toBeHidden()
  await expect(upstreamDrawer.getByPlaceholder(/上游模型名称|Upstream model name/).first()).toHaveValue('gemini-2.5-pro')
  await expect(upstreamDrawer.getByPlaceholder(/显示名称（可选）|Display name \(optional\)/).first()).toHaveValue('Gemini 2.5 Pro')
})

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
    await expect(page.locator('.desktop-location')).toHaveCSS('visibility', 'visible')
  }

  for (const [route, title] of accountRoutes) {
    await page.goto(route)
    await expect(page.locator('.desktop-location')).toContainText(title)
    await expect(page.locator('[data-page-title]')).toBeVisible()
  }
  await expect(page.getByRole('button', { name: /账户设置|Account Settings/ })).toHaveCount(0)

  await page.goto('/admin/settings')
  const generalSettingsCard = page.locator('[data-slot="card"]').filter({
    has: page.getByText(/通用配置|General Settings/i, { exact: true }),
  })
  const collectionSettingsCard = page.locator('[data-slot="card"]').filter({
    has: page.getByText(/采集与保留设置|Collection and Retention Settings/i, { exact: true }),
  })
  await expect(generalSettingsCard.getByText(/^(采集与保留参数|Collection and retention)$/i)).toHaveCount(0)
  await expect(collectionSettingsCard.getByText(/^(采集与保留参数|Collection and retention)$/i)).toBeVisible()
  await expect(collectionSettingsCard.getByLabel(/批量读取数|Batch size/i)).toBeVisible()

  await page.goto('/admin/usage')
  const headerLayout = await page.locator('.app-header').evaluate((header) => {
    const headerBox = header.getBoundingClientRect()
    const title = header.querySelector<HTMLElement>('.desktop-location')
    if (!title) return null
    const titleBox = title.getBoundingClientRect()
    return {
      titleInset: titleBox.x - headerBox.x,
      titleWeight: getComputedStyle(title).fontWeight,
    }
  })
  expect(headerLayout).not.toBeNull()
  expect(headerLayout?.titleInset).toBeCloseTo(77, 0)
  expect(headerLayout?.titleWeight).toBe('500')
  const analyticsQuickRangeTabs = page.locator('.quick-ranges[data-slot="tabs"]')
  await expect(analyticsQuickRangeTabs.getByRole('tablist')).toHaveAccessibleName(/快捷时间范围|Quick time ranges/)
  await expect(analyticsQuickRangeTabs.getByRole('tab')).toHaveCount(6)
  await expect(page.locator('body')).toHaveCSS('font-family', '"Geist Variable", sans-serif')
  expect(await page.locator('body').evaluate((element) => getComputedStyle(element).fontSynthesis)).toContain('weight')
  const analyticsRangeTrigger = page.locator('[data-slot="date-time-range-trigger"]')
  const analyticsRangeBox = await analyticsRangeTrigger.boundingBox()
  expect(analyticsRangeBox?.width ?? 0).toBeGreaterThanOrEqual(420)
  expect(analyticsRangeBox?.height).toBe(32)
  expect(await analyticsRangeTrigger.innerText()).not.toMatch(/\d{2}:\d{2}:\d{2}/)
  await expect(analyticsQuickRangeTabs.getByRole('tablist')).toHaveCSS('height', '32px')
  await expect(analyticsRangeTrigger).toHaveCSS('font-size', '14px')
  expect(await analyticsRangeTrigger.locator('[data-slot="date-time-range-start"], [data-slot="date-time-range-end"]').evaluateAll(
    (elements) => elements.every((element) => element.scrollWidth <= element.clientWidth),
  )).toBe(true)
  const analyticsFilterRow = page.locator('.field-row')
  const analyticsFilterLayout = await analyticsFilterRow.evaluate((row) => {
    const children = Array.from(row.children) as HTMLElement[]
    const rowBox = row.getBoundingClientRect()
    const boxes = children.map((child) => child.getBoundingClientRect())
    return {
      yDifference: Math.max(...boxes.map((box) => box.y)) - Math.min(...boxes.map((box) => box.y)),
      leftInset: boxes[0]!.left - rowBox.left,
      rightInset: rowBox.right - boxes.at(-1)!.right,
    }
  })
  expect(analyticsFilterLayout.yDifference).toBeLessThanOrEqual(1)
  expect(Math.abs(analyticsFilterLayout.leftInset)).toBeLessThanOrEqual(1)
  expect(Math.abs(analyticsFilterLayout.rightInset)).toBeLessThanOrEqual(1)
  const analyticsSearchableFilter = analyticsFilterRow.locator('.filter-combobox-trigger').first()
  await expect(analyticsSearchableFilter).toHaveCSS('font-weight', '400')
  await expect(analyticsSearchableFilter).toHaveCSS('height', '32px')
  await expect(analyticsSearchableFilter).toHaveCSS('font-size', '14px')
  await analyticsSearchableFilter.click()
  await expect(page.getByPlaceholder(/搜索用户|Search users/)).toBeVisible()
  await page.keyboard.press('Escape')

  await page.goto('/account/settings')
  await expect(page).toHaveURL(/\/account\/settings$/)
  await expect(page.locator('[data-page-title]')).toHaveCount(0)

  await page.route('**/api/codex-keeper/accounts', async (route) => {
    await route.fulfill({
      json: {
        items: [
          {
            name: 'menu-test.json',
            email: 'menu-test@example.com',
            account_type: 'plus',
            disabled: false,
            priority: 0,
            primary_used_percent: 20,
            secondary_used_percent: 10,
            primary_reset_at: null,
            secondary_reset_at: null,
            primary_window_seconds: 18_000,
            secondary_window_seconds: 604_800,
            primary_window_usage: null,
            secondary_window_usage: null,
            quota_threshold: 90,
            last_status_code: 200,
            last_error: null,
            latest_action: null,
            last_checked_at: null,
            last_healthy_at: null,
          },
        ],
        priority_rules: [],
      },
    })
  })
  await page.route('**/api/codex-keeper/auth-files/menu-test.json', async (route) => {
    await route.fulfill({
      json: {
        json: {
          note: 'E2E auth file note',
          priority: 0,
          websockets: false,
        },
      },
    })
  })
  await page.goto('/admin/account-mgmt')
  const accountTypeFilter = page.getByRole('combobox', { name: /账号类型|Account Type/ })
  await accountTypeFilter.click()
  const accountTypeSearch = page.getByPlaceholder(/账号类型|Account Type/)
  await expect(accountTypeSearch).toBeVisible()
  await accountTypeSearch.fill('missing-account-type')
  await accountTypeSearch.press('Escape')
  const accountPagination = page.locator('[data-slot="table-pagination-footer"]')
  await expect(accountPagination.locator('[data-slot="pagination"]')).toBeVisible()
  await expect(accountPagination.getByRole('combobox', { name: /每页数量|Rows per page/ })).toBeVisible()
  await expect(accountPagination.locator('[data-slot="pagination-first"]')).toBeVisible()
  await expect(accountPagination.locator('[data-slot="pagination-last"]')).toBeVisible()
  const accountTableBox = await page.locator('.account-table-shell').boundingBox()
  const accountPaginationBox = await accountPagination.boundingBox()
  expect(accountTableBox).not.toBeNull()
  expect(accountPaginationBox).not.toBeNull()
  expect(Math.abs((accountTableBox?.x ?? 0) - (accountPaginationBox?.x ?? 0))).toBeLessThanOrEqual(1)
  expect(Math.abs((accountTableBox?.width ?? 0) - (accountPaginationBox?.width ?? 0))).toBeLessThanOrEqual(1)
  const latestActionHeaderBox = await page.getByRole('columnheader', { name: /最近操作|Latest Action/ }).boundingBox()
  expect(latestActionHeaderBox).not.toBeNull()
  expect(latestActionHeaderBox?.width).toBeLessThanOrEqual(170)
  const accountActionHeaderBox = await page.locator('.account-table thead th').last().boundingBox()
  expect(accountActionHeaderBox).not.toBeNull()
  expect(accountActionHeaderBox?.width).toBeLessThanOrEqual(64)
  const accountRow = page.locator('.account-table tbody tr').first()
  const accountFirstCell = accountRow.locator('td').first()
  const accountFixedCell = accountRow.locator('td').last()
  await accountRow.hover()
  await expect.poll(async () => {
    const [firstCellColor, fixedCellColor] = await Promise.all([
      accountFirstCell.evaluate((element) => getComputedStyle(element).backgroundColor),
      accountFixedCell.evaluate((element) => getComputedStyle(element).backgroundColor),
    ])
    return fixedCellColor === firstCellColor
  }).toBe(true)
  await page.getByRole('button', { name: /打开 .*操作菜单|Open actions for/ }).first().click()
  await expect(page.getByRole('menuitem', { name: /详情|Details/ })).toBeVisible()
  await expect(page.getByRole('menuitem', { name: /禁用|Disable/ })).toBeVisible()
  await expect(page.getByRole('menuitem', { name: /刷新|Refresh/ })).toBeVisible()
  await expect(page.getByRole('menuitem', { name: /删除|Delete/ })).toBeVisible()
  await page.getByRole('menuitem', { name: /删除|Delete/ }).click()
  const accountDeleteDialog = page.getByRole('dialog', { name: /删除账号|Delete Account/ })
  await expectPlainDialogFooter(accountDeleteDialog)
  await accountDeleteDialog.getByRole('button', { name: /取消|Cancel/ }).click()
  await page.getByRole('button', { name: /打开 .*操作菜单|Open actions for/ }).first().click()
  await page.getByRole('menuitem', { name: /详情|Details/ }).click()
  await page.getByRole('button', { name: /认证文件详情 \/ 编辑|Auth File Details \/ Edit/ }).click()
  const authFileDialog = page.getByRole('dialog', { name: /认证文件详情 \/ 编辑|Auth File Details \/ Edit/ })
  await expect(authFileDialog.getByRole('textbox').last()).toHaveValue('E2E auth file note')
  await expectPlainDialogFooter(authFileDialog)
  await authFileDialog.locator('[data-slot="dialog-footer"]').getByRole('button', { name: /关闭|Close/ }).click()

  await page.goto('/admin/pricing')
  const priceMetricCards = page.locator('.price-metric-card')
  await expect(priceMetricCards.first()).toHaveCSS('border-left-width', '1px')
  await expect(priceMetricCards.last()).toHaveCSS('border-right-width', '1px')
  const statusFilter = page.getByRole('combobox').nth(1)
  await expect(statusFilter).toHaveAccessibleName(/全部状态|All statuses/)
  await statusFilter.click()
  const firstStatusOption = page.getByRole('option').first()
  const firstStatusLabel = await firstStatusOption.innerText()
  await firstStatusOption.click()
  await expect(statusFilter).toContainText(firstStatusLabel)
  await page.getByRole('button', { name: 'Clear selection' }).click()
  await expectTableInsidePanel(page, '.price-table', '.price-table-panel')
  const pricePagination = page.locator('[data-slot="table-pagination-footer"]')
  await expect(pricePagination.getByRole('combobox', { name: /每页数量|Rows per page/ })).toBeVisible()
  await expect(pricePagination.locator('[data-slot="pagination-first"]')).toBeVisible()
  await expect(pricePagination.locator('[data-slot="pagination-last"]')).toBeVisible()
  await page.getByRole('button', { name: /新增价格|Add price/ }).click()
  const priceDialog = page.getByRole('dialog', { name: /新增价格|Add price/ })
  const fastMultiplierInput = priceDialog.locator('#price-fast-multiplier')
  await fastMultiplierInput.fill('2')
  await expect(fastMultiplierInput).toHaveValue('2')
  await expect(fastMultiplierInput).toHaveAttribute('step', '0.01')
  expect(await fastMultiplierInput.evaluate((input: HTMLInputElement) => ({
    valid: input.validity.valid,
    stepMismatch: input.validity.stepMismatch,
  }))).toEqual({ valid: true, stepMismatch: false })
  const priceCancelButton = priceDialog.getByRole('button', { name: /取消|Cancel/ })
  await expect(priceCancelButton).toHaveCSS('cursor', 'pointer')
  await expect(priceCancelButton).toHaveCSS('border-top-width', '1px')
  await expectPlainDialogFooter(priceDialog)
  await priceCancelButton.click()

  await page.goto('/admin/upstreams')
  await expect(page.getByText(/上游配置|Upstream configuration/, { exact: true })).toHaveCount(0)
  await expect(page.getByText(/按提供商类型管理 API 密钥、路由和模型映射。|Manage API keys, routes, and model mappings by provider family\./, { exact: true })).toHaveCount(0)
  const providerTabs = page.locator('.provider-tabs[data-slot="tabs"]')
  await expect(providerTabs.getByRole('tablist')).toHaveAccessibleName(/提供商|Providers/)
  await expect(providerTabs.getByRole('tab')).toHaveCount(6)
  await expect(page.locator('.provider-nav')).toHaveCount(0)
  const upstreamWorkbenchLayout = await page.locator('.upstream-workbench').evaluate((workbench) => {
    const switcher = workbench.querySelector<HTMLElement>('.provider-switcher')
    const table = workbench.querySelector<HTMLElement>('.provider-table-shell')
    if (!switcher || !table) return null
    const switcherBox = switcher.getBoundingClientRect()
    const tableBox = table.getBoundingClientRect()
    return {
      leftDifference: Math.abs(switcherBox.x - tableBox.x),
      widthDifference: Math.abs(switcherBox.width - tableBox.width),
      tableBelowSwitcher: tableBox.y >= switcherBox.bottom,
    }
  })
  expect(upstreamWorkbenchLayout).not.toBeNull()
  expect(upstreamWorkbenchLayout?.leftDifference).toBeLessThanOrEqual(1)
  expect(upstreamWorkbenchLayout?.widthDifference).toBeLessThanOrEqual(1)
  expect(upstreamWorkbenchLayout?.tableBelowSwitcher).toBe(true)
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
  await expect(upstreamCreateButton).toHaveCSS('height', '32px')
  await expect(upstreamCreateButton).toHaveCSS('cursor', 'pointer')
  await upstreamCreateButton.click()
  const upstreamDrawer = page.locator('[data-slot="sheet-content"]')
  await expect(upstreamDrawer).toBeVisible()
  await expect(upstreamDrawer).toHaveCSS('width', '720px')
  await page.getByRole('button', { name: /取消|Cancel/ }).click()

  await page.goto('/admin/users')
  await expect(page.getByRole('button', { name: /^编辑 |^Edit / }).first()).toBeVisible()
  await expect(page.getByRole('button', { name: /打开 .*操作菜单|Open actions for/ })).toHaveCount(0)
  await page.getByRole('button', { name: /增加用户|Add user/ }).click()
  await expect(page.getByText(/增加用户|Add user/, { exact: true }).last()).toBeVisible()
  const adminSwitch = page.locator('button[role="switch"]').first()
  await adminSwitch.click()
  await expect(adminSwitch).toHaveAttribute('data-state', 'checked')
  await expect(adminSwitch.locator('[data-slot="switch-thumb"]')).toHaveCSS('border-top-width', '1px')
  await page.getByRole('button', { name: /取消|Cancel/ }).click()

  const usageRecordFixture = {
    id: 101,
    timestamp: '2026-08-31T12:00:00+08:00',
    api_key_description: 'E2E key',
    user_id: 1,
    user_label: 'admin',
    provider: 'openai',
    model: 'gpt-5.6-terra',
    reasoning_effort: 'xhigh',
    request_service_tier: 'fast',
    endpoint: '/v1/responses',
    source: 'e2e',
    request_id: 'request-detail-width-e2e',
    auth_index: null,
    auth: 'api_key',
    latency_ms: 1200,
    ttft_ms: 180,
    failed: false,
    input_tokens: 1200,
    output_tokens: 400,
    cached_tokens: 200,
    cache_read_tokens: 200,
    cache_creation_tokens: 0,
    reasoning_tokens: 120,
    total_tokens: 1600,
    estimated_cost_usd: 0.012,
    unpriced: false,
  }
  await page.route('**/api/usage/records?*', async (route) => {
    await route.fulfill({
      json: {
        items: [usageRecordFixture],
        total: 1,
        page: 1,
        page_size: 50,
        start: '2026-08-31T00:00:00+08:00',
        end: '2026-09-01T00:00:00+08:00',
      },
    })
  })
  await page.route('**/api/usage/records/101*', async (route) => {
    await route.fulfill({
      json: {
        ...usageRecordFixture,
        raw_json: {
          request: {
            model: 'gpt-5.6-terra',
            input: 'A deliberately long raw payload value used to verify that the wider request detail drawer remains readable.',
          },
          response: { id: 'request-detail-width-e2e' },
        },
      },
    })
  })
  await page.goto('/admin/records')
  const quickRangeTabs = page.locator('.quick-ranges[data-slot="tabs"]')
  await expect(quickRangeTabs.getByRole('tablist')).toHaveAccessibleName(/快捷时间范围|Quick time ranges/)
  await expect(quickRangeTabs.getByRole('tab')).toHaveCount(6)
  await expect(page.locator('input[type="datetime-local"]')).toHaveCount(0)
  await expect(page.locator('[data-slot="date-time-range-start"]')).toBeVisible()
  await expect(page.locator('[data-slot="date-time-range-end"]')).toBeVisible()
  const recordsRangeTrigger = page.locator('[data-slot="date-time-range-trigger"]')
  const recordsRangeBox = await recordsRangeTrigger.boundingBox()
  expect(recordsRangeBox?.width ?? 0).toBeGreaterThanOrEqual(420)
  expect(await recordsRangeTrigger.innerText()).not.toMatch(/\d{2}:\d{2}:\d{2}/)
  expect(await recordsRangeTrigger.locator('[data-slot="date-time-range-start"], [data-slot="date-time-range-end"]').evaluateAll(
    (elements) => elements.every((element) => element.scrollWidth <= element.clientWidth),
  )).toBe(true)
  const recordsFilterLayout = await page.locator('.field-row').evaluate((row) => {
    const boxes = Array.from(row.children, (child) => (child as HTMLElement).getBoundingClientRect())
    const rowBox = row.getBoundingClientRect()
    return {
      yDifference: Math.max(...boxes.map((box) => box.y)) - Math.min(...boxes.map((box) => box.y)),
      leftInset: boxes[0]!.left - rowBox.left,
      rightInset: rowBox.right - boxes.at(-1)!.right,
    }
  })
  expect(recordsFilterLayout.yDifference).toBeLessThanOrEqual(1)
  expect(Math.abs(recordsFilterLayout.leftInset)).toBeLessThanOrEqual(1)
  expect(Math.abs(recordsFilterLayout.rightInset)).toBeLessThanOrEqual(1)
  await page.locator('[data-slot="date-time-range-trigger"]').click()
  await expect(page.locator('[data-slot="range-calendar"]')).toBeVisible()
  const rangeTimeInputs = page.locator('input[type="time"]')
  await expect(rangeTimeInputs).toHaveCount(2)
  await expect(rangeTimeInputs.nth(0)).toHaveAttribute('step', '60')
  await expect(rangeTimeInputs.nth(1)).toHaveAttribute('step', '60')
  expect(await rangeTimeInputs.evaluateAll((inputs: HTMLInputElement[]) =>
    inputs.every((input) => /^\d{2}:\d{2}$/.test(input.value)),
  )).toBe(true)
  await page.keyboard.press('Escape')
  await expect(page.locator('table')).toBeVisible()
  const adminTimeHeaderBox = await page.getByRole('columnheader', { name: /^(时间|Time)$/ }).boundingBox()
  const adminNicknameHeaderBox = await page.getByRole('columnheader', { name: /^(用户昵称|User nickname)$/ }).boundingBox()
  expect(adminTimeHeaderBox?.width ?? 0).toBeGreaterThanOrEqual(160)
  expect(adminNicknameHeaderBox?.width ?? Number.POSITIVE_INFINITY).toBeLessThanOrEqual(110)
  await expect(page.getByRole('columnheader', { name: /^(来源|Source)$/ })).toHaveCount(0)
  await expect(page.getByRole('columnheader', { name: /^(思考|Reasoning)$/ })).toHaveCount(0)
  await expect(page.getByRole('columnheader', { name: /^(接口|Endpoint)$/ })).toHaveCount(0)
  await expect(page.getByRole('columnheader', { name: /^(费用|Cost)$/ })).toHaveCount(0)
  await expect(page.locator('.records-table')).toHaveCSS('border-top-left-radius', /[1-9]/)
  await expectTableInsidePanel(page, '.records-table', '.records-table-panel')
  const recordsPagination = page.locator('[data-slot="table-pagination-footer"]')
  await expect(recordsPagination.getByRole('combobox', { name: /每页数量|Rows per page/ })).toBeVisible()
  await expect(recordsPagination.locator('[data-slot="pagination-first"]')).toBeVisible()
  await expect(recordsPagination.locator('[data-slot="pagination-last"]')).toBeVisible()
  await page.getByRole('button', { name: /详情|Details/ }).first().click()
  const requestDetailSheet = page.getByRole('dialog', { name: /请求事件详情|Request event details/ })
  const requestDetailSheetBox = await requestDetailSheet.boundingBox()
  expect(requestDetailSheetBox).not.toBeNull()
  expect(requestDetailSheetBox?.width).toBeGreaterThanOrEqual(900)
  await expect(requestDetailSheet).toHaveCSS('border-left-width', '0px')
  await expect(requestDetailSheet.locator('.mono-json')).toHaveCSS('overflow-x', 'auto')
  await requestDetailSheet.getByRole('button', { name: /Close/ }).click()

  await page.goto('/account/records')
  const accountTimeHeaderBox = await page.getByRole('columnheader', { name: /^(时间|Time)$/ }).boundingBox()
  expect(accountTimeHeaderBox?.width ?? 0).toBeGreaterThanOrEqual(160)
  await expect(page.getByRole('columnheader', { name: /^(用户昵称|User nickname)$/ })).toHaveCount(0)

  await page.goto('/admin/settings')
  const brandingSection = page.locator('.settings-section').filter({ hasText: /界面品牌|Interface branding/ })
  const accessSection = page.locator('.settings-section').filter({ hasText: /访问配置|Access control/ })
  const collectionSection = page.locator('.settings-section').filter({ hasText: /采集与保留参数|Collection and retention/ })
  await expect(accessSection).toBeVisible()
  await expect(accessSection.getByText(/开启本地采集|Enable local collection/)).toHaveCount(0)
  await expect(collectionSection.getByText(/开启本地采集|Enable local collection/)).toBeVisible()
  await expect(collectionSection.getByText(/用量明细保留天数|Usage detail retention days/)).toBeVisible()
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
  const extraEndpointGroups = page.locator('.extra-endpoint-row [data-slot="input-group"]')
  const extraEndpointURLBox = await extraEndpointGroups.nth(0).boundingBox()
  const extraEndpointDescriptionBox = await extraEndpointGroups.nth(1).boundingBox()
  const cliProxyBox = await page.locator('#cliaproxy-url').boundingBox()
  const modelRequestBox = await page.locator('#model-request-url').boundingBox()
  expect(extraEndpointURLBox).not.toBeNull()
  expect(extraEndpointDescriptionBox).not.toBeNull()
  expect(cliProxyBox).not.toBeNull()
  expect(modelRequestBox).not.toBeNull()
  expect(Math.abs((extraEndpointURLBox?.width ?? 0) - (extraEndpointDescriptionBox?.width ?? 0))).toBeLessThanOrEqual(1)
  expect(Math.abs((extraEndpointURLBox?.x ?? 0) - (cliProxyBox?.x ?? 0))).toBeLessThanOrEqual(1)
  expect(Math.abs((extraEndpointDescriptionBox?.x ?? 0) - (modelRequestBox?.x ?? 0))).toBeLessThanOrEqual(1)
  const managementKeyGroup = page.locator('#management-key').locator('..')
  await expect(extraEndpointGroups.nth(0)).toHaveCSS(
    'background-color',
    await managementKeyGroup.evaluate((element) => getComputedStyle(element).backgroundColor),
  )
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
  const endpointTypeTabs = page.locator('.api-endpoint-type-tabs[data-slot="tabs"]')
  const endpointTypeGroup = endpointTypeTabs.locator('[data-slot="tabs-list"]')
  const endpointTypeButtons = endpointTypeGroup.locator('[data-slot="tabs-trigger"]')
  await expect(endpointTypeGroup).toHaveAccessibleName(/URL 类型|URL type/)
  await expect(page.getByText(/^(URL 类型|URL type)$/)).toHaveCount(0)
  await expect(endpointTypeButtons).toHaveCount(4)
  await expect(endpointTypeButtons.first()).toHaveCSS('cursor', 'pointer')
  await expect(endpointTypeButtons.first()).toHaveAttribute('data-state', 'active')
  const endpointTypeLayout = await endpointTypeButtons.evaluateAll((elements) => {
    const boxes = elements.map((element) => element.getBoundingClientRect())
    const yPositions = boxes.map((box) => box.y)
    return {
      yDifference: Math.max(...yPositions) - Math.min(...yPositions),
    }
  })
  expect(endpointTypeLayout.yDifference).toBeLessThanOrEqual(1)
  const endpointGroupBox = await endpointTypeGroup.boundingBox()
  const endpointFirstButtonBox = await endpointTypeButtons.first().boundingBox()
  const endpointLastButtonBox = await endpointTypeButtons.last().boundingBox()
  expect(endpointGroupBox).not.toBeNull()
  expect(endpointFirstButtonBox).not.toBeNull()
  expect(endpointLastButtonBox).not.toBeNull()
  const endpointLeftInset = (endpointFirstButtonBox?.x ?? 0) - (endpointGroupBox?.x ?? 0)
  const endpointRightInset =
    ((endpointGroupBox?.x ?? 0) + (endpointGroupBox?.width ?? 0)) -
    ((endpointLastButtonBox?.x ?? 0) + (endpointLastButtonBox?.width ?? 0))
  expect(endpointLeftInset).toBeGreaterThanOrEqual(2)
  expect(endpointRightInset).toBeGreaterThanOrEqual(2)
  expect(
    Math.abs(endpointLeftInset - endpointRightInset),
  ).toBeLessThanOrEqual(1)
  await endpointTypeButtons.nth(1).click()
  await expect(endpointTypeButtons.nth(1)).toHaveAttribute('data-state', 'active')
  await expect(endpointTypeButtons.first()).toHaveAttribute('data-state', 'inactive')

  expect(consoleErrors).toEqual([])
})

test('available models uses compact price columns and shows the FAST multiplier', async ({ page }) => {
  await setupOrLogin(page)
  await page.route('**/api/account/models', async (route) => {
    await route.fulfill({
      json: {
        has_api_keys: true,
        api_key_count: 1,
        queryable_api_key_count: 1,
        errors: [],
        models: [
          {
            id: 'gpt-5.6-terra',
            name: 'GPT 5.6 Terra',
            object: 'model',
            owner: 'openai',
            created: null,
            metadata: {},
            sources: [
              {
                api_key_hash: 'fixture-key',
                api_key_preview: 'sk-…test',
                description: 'Test key',
              },
            ],
            price: {
              provider: 'openai',
              model: 'gpt-5.6-terra',
              input_usd_per_million: 2,
              output_usd_per_million: 12,
              cache_read_usd_per_million: 0.2,
              cache_creation_usd_per_million: 2.5,
              request_usd: null,
              fast_multiplier: 1.8,
              billing_unit: 'token',
            },
          },
        ],
      },
    })
  })

  await page.goto('/account/models')
  await expect(page.getByRole('columnheader', { name: /FAST 倍率|FAST multiplier/ })).toBeVisible()
  await expect(page.getByRole('cell', { name: '×1.8' })).toBeVisible()
  await expect(page.locator('.available-models-table [data-slot="table"]')).toHaveCSS('min-width', '1240px')
})

test('theme and mobile navigation survive the migration', async ({ page }) => {
  await setupOrLogin(page)
  await expect(page.locator('[data-slot="sidebar"][data-variant="inset"]')).toBeVisible()
  await expect.poll(() =>
    page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--cpa-primary').trim()),
  ).toMatch(/^oklch\((?:0\.55|55%) (?:0\.19|\.19) 257\)$/)
  await expect(page.locator('[data-slot="sidebar-container"]')).toHaveCSS('border-right-width', '0px')
  await expect(page.locator('[data-slot="sidebar-container"]')).toHaveCSS('width', '256px')
  const usageMenuButton = page.getByRole('button', { name: /用量分析|Usage Analytics/ })
  await expect(usageMenuButton).toHaveCSS('cursor', 'pointer')
  await expect(usageMenuButton).toHaveCSS('font-size', '14px')
  await expect(usageMenuButton).toHaveCSS('height', '32px')
  await expect(page.locator('a[href*="github.com/walkingddd/CPA-Helper"]')).toHaveCount(0)
  const desktopUserMenuButton = page.getByRole('button', { name: /admin.*管理员|admin.*Admin/i })
  await expect(desktopUserMenuButton.locator('.lucide-ellipsis-vertical')).toBeVisible()
  const desktopUserButtonBox = await desktopUserMenuButton.boundingBox()
  await desktopUserMenuButton.click()
  const desktopUserMenu = page.getByRole('menu')
  await expect(page.getByRole('menuitem', { name: /退出登录|Sign out/ })).toBeVisible()
  const desktopUserMenuBox = await desktopUserMenu.boundingBox()
  expect(desktopUserButtonBox).not.toBeNull()
  expect(desktopUserMenuBox).not.toBeNull()
  expect(desktopUserMenuBox?.x ?? 0).toBeGreaterThanOrEqual(
    (desktopUserButtonBox?.x ?? 0) + (desktopUserButtonBox?.width ?? 0) - 1,
  )
  await page.keyboard.press('Escape')

  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await expect(page.locator('div[data-state="collapsed"][data-side="left"]')).toBeVisible()
  await expect.poll(() => page.locator('[data-slot="sidebar-container"]').evaluate((sidebar) => {
    const mark = sidebar.querySelector<HTMLElement>('.brand-mark')
    if (!mark) return Number.POSITIVE_INFINITY
    const sidebarBox = sidebar.getBoundingClientRect()
    const markBox = mark.getBoundingClientRect()
    return Math.abs(
      (sidebarBox.left + sidebarBox.width / 2) - (markBox.left + markBox.width / 2),
    )
  })).toBeLessThanOrEqual(1)
  const collapsedBrandHeader = page.locator('[data-slot="sidebar-container"] [data-sidebar="header"]')
  await expect(collapsedBrandHeader).toHaveCSS('overflow', 'visible')
  await expect(collapsedBrandHeader.locator('.sidebar-brand-button')).toHaveCSS('overflow', 'visible')
  expect(await page.locator('[data-slot="sidebar-container"]').evaluate((sidebar) => {
    const mark = sidebar.querySelector<HTMLElement>('.brand-mark')
    if (!mark) return false
    const sidebarBox = sidebar.getBoundingClientRect()
    const markBox = mark.getBoundingClientRect()
    return markBox.left > sidebarBox.left && markBox.right < sidebarBox.right
  })).toBe(true)
  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await page.getByRole('button', { name: /切换主题|Switch theme/ }).first().click()
  await page.getByRole('button', { name: /切换主题|Switch theme/ }).first().click()
  await expect(page.locator('html')).toHaveClass(/dark/)
  await expect.poll(() =>
    page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--cpa-primary').trim()),
  ).toMatch(/^oklch\((?:0\.68|68%) (?:0\.16|\.16) 257\)$/)
  await expect(page.locator('[data-slot="sidebar-container"]')).toHaveCSS('border-right-width', '0px')

  await page.setViewportSize({ width: 390, height: 844 })
  await page.reload()
  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await expect(page.locator('[data-mobile="true"]')).toHaveCSS('border-right-color', 'oklch(1 0 0 / 0.1)')
  await expect(page.locator('[data-mobile="true"]')).toHaveCSS('width', '288px')
  await expect(page.locator('.mobile-brand-copy strong')).toBeVisible()
  await expect(page.locator('.mobile-version-badge')).toHaveCount(0)
  await expect(page.getByRole('button', { name: /用量分析|Usage Analytics/ }).first()).toBeVisible()
  const mobileUserMenuButton = page.getByRole('button', { name: /admin.*管理员|admin.*Admin/i })
  await expect(mobileUserMenuButton.locator('.lucide-ellipsis-vertical')).toBeVisible()
  const mobileUserButtonBox = await mobileUserMenuButton.boundingBox()
  await mobileUserMenuButton.click()
  const mobileUserMenuBox = await page.getByRole('menu').boundingBox()
  expect(mobileUserButtonBox).not.toBeNull()
  expect(mobileUserMenuBox).not.toBeNull()
  expect((mobileUserMenuBox?.y ?? 0) + (mobileUserMenuBox?.height ?? 0)).toBeLessThanOrEqual(
    (mobileUserButtonBox?.y ?? 0) + 1,
  )
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: /用户管理|Users/ }).click()
  await expect(page).toHaveURL(/\/admin\/users/)
  await expect(page.locator('[data-mobile="true"]')).toBeHidden()
})
