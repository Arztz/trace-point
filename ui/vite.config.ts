import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  
  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      port: parseInt(env.FRONTEND_PORT || '3000'),
      proxy: {
        '/api': {
          target: env.BACKEND_URL || 'http://localhost:8081',
          changeOrigin: true,
        },
      },
    },
  };
});