import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import tsconfigPaths from 'vite-tsconfig-paths'

// https://vite.dev/config/
export default defineConfig({
  // Served by the Go server at /admin/ (see server/README.md, "Deployment").
  base: '/admin/',
  server: {
    // Used when dev runs against the real Go server: VITE_USE_MOCKS=false bun run dev
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  plugins: [
    tsconfigPaths(),
    tanstackRouter({
      target: 'react',
      autoCodeSplitting: true,
    }),
    react()
  ],
})
