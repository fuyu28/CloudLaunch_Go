//go:build linux

// Linux で xdg-open によりパスを開く。
package app

func openPath(path string) error {
	return runOpenPath("xdg-open", path)
}
