package platform

import (
	"runtime"
	"testing"
)

func TestResolveEffectsAuto(t *testing.T) {
	win := Capabilities{OS: "windows"}
	if got := ResolveEffects(EffectsAuto, win); got != EffectsFull {
		t.Fatalf("windows auto = %q", got)
	}
	linux := Capabilities{OS: "linux"}
	if got := ResolveEffects(EffectsAuto, linux); got != EffectsReduced {
		t.Fatalf("linux auto = %q", got)
	}
}

func TestDetectRespectsOverride(t *testing.T) {
	reduced := Detect(EffectsReduced)
	if reduced.EffectiveEffects != EffectsReduced || reduced.BackdropFilter {
		t.Fatalf("reduced override: %+v", reduced)
	}
	full := Detect(EffectsFull)
	if full.EffectiveEffects != EffectsFull {
		t.Fatalf("full override: %+v", full)
	}
	if runtime.GOOS == "linux" && full.FramelessOK {
		t.Fatal("linux must keep frameless off even in full effects")
	}
}
