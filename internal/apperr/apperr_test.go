package apperr

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAsTyped(t *testing.T) {
	err := New(CodeInvalidLocale, map[string]string{"locale": "de"})
	got := As(err)
	if got.Code != CodeInvalidLocale || got.Params["locale"] != "de" {
		t.Fatalf("unexpected public error: %+v", got)
	}
}

func TestAsUnknownIsInternal(t *testing.T) {
	got := As(errors.New("boom"))
	if got.Code != CodeInternal {
		t.Fatalf("code = %q, want %q", got.Code, CodeInternal)
	}
}

func TestFormatWailsJSONString(t *testing.T) {
	raw, ok := FormatWails(New(CodeHTTPPortInUse, map[string]string{"port": "8787"})).(string)
	if !ok {
		t.Fatal("formatter must return a JSON string")
	}
	var pub Public
	if err := json.Unmarshal([]byte(raw), &pub); err != nil {
		t.Fatal(err)
	}
	if pub.Code != CodeHTTPPortInUse || pub.Params["port"] != "8787" {
		t.Fatalf("payload = %+v", pub)
	}
}

func TestFormatWailsNil(t *testing.T) {
	if FormatWails(nil) != "" {
		t.Fatal("nil error should format as empty string")
	}
}
