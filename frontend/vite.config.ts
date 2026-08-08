import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080'
const stripApiPrefix = process.env.VITE_API_PROXY_STRIP_PREFIX === 'true'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
        // Prism не учитывает базовый путь servers.url из OpenAPI,
        // поэтому для эмулятора префикс /api нужно срезать.
        ...(stripApiPrefix
          ? { rewrite: (path) => path.replace(/^\/api/, '') }
          : {}),
      },
    },
  },
})
