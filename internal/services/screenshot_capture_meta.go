// @fileoverview DXGIキャプチャの共通メタ情報。
package services

// CaptureMeta はキャプチャ時の付随情報を保持する。
type CaptureMeta struct {
	HWND       uintptr
	CropX      int
	CropY      int
	CropW      int
	CropH      int
	Monitor    int
	DXGIStdout string
	DXGIStderr string
}
