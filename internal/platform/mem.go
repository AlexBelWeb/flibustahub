package platform

// ProcessPrivateBytes is the current process private working set, or 0 when
// the platform does not expose it.
func ProcessPrivateBytes() (uint64, error) {
	return processPrivateBytes()
}
