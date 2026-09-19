//go:build !windows && !linux

package platform

func identifyVolume(string) (string, string) { return "", "" }

func findVolumePath(string, string) (string, bool, bool) { return "", false, false }
