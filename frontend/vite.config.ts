import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'
export default defineConfig({
  plugins: [tailwindcss()],
  build: { assetsInlineLimit: 0 },
  server: {
    proxy: {
      '/api': { target: 'http://127.0.0.1:8080', changeOrigin: false },
      '/media': { target: 'http://127.0.0.1:8080', changeOrigin: false },
    },
  },
})
