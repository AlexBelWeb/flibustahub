package platform

// WebViewVersion is the installed WebView2 version, or "unknown".
func WebViewVersion() string {
	v := webViewVersion()
	if v == "" {
		return "unknown"
	}
	return v
}
