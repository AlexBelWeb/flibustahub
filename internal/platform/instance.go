package platform

// AcquireInstance claims the single-process lock for dataDir.
// release is always non-nil and safe to defer. primary is true only for the
// first live process; a second process must not open the catalog or bind HTTP.
func AcquireInstance(dataDir string) (release func(), primary bool, err error) {
	return acquireInstance(dataDir)
}
