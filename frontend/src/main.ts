import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { createAppI18n } from './i18n'
import { fallbackLocale } from './i18n/registry'
import { router } from './router'
import { useAppStore } from './stores/app'
import { installLayeredEscape } from '@/lib/layered-escape'
import './style.css'

installLayeredEscape()

async function boot() {
  const i18n = await createAppI18n(fallbackLocale)
  const app = createApp(App)
  app.use(createPinia())
  app.use(i18n)
  app.use(router)
  app.mount('#app')
  const store = useAppStore()
  await store.load()
}

void boot()
