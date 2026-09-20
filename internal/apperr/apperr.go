// Package apperr defines typed backend errors.
// Public payloads contain a stable code plus parameters; UI text lives in frontend locales.
package apperr

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	CodeInternal              = "internal"
	CodeConfigUnreadable      = "config_unreadable"
	CodeConfigWriteFailed     = "config_write_failed"
	CodeInvalidLocale         = "invalid_locale"
	CodeInvalidTheme          = "invalid_theme"
	CodeInvalidEffects        = "invalid_visual_effects"
	CodeHTTPPortInUse         = "http_port_in_use"
	CodeDBOpenFailed          = "db_open_failed"
	CodeDBMigrateFailed       = "db_migrate_failed"
	CodeDBIncompatible        = "db_incompatible"
	CodeDBBackupFailed        = "db_backup_failed"
	CodeOpenDirFailed         = "open_dir_failed"
	CodeImportCancelled       = "import_cancelled"
	CodeImportFailed          = "import_failed"
	CodeINPXNotFound          = "inpx_not_found"
	CodeLibraryUnreadable     = "library_unreadable"
	CodeOpenFileFailed        = "open_file_failed"
	CodeNotFound              = "not_found"
	CodeInvalidCatalogView    = "invalid_catalog_view"
	CodeLibraryOffline        = "library_offline"
	CodeLibraryUnreachable    = "library_unreachable"
	CodeArchiveMissing        = "archive_missing"
	CodeFB2Unreadable         = "fb2_unreadable"
	CodeReaderUnavailable     = "reader_unavailable"
	CodeReaderMissing         = "reader_missing"
	CodeDownloadsDirUnusable  = "downloads_dir_unusable"
	CodeCancelled             = "cancelled"
	CodeInvalidRating         = "invalid_rating"
	CodePersonalExportFailed  = "personal_export_failed"
	CodePersonalImportFailed  = "personal_import_failed"
	CodeSecretInvalidID       = "secret_invalid_id"
	CodeSecretEmpty           = "secret_empty"
	CodeSecretTooLarge        = "secret_too_large"
	CodeSecretStoreFailed     = "secret_store_failed"
	CodeSecretStoreUnreadable = "secret_store_unreadable"
	CodeInvalidAIProvider     = "invalid_ai_provider"
	CodeDBNoSpace             = "db_no_space"
	CodeDBMaintenanceBusy     = "db_maintenance_busy"
	CodeImportInProgress      = "import_in_progress"
	CodeCoverWarmupInProgress = "cover_warmup_in_progress"
	CodeDBOptimizeFailed      = "db_optimize_failed"
	CodeDiagFailed            = "diag_failed"
	CodeDiagArchiveFailed     = "diag_archive_failed"
)

// Error is a typed application error safe to send to the frontend.
type Error struct {
	Code    string            `json:"code"`
	Params  map[string]string `json:"params,omitempty"`
	cause   error
	private string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.private != "" {
		return e.private
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.Code
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// New creates a public error with the given code.
func New(code string, params map[string]string) *Error {
	return &Error{Code: code, Params: params}
}

// Wrap annotates a cause with a public code. The cause is for logs only.
func Wrap(code string, cause error, params map[string]string) *Error {
	return &Error{Code: code, Params: params, cause: cause, private: cause.Error()}
}

// Public is the JSON object returned to the frontend.
type Public struct {
	Code   string            `json:"code"`
	Params map[string]string `json:"params,omitempty"`
}

func (e *Error) Public() Public {
	if e == nil {
		return Public{Code: CodeInternal}
	}
	return Public{Code: e.Code, Params: e.Params}
}

// As extracts an *Error from err, wrapping unknown errors as internal.
func As(err error) *Error {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed
	}
	return Wrap(CodeInternal, err, nil)
}

// FormatWails is a Wails ErrorFormatter. The runtime turns the value into
// `new Error(string)`, so the payload must be a JSON string the frontend can parse.
func FormatWails(err error) any {
	if err == nil {
		return ""
	}
	payload, marshalErr := json.Marshal(As(err).Public())
	if marshalErr != nil {
		return fmt.Sprintf(`{"code":%q}`, CodeInternal)
	}
	return string(payload)
}
