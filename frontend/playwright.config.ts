import { defineConfig, devices } from '@playwright/test'

// Optional: point at an already-installed Chromium when the bundled one does not match this Playwright version.
const executablePath = process.env.PW_CHROMIUM_PATH || undefined
const launchOptions = executablePath ? { executablePath } : {}

export default defineConfig({
  testDir: 'e2e',
  webServer: { command: 'pnpm build && pnpm preview --port 4173 --strictPort', port: 4173, reuseExistingServer: true, timeout: 180_000 },
  use: { baseURL: 'http://localhost:4173' },
  projects: [
    { name: 'desktop-light', use: { ...devices['Desktop Chrome'], viewport: { width: 1360, height: 900 }, colorScheme: 'light', launchOptions } },
    { name: 'desktop-dark', use: { ...devices['Desktop Chrome'], viewport: { width: 1360, height: 900 }, colorScheme: 'dark', launchOptions } },
    { name: 'mobile', use: { ...devices['Pixel 7'], colorScheme: 'light', launchOptions } },
  ],
})
