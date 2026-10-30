import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// h t t p s : / / v i t e j s . d e v / c o n f i g /
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000, // Запуск фронтенда строго на порту 3000 по ТЗ
    proxy: {
      // Все запросы к /api Vite автоматически перенаправит на наш Go-бэкенд
      '/api': {
        target: 'http://backend:8080',
        changeOrigin: true,
        secure: false,
      }
    }
  }
})
