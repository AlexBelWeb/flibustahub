import { getI18n } from '@/i18n'
import { formatBytes } from '@/lib/format'

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
    case 'db_open_failed':
      return String(t('errors.db_open_failed', params))
    case 'db_migrate_failed':
      return String(t('errors.db_migrate_failed', params))
    case 'db_incompatible':
      return String(t('errors.db_incompatible', params))
    case 'db_backup_failed':
      return String(t('errors.db_backup_failed', params))
    case 'open_dir_failed':
      return String(t('errors.open_dir_failed', params))
    case 'import_cancelled':
      return String(t('errors.import_cancelled', params))
    case 'import_failed':
      return String(t('errors.import_failed', params))
    case 'inpx_not_found':
      return String(t('errors.inpx_not_found', params))
    case 'library_unreadable':
      return String(t('errors.library_unreadable', params))
    case 'open_file_failed':
      return String(t('errors.open_file_failed', params))
    case 'not_found':
      return String(t('errors.not_found', params))
    case 'invalid_catalog_view':
      return String(t('errors.invalid_catalog_view', params))
    case 'library_offline':
      return String(t('errors.library_offline', params))
    case 'archive_missing':
      return String(t('errors.archive_missing', params))
    case 'fb2_unreadable':
      return String(t('errors.fb2_unreadable', params))
    case 'library_unreachable':
      return String(t('errors.library_unreachable', params))
    case 'reader_unavailable':
      return String(t('errors.reader_unavailable', params))
    case 'reader_missing':
      return String(t('errors.reader_missing', params))
    case 'downloads_dir_unusable':
      return String(t('errors.downloads_dir_unusable', params))
    case 'cancelled':
      return String(t('errors.cancelled', params))
    case 'invalid_rating':
      return String(t('errors.invalid_rating', params))
    case 'personal_export_failed':
      return String(t('errors.personal_export_failed', params))
    case 'personal_import_failed':
      return String(t('errors.personal_import_failed', params))
    case 'secret_invalid_id':
      return String(t('errors.secret_invalid_id', params))
    case 'secret_empty':
      return String(t('errors.secret_empty', params))
    case 'secret_too_large':
      return String(t('errors.secret_too_large', params))
    case 'secret_store_failed':
      return String(t('errors.secret_store_failed', params))
    case 'secret_store_unreadable':
      return String(t('errors.secret_store_unreadable', params))
    case 'invalid_ai_provider':
      return String(t('errors.invalid_ai_provider', params))
    case 'db_no_space': {
      const loc = String(getI18n().global.locale.value)
      return String(
        t('errors.db_no_space', {
          need: formatBytes(Number(params.need) || 0, loc),
          have: formatBytes(Number(params.have) || 0, loc),
        }),
      )
    }
    case 'db_maintenance_busy':
      return String(t('errors.db_maintenance_busy', params))
    case 'import_in_progress':
      return String(t('errors.import_in_progress', params))
    case 'cover_warmup_in_progress':
      return String(t('errors.cover_warmup_in_progress', params))
    case 'db_optimize_failed':
      return String(t('errors.db_optimize_failed', params))
    case 'diag_failed':
      return String(t('errors.diag_failed', params))
    case 'diag_archive_failed':
      return String(t('errors.diag_archive_failed', params))
    default:
      return String(t('errors.internal', params))
  }
}
