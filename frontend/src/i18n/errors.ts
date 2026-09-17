import { getI18n } from '@/i18n'

/** Keep backend error codes visible to the i18n unused-keys lint. */
export function errorMessage(code: string, params: Record<string, string> = {}) {
  const t = getI18n().global.t
  switch (code) {
    case 'config_unreadable':
      return String(t('errors.config_unreadable', params))
    case 'config_write_failed':
      return String(t('errors.config_write_failed', params))
    case 'invalid_locale':
      return String(t('errors.invalid_locale', params))
    case 'invalid_theme':
      return String(t('errors.invalid_theme', params))
    case 'invalid_visual_effects':
      return String(t('errors.invalid_visual_effects', params))
    case 'http_port_in_use':
      return String(t('errors.http_port_in_use', params))
    default:
      return String(t('errors.internal', params))
  }
}
