//go:build darwin

// macOS で open によりパスを開く。
package app

func openPath(path string) error {
	return runOpenPath("open", path)
}
