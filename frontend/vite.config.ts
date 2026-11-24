import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import path from 'path'

// Two builds share dist/: the dashboard (index.html, absolute /assets/ URLs, SPA routes at any depth) and,
// with `--mode me`, the « Mes statistiques » page (me.html) with relative ./assets/ URLs, because it is served
// under <jellyfin>/Jellystics/Me/ by the plugin relay and under /me/ by Jellystics. Separate builds keep the
// me bundle free of dashboard code.
export default defineConfig(({ mode }) => ({
  base: mode === 'me' ? './' : '/',
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3001,
    proxy: {
      '/api': { target: 'http://localhost:3000', changeOrigin: true },
      '/auth': { target: 'http://localhost:3000', changeOrigin: true },
      '/sync': { target: 'http://localhost:3000', changeOrigin: true },
      '/stats': { target: 'http://localhost:3000', changeOrigin: true },
      '/sessions': { target: 'http://localhost:3000', changeOrigin: true },
      '/backup': { target: 'http://localhost:3000', changeOrigin: true },
      '/logs': { target: 'http://localhost:3000', changeOrigin: true },
      '/utils': { target: 'http://localhost:3000', changeOrigin: true },
      '/webhooks': { target: 'http://localhost:3000', changeOrigin: true },
      '/proxy': { target: 'http://localhost:3000', changeOrigin: true },
      '/socket.io': { target: 'http://localhost:3000', ws: true },
    },
  },
  build: mode === 'me'
    ? { outDir: '../dist', emptyOutDir: false, copyPublicDir: false, rollupOptions: { input: path.resolve(__dirname, 'me.html') } }
    : { outDir: '../dist', emptyOutDir: true },
}))
