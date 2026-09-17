package platform

// Rect is a window or screen rectangle in logical pixels.
type Rect struct {
	X, Y, W, H int
}

const (
	DefaultWidth  = 1280
	DefaultHeight = 840
	MinWidth      = 1024
	MinHeight     = 640
)

// FitWindow keeps the window on-screen. Screens without origin (Wails v2)
// are treated as sitting at (0,0) with their logical size; if the saved
// geometry cannot fit any of them, the window is centered on the first screen.
func FitWindow(win Rect, screens []Rect) Rect {
	if win.W < MinWidth {
		win.W = MinWidth
	}
	if win.H < MinHeight {
		win.H = MinHeight
	}
	if len(screens) == 0 {
		if win.W == 0 {
			win.W = DefaultWidth
		}
		if win.H == 0 {
			win.H = DefaultHeight
		}
		return win
	}
	if win.W == 0 || win.H == 0 {
		win.W = DefaultWidth
		win.H = DefaultHeight
		return centerOn(screens[0], win)
	}
	if intersectsAny(win, screens) {
		return clampSize(win, screens)
	}
	return centerOn(screens[0], clampSize(win, screens))
}

func clampSize(win Rect, screens []Rect) Rect {
	maxW, maxH := screens[0].W, screens[0].H
	for _, s := range screens {
		if s.W > maxW {
			maxW = s.W
		}
		if s.H > maxH {
			maxH = s.H
		}
	}
	if win.W > maxW {
		win.W = maxW
	}
	if win.H > maxH {
		win.H = maxH
	}
	if win.W < MinWidth && maxW >= MinWidth {
		win.W = MinWidth
	}
	if win.H < MinHeight && maxH >= MinHeight {
		win.H = MinHeight
	}
	return win
}

func intersectsAny(win Rect, screens []Rect) bool {
	for _, s := range screens {
		if intersects(win, s) {
			return true
		}
	}
	return false
}

func intersects(a, b Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

func centerOn(screen, win Rect) Rect {
	win.X = screen.X + (screen.W-win.W)/2
	win.Y = screen.Y + (screen.H-win.H)/2
	if win.X < screen.X {
		win.X = screen.X
	}
	if win.Y < screen.Y {
		win.Y = screen.Y
	}
	return win
}
