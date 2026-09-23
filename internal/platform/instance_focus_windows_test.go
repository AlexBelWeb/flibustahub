//go:build windows

package platform

import (
	"testing"
	"unsafe"
)

func TestFocusWindowStructLayout(t *testing.T) {
	if unsafe.Sizeof(wndClassExW{}) != 80 {
		t.Fatalf("WNDCLASSEXW size = %d, want 80", unsafe.Sizeof(wndClassExW{}))
	}
	if unsafe.Sizeof(winMSG{}) != 48 {
		t.Fatalf("MSG size = %d, want 48", unsafe.Sizeof(winMSG{}))
	}
}
