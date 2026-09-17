// Package platform reports OS capabilities and restores the window safely.
package platform

import "runtime"

const (
	EffectsFull    = "full"
	EffectsReduced = "reduced"
	EffectsAuto    = "auto"
)

// Capabilities describes what the current WebView can do without breaking layout.
type Capabilities struct {
	OS               string `json:"os"`
	BackdropFilter   bool   `json:"backdropFilter"`
	FramelessOK      bool   `json:"framelessOk"`
	EffectiveEffects string `json:"effectiveEffects"`
}

// Detect returns the default capability set. Linux uses the reduced visual plan.
func Detect(pref string) Capabilities {
	caps := Capabilities{OS: runtime.GOOS}
	switch runtime.GOOS {
	case "windows":
		caps.BackdropFilter = true
		caps.FramelessOK = true
	default:
		caps.BackdropFilter = false
		caps.FramelessOK = false
	}
	caps.EffectiveEffects = ResolveEffects(pref, caps)
	if caps.EffectiveEffects == EffectsReduced {
		caps.BackdropFilter = false
		caps.FramelessOK = false
	}
	if caps.EffectiveEffects == EffectsFull {
		caps.BackdropFilter = true
		// Frameless on Linux stays false: Wayland drag is unreliable.
		if runtime.GOOS != "linux" {
			caps.FramelessOK = true
		}
	}
	return caps
}

// ResolveEffects maps auto/full/reduced to the effective mode.
func ResolveEffects(pref string, caps Capabilities) string {
	switch pref {
	case EffectsFull:
		return EffectsFull
	case EffectsReduced:
		return EffectsReduced
	default:
		if caps.OS == "windows" {
			return EffectsFull
		}
		return EffectsReduced
	}
}
