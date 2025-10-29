import path from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'

// https://vite.dev/config/
export default defineConfig({
  // Always use /admin as base path for consistency between dev and prod
  base: '/admin/',
  plugins: [
    tanstackRouter({
      target: 'react',
      autoCodeSplitting: true,
    }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      // Force all React imports to use the admin's version (React 19)
      'react': path.resolve(__dirname, './node_modules/react'),
      'react-dom': path.resolve(__dirname, './node_modules/react-dom'),
      'react/jsx-runtime': path.resolve(__dirname, './node_modules/react/jsx-runtime'),
      'react/jsx-dev-runtime': path.resolve(__dirname, './node_modules/react/jsx-dev-runtime'),
      // Force React Query to use admin's version for consistent context
      '@tanstack/react-query': path.resolve(__dirname, './node_modules/@tanstack/react-query'),
    },
  },
  server: {
    host: '0.0.0.0', // Listen on all interfaces (required for devcontainer port forwarding)
    port: 5173,
    strictPort: true, // Fail if port is already in use
    proxy: {
      // Proxy v1 storage API with special handling for file uploads
      '/api/v1/storage': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        timeout: 600000, // 10 minute timeout for large file uploads
        proxyTimeout: 600000,
        configure: (proxy, _options) => {
          proxy.on('error', (err, _req, _res) => {
            console.log('Storage proxy error:', err)
          })
          proxy.on('proxyReq', (proxyReq, _req, _res) => {
            // Set longer timeout on the proxy request
            proxyReq.setTimeout(600000)
          })
        },
      },
      // Proxy v1 API requests to the backend
      '/api/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // Keep non-versioned endpoints
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/openapi.json': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/realtime': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
})
