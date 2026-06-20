import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The build output goes straight into the Go backend's embed directory so a
// single `go build` produces a self-contained binary.
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../backend/web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8088',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
