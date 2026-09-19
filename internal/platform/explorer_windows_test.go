//go:build windows

package platform

import (
	"strings"
	"testing"
)

func TestExplorerSelectCmdLineQuotesSpaces(t *testing.T) {
	got := explorerSelectCmdLine(`C:\Users\Alex Books\War and Peace.fb2`)
	want := `explorer.exe /select,"C:\Users\Alex Books\War and Peace.fb2"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExplorerSelectCmdLineAlwaysQuotes(t *testing.T) {
	got := explorerSelectCmdLine(`C:\Users\alex\Downloads\book.fb2`)
	if !strings.HasPrefix(got, `explorer.exe /select,"`) || !strings.HasSuffix(got, `"`) {
		t.Fatalf("path must be quoted: %q", got)
	}
	if strings.Contains(got, `/select,C:`) {
		t.Fatalf("unquoted /select, path: %q", got)
	}
}

func TestExplorerOpenCmdLineQuotesSpaces(t *testing.T) {
	got := explorerOpenCmdLine(`C:\Users\Alex Books\Downloads`)
	want := `explorer.exe "C:\Users\Alex Books\Downloads"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
