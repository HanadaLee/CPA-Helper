import { existsSync } from 'node:fs'
import { defineConfig } from '@playwright/test'

const windowsChromePath = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'
const executablePath = process.env.CPA_HELPER_CHROME_PATH
  || (existsSync(windowsChromePath) ? windowsChromePath : undefined)

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL: process.env.CPA_HELPER_E2E_BASE_URL || 'http://127.0.0.1:5174',
    browserName: 'chromium',
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    viewport: { width: 1440, height: 1000 },
    launchOptions: executablePath ? { executablePath } : {},
  },
})
