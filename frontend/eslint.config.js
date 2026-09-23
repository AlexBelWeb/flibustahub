import pluginVue from 'eslint-plugin-vue'
import vueI18n from '@intlify/eslint-plugin-vue-i18n'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },
  {
    name: 'app/files-to-ignore',
    ignores: ['dist/**', 'wailsjs/**', 'node_modules/**'],
  },
  pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  ...vueI18n.configs['flat/recommended'],
  {
    rules: {
      'vue/multi-word-component-names': 'off',
      '@intlify/vue-i18n/no-raw-text': [
        'error',
        {
          ignorePattern: '^[-#:()0-9./]+$',
        },
      ],
      '@intlify/vue-i18n/no-missing-keys': 'error',
      '@intlify/vue-i18n/no-unused-keys': [
        'error',
        {
          src: './src',
          extensions: ['.js', '.ts', '.vue'],
        },
      ],
    },
    settings: {
      'vue-i18n': {
        localeDir: './src/locales/*.{json,json5,yaml,yml}',
        messageSyntaxVersion: '^11.0.0',
      },
    },
  },
  {
    files: ['src/views/palette-preview/**'],
    rules: {
      '@intlify/vue-i18n/no-raw-text': 'off',
    },
  },
  skipFormatting,
)
