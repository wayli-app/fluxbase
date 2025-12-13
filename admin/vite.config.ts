import path from 'path'
import type { ServerResponse } from 'http'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'

// Helper to handle proxy errors gracefully during backend restarts
const handleProxyError = (err: Error, res: ServerResponse) => {
  // eslint-disable-next-line no-console
  console.error('Proxy error:', err.message)
  if (res && !res.headersSent && res.writeHead) {
    res.writeHead(503, {
      'Content-Type': 'text/html',
      'Cache-Control': 'no-store, no-cache, must-revalidate',
    })
    res.end(`
      <html>
        <head><title>Backend Unavailable</title></head>
        <body style="font-family: system-ui; padding: 2rem; text-align: center;">
          <h1>Backend Unavailable</h1>
          <p>The backend is restarting. Retrying in 2 seconds...</p>
          <script>setTimeout(() => location.reload(), 2000)</script>
        </body>
      </html>
    `)
  }
}

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
  optimizeDeps: {
    exclude: ['esbuild'],
  },
  build: {
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      external: ['esbuild'],
    },
  },
  server: {
    host: '0.0.0.0', // Listen on all interfaces (required for devcontainer port forwarding)
    port: 5050,
    strictPort: true, // Fail if port is already in use
    proxy: {
      // Proxy v1 storage API with special handling for file uploads
      '/api/v1/storage': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        timeout: 600000, // 10 minute timeout for large file uploads
        proxyTimeout: 600000,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            handleProxyError(err, res as ServerResponse)
          })
          proxy.on('proxyReq', (proxyReq) => {
            // Set longer timeout on the proxy request
            proxyReq.setTimeout(600000)
          })
        },
      },
      // Proxy v1 API requests to the backend
      '/api/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            handleProxyError(err, res as ServerResponse)
          })
        },
      },
      // Keep non-versioned endpoints
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            handleProxyError(err, res as ServerResponse)
          })
        },
      },
      '/openapi.json': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            handleProxyError(err, res as ServerResponse)
          })
        },
      },
      '/realtime': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            handleProxyError(err, res as ServerResponse)
          })
        },
      },
      // Proxy AI WebSocket for chatbot testing
      '/ai/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            handleProxyError(err, res as ServerResponse)
          })
        },
      },
    },
  },
})
