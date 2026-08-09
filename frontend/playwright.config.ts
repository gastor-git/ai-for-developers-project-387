import { defineConfig } from '@playwright/test'

const backendUrl = 'http://localhost:8080'
const frontendUrl = 'http://localhost:5173'
const dbPath = '/tmp/booking-e2e.db'

// Локально можно использовать системный Chrome (в CI ставится штатный
// Chromium командой `playwright install --with-deps chromium`):
// PLAYWRIGHT_USE_SYSTEM_CHROME=1 npm run test:e2e
const useSystemChrome = process.env.PLAYWRIGHT_USE_SYSTEM_CHROME === '1'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  use: {
    baseURL: frontendUrl,
    locale: 'en-US',
    screenshot: 'only-on-failure',
    trace: 'on-first-retry',
    ...(useSystemChrome ? { channel: 'chrome' } : {}),
  },
  webServer: [
    {
      // Свежая БД для каждого прогона: удаляем файлы перед стартом.
      command: `rm -f ${dbPath}* && go run .`,
      cwd: '../backend',
      url: `${backendUrl}/api/health`,
      reuseExistingServer: false,
      timeout: 120_000,
      env: {
        DB_PATH: dbPath,
        ADDR: ':8080',
      },
    },
    {
      command: 'npm run dev -- --strictPort',
      cwd: '.',
      url: frontendUrl,
      reuseExistingServer: false,
      timeout: 120_000,
      env: {
        VITE_API_PROXY_TARGET: backendUrl,
      },
    },
  ],
})
