package platform

import "testing"

func TestFitWindowKeepsVisibleGeometry(t *testing.T) {
	screens := []Rect{{X: 0, Y: 0, W: 1920, H: 1080}}
	got := FitWindow(Rect{X: 100, Y: 80, W: 1280, H: 840}, screens)
	if got.X != 100 || got.Y != 80 {
		t.Fatalf("visible window moved: %+v", got)
	}
}

func TestFitWindowRecentersWhenOffscreen(t *testing.T) {
	screens := []Rect{{X: 0, Y: 0, W: 1920, H: 1080}}
	got := FitWindow(Rect{X: 8000, Y: 8000, W: 1280, H: 840}, screens)
	if got.X == 8000 || got.Y == 8000 {
		t.Fatalf("expected recenter, got %+v", got)
	}
}

func TestFitWindowEnforcesMinimum(t *testing.T) {
	screens := []Rect{{X: 0, Y: 0, W: 1920, H: 1080}}
	got := FitWindow(Rect{W: 800, H: 400}, screens)
	if got.W < MinWidth || got.H < MinHeight {
		t.Fatalf("min size not applied: %+v", got)
	}
}
