import { expect, test, type Page } from '@playwright/test'

const adminRoutes = [
  ['/admin/usage', /历史用量|Usage History/i],
  ['/admin/records', /请求明细|Request Records/i],
  ['/admin/users', /用户管理|Users/i],
  ['/admin/pricing', /模型价格|Model Prices/i],
  ['/admin/upstreams', /上游管理|Upstreams/i],
  ['/admin/account-mgmt', /账号管理|Account Management/i],
  ['/admin/cpamc', 'CPAMC'],
  ['/admin/settings', /系统设置|System Settings/i],
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
    await expect(page.locator('main main h1')).toHaveCount(0)
  }

  await page.goto('/admin/account-mgmt')
  const accountTypeFilter = page.getByRole('combobox', { name: /账号类型|Account Type/ })
  await accountTypeFilter.click()
  const accountTypeSearch = page.getByPlaceholder(/账号类型|Account Type/)
  await expect(accountTypeSearch).toBeVisible()
  await accountTypeSearch.fill('missing-account-type')
  await accountTypeSearch.press('Escape')

  await page.goto('/admin/pricing')
  const statusFilter = page.getByRole('combobox').nth(1)
  await expect(statusFilter).toHaveAccessibleName(/全部状态|All statuses/)
  await statusFilter.click()
  const firstStatusOption = page.getByRole('option').first()
  const firstStatusLabel = await firstStatusOption.innerText()
  await firstStatusOption.click()
  await expect(statusFilter).toContainText(firstStatusLabel)
  await page.getByRole('button', { name: 'Clear selection' }).click()

  await page.goto('/admin/upstreams')
  await page.getByRole('button', { name: /新建|New/ }).click()
  const upstreamDrawer = page.locator('[data-slot="sheet-content"]')
  await expect(upstreamDrawer).toBeVisible()
  await expect(upstreamDrawer).toHaveCSS('width', '720px')
  await page.getByRole('button', { name: /取消|Cancel/ }).click()

  await page.goto('/admin/users')
  await page.getByRole('button', { name: /增加用户|Add user/ }).click()
  await expect(page.getByText(/增加用户|Add user/, { exact: true }).last()).toBeVisible()
  await page.getByRole('button', { name: /取消|Cancel/ }).click()

  await page.goto('/admin/records')
  await expect(page.locator('input[type="datetime-local"]')).toHaveCount(0)
  await page.locator('.n-date-range-trigger').click()
  await expect(page.locator('[data-slot="range-calendar"]')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.locator('table')).toBeVisible()

  await page.goto('/admin/settings')
  const switches = page.locator('button[role="switch"]')
  await expect(switches.first()).toBeVisible()

  await page.goto('/account/keys')
  await expect(page.locator('.desktop-location')).toContainText(/API 密钥|API Keys/i)
  await page.getByRole('button', { name: /新建|New API key/i }).click()
  await expect(page.getByRole('heading', { name: /新建 API 密钥|New API key/i })).toBeVisible()
  await page.getByRole('button', { name: /取消|Cancel/ }).click()

  expect(consoleErrors).toEqual([])
})

test('theme and mobile navigation survive the migration', async ({ page }) => {
  await setupOrLogin(page)
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

  await page.setViewportSize({ width: 390, height: 844 })
  await page.reload()
  await page.getByRole('button', { name: /打开导航|Open navigation/ }).click()
  await expect(page.getByText('CPA-Helper', { exact: false }).last()).toBeVisible()
  await expect(page.getByRole('button', { name: /历史用量|Usage History/ }).first()).toBeVisible()
  await page.getByRole('button', { name: /用户管理|Users/ }).click()
  await expect(page).toHaveURL(/\/admin\/users/)
  await expect(page.locator('[data-mobile="true"]')).toBeHidden()
})
