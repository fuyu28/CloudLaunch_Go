//go:build windows

// Windows でエクスプローラーによりパスを開く。
package app

func openPath(path string) error {
	return runOpenPath("explorer", path)
}
