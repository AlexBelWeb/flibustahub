package platform

import (
	"testing"
)

func TestAcquireInstanceExclusive(t *testing.T) {
	dir := t.TempDir()
	release, primary, err := AcquireInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !primary {
		t.Fatal("first process must own the instance lock")
	}
	t.Cleanup(release)

	_, second, err := AcquireInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if second {
		t.Fatal("second process must not own the instance lock")
	}

	other, otherPrimary, err := AcquireInstance(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(other)
	if !otherPrimary {
		t.Fatal("a different dataDir must get its own lock")
	}
}
