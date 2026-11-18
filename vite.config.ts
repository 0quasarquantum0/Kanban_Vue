import { fileURLToPath, URL } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// Where the backend listens. Set KANBAN_API in the environment (or in a git-ignored
// .env.local file) to point the app at another one: a Go server started with `-addr :8081`,
// or a different implementation of docs/api.md.
const env = loadEnv('development', process.cwd(), 'KANBAN_')
const apiTarget = env.KANBAN_API ?? 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), vueDevTools()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    // The app calls relative URLs such as /api/state. In development Vite forwards them to
    // the backend, so the browser sees one origin and no CORS setup is needed.
    proxy: {
      '/api': { target: apiTarget, changeOrigin: true },
    },
  },
})
