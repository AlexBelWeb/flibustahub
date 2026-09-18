//go:build !windows && !unix

package platform

func acquireInstance(string) (func(), bool, error) {
	return func() {}, true, nil
}
