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
  await expect(page.getByRole('button', { name: /账户设置|Account Settings/ })).toHaveCount(0)
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
  await expect(page.locator('.account-table-footer .n-pagination')).toBeVisible()
  await expect(page.locator('.account-table-footer').getByRole('combobox')).toBeVisible()
  const latestActionHeaderBox = await page.getByRole('columnheader', { name: /最近操作|Latest Action/ }).boundingBox()
  expect(latestActionHeaderBox).not.toBeNull()
  expect(latestActionHeaderBox?.width).toBeLessThanOrEqual(170)
  const accountActionHeaderBox = await page.locator('.account-table .n-data-table-th').last().boundingBox()
  expect(accountActionHeaderBox).not.toBeNull()
  expect(accountActionHeaderBox?.width).toBeLessThanOrEqual(64)
  const accountRow = page.locator('.account-table .n-data-table-base-table-body .n-data-table-tr').first()
  const accountFixedCell = accountRow.locator('.n-data-table-td').last()
  await accountRow.hover()
  await expect.poll(async () => {
    const [rowColor, fixedCellColor] = await Promise.all([
      accountRow.evaluate((element) => getComputedStyle(element).backgroundColor),
      accountFixedCell.evaluate((element) => getComputedStyle(element).backgroundColor),
    ])
    return fixedCellColor === rowColor
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
  await expectPlainDialogFooter(priceDialog)
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
  await expect(adminSwitch.locator('[data-slot="switch-thumb"]')).toHaveCSS('border-top-width', '1px')
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
  const endpointTypeGroup = page.locator('.api-endpoint-type-options[data-slot="toggle-group"]')
  const endpointTypeButtons = endpointTypeGroup.locator('[data-slot="toggle-group-item"]')
  await expect(endpointTypeGroup).toHaveAttribute('aria-labelledby', 'api-endpoint-url-type-label')
  await expect(endpointTypeButtons).toHaveCount(4)
  await expect(endpointTypeButtons.first()).toHaveCSS('cursor', 'pointer')
  await expect(endpointTypeButtons.first()).toHaveAttribute('data-state', 'on')
  const endpointTypeLayout = await endpointTypeButtons.evaluateAll((elements) => {
    const boxes = elements.map((element) => element.getBoundingClientRect())
    const yPositions = boxes.map((box) => box.y)
    return {
      yDifference: Math.max(...yPositions) - Math.min(...yPositions),
    }
  })
  expect(endpointTypeLayout.yDifference).toBeLessThanOrEqual(1)
  const endpointGroupBox = await endpointTypeGroup.boundingBox()
  const endpointLastButtonBox = await endpointTypeButtons.last().boundingBox()
  expect(endpointGroupBox).not.toBeNull()
  expect(endpointLastButtonBox).not.toBeNull()
  expect(
    Math.abs(
      ((endpointGroupBox?.x ?? 0) + (endpointGroupBox?.width ?? 0)) -
      ((endpointLastButtonBox?.x ?? 0) + (endpointLastButtonBox?.width ?? 0)),
    ),
  ).toBeLessThanOrEqual(1)
  await endpointTypeButtons.nth(1).click()
  await expect(endpointTypeButtons.nth(1)).toHaveAttribute('data-state', 'on')
  await expect(endpointTypeButtons.first()).toHaveAttribute('data-state', 'off')

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
  await expect(page.locator('.available-models-table [data-slot="table-container"]')).toHaveCSS('min-width', '1360px')
})

test('theme and mobile navigation survive the migration', async ({ page }) => {
  await setupOrLogin(page)
  await expect.poll(() =>
    page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--cpa-primary').trim()),
  ).toBe('#2563eb')
  await expect(page.locator('[data-slot="sidebar-container"]')).toHaveCSS('border-right-color', 'rgba(15, 23, 42, 0.1)')
  const usageMenuButton = page.getByRole('button', { name: /用量分析|Usage Analytics/ })
  await expect(usageMenuButton).toHaveCSS('cursor', 'pointer')
  await expect(usageMenuButton).toHaveCSS('font-size', '12.25px')
  await expect(page.locator('a[href*="github.com/walkingddd/CPA-Helper"]')).toHaveCount(0)
  await page.getByRole('button', { name: /admin.*管理员|admin.*Admin/i }).click()
  await expect(page.getByRole('menuitem', { name: /退出登录|Sign out/ })).toBeVisible()
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
  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await page.getByRole('button', { name: /切换主题|Switch theme/ }).first().click()
  await page.getByRole('button', { name: /切换主题|Switch theme/ }).first().click()
  await expect(page.locator('html')).toHaveClass(/dark/)
  await expect.poll(() =>
    page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--cpa-primary').trim()),
  ).toBe('#60a5fa')
  await expect(page.locator('[data-slot="sidebar-container"]')).toHaveCSS('border-right-color', 'rgba(148, 163, 184, 0.16)')

  await page.setViewportSize({ width: 390, height: 844 })
  await page.reload()
  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await expect(page.getByText('CPA-Helper', { exact: false }).last()).toBeVisible()
  await expect(page.getByRole('button', { name: /用量分析|Usage Analytics/ }).first()).toBeVisible()
  await page.getByRole('button', { name: /用户管理|Users/ }).click()
  await expect(page).toHaveURL(/\/admin\/users/)
  await expect(page.locator('[data-mobile="true"]')).toBeHidden()
})
