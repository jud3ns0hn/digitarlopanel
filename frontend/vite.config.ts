import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The build output goes straight into the Go backend's embed directory so a
// single `go build` produces a self-contained binary.
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../backend/web/dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      output: {
        // Split the heaviest libraries into their own chunks. This keeps any
        // single chunk small, which lowers the build's peak memory use and
        // helps it finish on low-RAM VPSes.
        manualChunks: {
          echarts: ['echarts'],
          'element-plus': ['element-plus', '@element-plus/icons-vue'],
          xterm: ['@xterm/xterm', '@xterm/addon-fit'],
          vue: ['vue', 'vue-router', 'pinia'],
        },
      },
    },
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
