// Package apperr defines typed backend errors.
// Public payloads contain a stable code plus parameters; UI text lives in frontend locales.
package apperr

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	CodeInternal          = "internal"
	CodeConfigUnreadable  = "config_unreadable"
	CodeConfigWriteFailed = "config_write_failed"
	CodeInvalidLocale     = "invalid_locale"
	CodeInvalidTheme      = "invalid_theme"
	CodeInvalidEffects    = "invalid_visual_effects"
	CodeHTTPPortInUse     = "http_port_in_use"
	CodeOpenDirFailed     = "open_dir_failed"
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
