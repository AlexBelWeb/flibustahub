import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig, type Plugin } from 'vite'

const root = path.dirname(fileURLToPath(import.meta.url))

function restoreDistGitkeep(): Plugin {
  return {
    name: 'restore-dist-gitkeep',
    closeBundle() {
      fs.writeFileSync(path.resolve(root, 'dist/.gitkeep'), '')
    },
  }
}

export default defineConfig({
  plugins: [vue(), tailwindcss(), restoreDistGitkeep()],
  resolve: {
    alias: {
      '@': path.resolve(root, './src'),
    },
  },
})
