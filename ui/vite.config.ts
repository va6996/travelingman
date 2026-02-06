import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [react()],
    server: {
        port: 5173,
        proxy: {
            // Proxy API requests to backend during development
            '/TravelService': {
                target: 'http://localhost:8000',
                changeOrigin: true,
            },
        },
    },
    // In production, the base path will be served by the Go server
    // so we keep it as '/'
    base: '/',
})
