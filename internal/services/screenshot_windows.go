//go:build windows

package services

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	biRGB                           = 0
	dibRGBColors                    = 0
	srccopy                         = 0x00CC0020
	pwClientOnly                    = 0x00000001
	dwmExtendedFrameBounds          = 9
	dwmCloaked                      = 14
	gaRoot                          = 2
	gaRootOwner                     = 3
	wsChild                         = 0x40000000
	wsExToolWindow                  = 0x00000080
	dpiAwarenessContextPerMonitorV2 = ^uintptr(3) // -4
	smXVIRTUALSCREEN                = 76
	smYVIRTUALSCREEN                = 77
	smCXVIRTUALSCREEN               = 78
	smCYVIRTUALSCREEN               = 79
)

var (
	gwlStyle   int32 = -16
	gwlExStyle int32 = -20
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	gdi32                   = windows.NewLazySystemDLL("gdi32.dll")
	dwmapi                  = windows.NewLazySystemDLL("dwmapi.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procIsIconic            = user32.NewProc("IsIconic")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procGetClientRect       = user32.NewProc("GetClientRect")
	procClientToScreen      = user32.NewProc("ClientToScreen")
	procGetAncestor         = user32.NewProc("GetAncestor")
	procGetWindowLongPtr    = user32.NewProc("GetWindowLongPtrW")
	procGetClassNameW       = user32.NewProc("GetClassNameW")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procEnumChildWindows    = user32.NewProc("EnumChildWindows")
	procSetDpiAwarenessCtx  = user32.NewProc("SetProcessDpiAwarenessContext")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procGetDC               = user32.NewProc("GetDC")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowDC         = user32.NewProc("GetWindowDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procPrintWindow         = user32.NewProc("PrintWindow")
	procDwmGetWindowAttr    = dwmapi.NewProc("DwmGetWindowAttribute")
	procCreateCompatibleDC  = gdi32.NewProc("CreateCompatibleDC")
	procCreateBitmap        = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject        = gdi32.NewProc("SelectObject")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procDeleteDC            = gdi32.NewProc("DeleteDC")
	procBitBlt              = gdi32.NewProc("BitBlt")
	procGetDIBits           = gdi32.NewProc("GetDIBits")
)

type windowRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type point struct {
	X int32
	Y int32
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

var dpiOnce = sync.Once{}

func ensureDpiAwareness() {
	dpiOnce.Do(func() {
		procSetDpiAwarenessCtx.Call(dpiAwarenessContextPerMonitorV2)
	})
}

func captureWindowImageByPID(pid int, clientOnly bool) (image.Image, error) {
	ensureDpiAwareness()
	hwnd, err := findBestWindowForPID(uint32(pid))
	if err != nil {
		return nil, err
	}

	var failures []error
	if clientOnly {
		if img, err := captureWindowWithPrintWindow(hwnd, true); err == nil && img != nil {
			if !isMostlyBlack(img) {
				return img, nil
			}
			failures = append(failures, errors.New("PrintWindow(client) produced black image"))
		} else if err != nil {
			failures = append(failures, fmt.Errorf("PrintWindow(client) failed: %w", err))
		}

		if img, err := captureWindowWithPrintWindow(hwnd, false); err == nil && img != nil {
			if trimmed, trimErr := trimWithDwmBounds(hwnd, img); trimErr == nil && trimmed != nil {
				if !isMostlyBlack(trimmed) {
					return trimmed, nil
				}
				failures = append(failures, errors.New("PrintWindow(window) trimmed image is black"))
			} else if !isMostlyBlack(img) {
				return img, nil
			} else if trimErr != nil {
				failures = append(failures, fmt.Errorf("PrintWindow(window) trim failed: %w", trimErr))
			} else {
				failures = append(failures, errors.New("PrintWindow(window) produced black image"))
			}
		} else if err != nil {
			failures = append(failures, fmt.Errorf("PrintWindow(window) failed: %w", err))
		}
	} else {
		if img, err := captureWindowWithPrintWindow(hwnd, false); err == nil && img != nil {
			if !isMostlyBlack(img) {
				return img, nil
			}
			failures = append(failures, errors.New("PrintWindow(window) produced black image"))
		} else if err != nil {
			failures = append(failures, fmt.Errorf("PrintWindow(window) failed: %w", err))
		}
		if img, err := captureWindowWithPrintWindow(hwnd, true); err == nil && img != nil {
			if !isMostlyBlack(img) {
				return img, nil
			}
			failures = append(failures, errors.New("PrintWindow(client) produced black image"))
		} else if err != nil {
			failures = append(failures, fmt.Errorf("PrintWindow(client) failed: %w", err))
		}
	}

	if img, err := captureWindowClientWithBitBlt(hwnd); err == nil && img != nil {
		if !isMostlyBlack(img) {
			return img, nil
		}
		failures = append(failures, errors.New("BitBlt(client) produced black image"))
	} else if err != nil {
		failures = append(failures, fmt.Errorf("BitBlt(client) failed: %w", err))
	}

	if img, err := captureWindowClientFromScreen(hwnd); err == nil && img != nil {
		if !isMostlyBlack(img) {
			return img, nil
		}
		failures = append(failures, errors.New("Screen capture produced black image"))
	} else if err != nil {
		failures = append(failures, fmt.Errorf("Screen capture failed: %w", err))
	}

	if img, err := captureDesktopScreen(); err == nil && img != nil {
		if !isMostlyBlack(img) {
			return img, nil
		}
		failures = append(failures, errors.New("Desktop capture produced black image"))
	} else if err != nil {
		failures = append(failures, fmt.Errorf("Desktop capture failed: %w", err))
	}

	if len(failures) == 0 {
		return nil, errors.New("failed to capture window")
	}
	return nil, errors.Join(failures...)
}

func captureWindowWithWGC(pid int, outputPath string, clientOnly bool) (bool, error) {
	ensureDpiAwareness()
	helperPath, err := wgcHelperPath()
	if err != nil || helperPath == "" {
		return false, nil
	}

	if fg := foregroundWindowCandidate(uint32(pid)); fg != 0 {
		if ok, err := captureWGCByHWND(fg, outputPath, clientOnly, helperPath); ok {
			return true, nil
		} else if err != nil {
			// fall through with diagnostics below
		}
	}

	candidates := listWindowCandidates(uint32(pid), true, true, true)
	if len(candidates) == 0 {
		candidates = listWindowCandidates(uint32(pid), false, true, true)
	}
	if len(candidates) == 0 {
		return false, errors.New("window not found")
	}
	fgCandidate := foregroundWindowCandidate(uint32(pid))
	if fgCandidate != 0 {
		candidates = prependUniqueWindow(candidates, fgCandidate)
	}
	metrics := rankWindowMetrics(buildCandidateMetrics(candidates))
	candidateInfos := describeMetrics(metrics)
	childInfos := describeChildWindows(metrics)
	childCandidates := collectChildCandidates(metrics)
	if len(childCandidates) > 0 {
		candidates = prependUniqueWindows(candidates, childCandidates)
		metrics = rankWindowMetrics(buildCandidateMetrics(candidates))
	}
	foregroundInfo := describeForegroundCandidate(uint32(pid))

	var failures []error
	for _, m := range metrics {
		hwnd := normalizeRootWindow(m.hwnd)
		if m.isCloaked || !m.visible || m.iconic || m.area <= 0 {
			continue
		}
		if m.isToolWindow {
			// WGCが拒否することが多いのでスキップし、候補情報はログに残す
			continue
		}

		if ok, err := captureWGCByHWND(hwnd, outputPath, clientOnly, helperPath); ok {
			return true, nil
		} else if err != nil {
			failures = append(failures, fmt.Errorf("hwnd=%d: %w", hwnd, err))
		}
	}

	if len(failures) == 0 {
		extra := strings.TrimSpace(strings.Join([]string{foregroundInfo, candidateInfos, childInfos}, " | "))
		if extra != "" {
			return false, fmt.Errorf("WGC capture failed; %s", extra)
		}
		return false, errors.New("WGC capture failed")
	}
	joined := errors.Join(failures...)
	extra := strings.TrimSpace(strings.Join([]string{foregroundInfo, candidateInfos, childInfos}, " | "))
	if extra != "" {
		return false, fmt.Errorf("%w; %s", joined, extra)
	}
	return false, joined
}

func wgcHelperPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "wgc_screenshot.exe"), nil
}

func captureWGCByHWND(hwnd windows.Handle, outputPath string, clientOnly bool, helperPath string) (bool, error) {
	if hwnd == 0 {
		return false, errors.New("hwnd is zero")
	}
	args := []string{
		"--hwnd",
		strconv.FormatUint(uint64(hwnd), 10),
		"--out",
		outputPath,
	}
	if clientOnly {
		args = append(args, "--client-only")
	}

	command := execCommandHidden(context.Background(), helperPath, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return false, fmt.Errorf("%w: %s", err, message)
		}
		return false, err
	}
	return true, nil
}

func normalizeRootWindow(hwnd windows.Handle) windows.Handle {
	if hwnd == 0 {
		return hwnd
	}
	rootOwner, _, _ := procGetAncestor.Call(uintptr(hwnd), uintptr(gaRootOwner))
	if rootOwner != 0 {
		hwnd = windows.Handle(rootOwner)
	}
	root, _, _ := procGetAncestor.Call(uintptr(hwnd), uintptr(gaRoot))
	if root == 0 {
		return hwnd
	}
	var rootPID uint32
	procGetWindowThreadPID.Call(root, uintptr(unsafe.Pointer(&rootPID)))
	var hwndPID uint32
	procGetWindowThreadPID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&hwndPID)))
	if rootPID == hwndPID {
		return windows.Handle(root)
	}
	return hwnd
}

func isWindowCloaked(hwnd windows.Handle) bool {
	if hwnd == 0 {
		return false
	}
	var cloaked uint32
	if ret, _, _ := procDwmGetWindowAttr.Call(
		uintptr(hwnd),
		uintptr(dwmCloaked),
		uintptr(unsafe.Pointer(&cloaked)),
		unsafe.Sizeof(cloaked),
	); ret == 0 {
		return cloaked != 0
	}
	return false
}

func captureWindowWithPrintWindow(hwnd windows.Handle, clientOnly bool) (image.Image, error) {
	var rect windowRect
	if clientOnly {
		if ret, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect))); ret == 0 {
			return nil, errors.New("failed to get client rect")
		}
	} else {
		if ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect))); ret == 0 {
			return nil, errors.New("failed to get window rect")
		}
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return nil, errors.New("invalid window size")
	}

	hdcWindow, _, _ := procGetWindowDC.Call(uintptr(hwnd))
	if hdcWindow == 0 {
		return nil, errors.New("failed to get window DC")
	}
	defer procReleaseDC.Call(uintptr(hwnd), hdcWindow)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcWindow)
	if hdcMem == 0 {
		return nil, errors.New("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	hBitmap, _, _ := procCreateBitmap.Call(hdcWindow, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, errors.New("failed to create bitmap")
	}
	defer procDeleteObject.Call(hBitmap)

	oldObj, _, _ := procSelectObject.Call(hdcMem, hBitmap)
	defer procSelectObject.Call(hdcMem, oldObj)

	flags := uintptr(0)
	if clientOnly {
		flags = pwClientOnly
	}
	printResult, _, _ := procPrintWindow.Call(uintptr(hwnd), hdcMem, flags)
	if printResult == 0 {
		return nil, errors.New("PrintWindow failed")
	}

	return bitmapToImage(hdcMem, hBitmap, width, height)
}

func captureWindowClientWithBitBlt(hwnd windows.Handle) (image.Image, error) {
	var rect windowRect
	if ret, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect))); ret == 0 {
		return nil, errors.New("failed to get client rect")
	}
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return nil, errors.New("invalid client size")
	}

	hdcWindow, _, _ := procGetDC.Call(uintptr(hwnd))
	if hdcWindow == 0 {
		return nil, errors.New("failed to get client DC")
	}
	defer procReleaseDC.Call(uintptr(hwnd), hdcWindow)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcWindow)
	if hdcMem == 0 {
		return nil, errors.New("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	hBitmap, _, _ := procCreateBitmap.Call(hdcWindow, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, errors.New("failed to create bitmap")
	}
	defer procDeleteObject.Call(hBitmap)

	oldObj, _, _ := procSelectObject.Call(hdcMem, hBitmap)
	defer procSelectObject.Call(hdcMem, oldObj)

	bitbltResult, _, _ := procBitBlt.Call(
		hdcMem,
		0,
		0,
		uintptr(width),
		uintptr(height),
		hdcWindow,
		0,
		0,
		srccopy,
	)
	if bitbltResult == 0 {
		return nil, errors.New("failed to capture client")
	}

	return bitmapToImage(hdcMem, hBitmap, width, height)
}

func captureWindowClientFromScreen(hwnd windows.Handle) (image.Image, error) {
	screenRect, err := getClientRectOnScreen(hwnd)
	if err != nil {
		return nil, err
	}
	width := int(screenRect.Right - screenRect.Left)
	height := int(screenRect.Bottom - screenRect.Top)
	if width <= 0 || height <= 0 {
		return nil, errors.New("invalid screen rect")
	}

	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, errors.New("failed to get screen DC")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, errors.New("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	hBitmap, _, _ := procCreateBitmap.Call(hdcScreen, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, errors.New("failed to create bitmap")
	}
	defer procDeleteObject.Call(hBitmap)

	oldObj, _, _ := procSelectObject.Call(hdcMem, hBitmap)
	defer procSelectObject.Call(hdcMem, oldObj)

	bitbltResult, _, _ := procBitBlt.Call(
		hdcMem,
		0,
		0,
		uintptr(width),
		uintptr(height),
		hdcScreen,
		uintptr(screenRect.Left),
		uintptr(screenRect.Top),
		srccopy,
	)
	if bitbltResult == 0 {
		return nil, errors.New("failed to capture screen")
	}

	return bitmapToImage(hdcMem, hBitmap, width, height)
}

func captureDesktopScreen() (image.Image, error) {
	left := int32(getSystemMetrics(smXVIRTUALSCREEN))
	top := int32(getSystemMetrics(smYVIRTUALSCREEN))
	width := int32(getSystemMetrics(smCXVIRTUALSCREEN))
	height := int32(getSystemMetrics(smCYVIRTUALSCREEN))
	if width <= 0 || height <= 0 {
		return nil, errors.New("invalid desktop size")
	}

	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, errors.New("failed to get screen DC")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, errors.New("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	hBitmap, _, _ := procCreateBitmap.Call(hdcScreen, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, errors.New("failed to create bitmap")
	}
	defer procDeleteObject.Call(hBitmap)

	oldObj, _, _ := procSelectObject.Call(hdcMem, hBitmap)
	defer procSelectObject.Call(hdcMem, oldObj)

	bitbltResult, _, _ := procBitBlt.Call(
		hdcMem,
		0,
		0,
		uintptr(width),
		uintptr(height),
		hdcScreen,
		uintptr(left),
		uintptr(top),
		srccopy,
	)
	if bitbltResult == 0 {
		return nil, errors.New("failed to capture desktop")
	}

	return bitmapToImage(hdcMem, hBitmap, int(width), int(height))
}

func getSystemMetrics(index int32) int32 {
	ret, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int32(ret)
}

func bitmapToImage(hdcMem uintptr, hBitmap uintptr, width int, height int) (image.Image, error) {
	bmi := bitmapInfo{}
	bmi.Header.Size = uint32(unsafe.Sizeof(bmi.Header))
	bmi.Header.Width = int32(width)
	bmi.Header.Height = -int32(height)
	bmi.Header.Planes = 1
	bmi.Header.BitCount = 32
	bmi.Header.Compression = biRGB

	buf := make([]byte, width*height*4)
	ret, _, _ := procGetDIBits.Call(
		hdcMem,
		hBitmap,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bmi)),
		dibRGBColors,
	)
	if ret == 0 {
		return nil, errors.New("failed to read bitmap")
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(buf); i += 4 {
		b := buf[i]
		g := buf[i+1]
		r := buf[i+2]
		a := buf[i+3]
		img.Pix[i] = r
		img.Pix[i+1] = g
		img.Pix[i+2] = b
		img.Pix[i+3] = a
	}
	return img, nil
}

func trimWithDwmBounds(hwnd windows.Handle, img image.Image) (image.Image, error) {
	var frame windowRect
	ret, _, _ := procDwmGetWindowAttr.Call(
		uintptr(hwnd),
		uintptr(dwmExtendedFrameBounds),
		uintptr(unsafe.Pointer(&frame)),
		unsafe.Sizeof(frame),
	)
	if ret != 0 {
		return nil, errors.New("failed to get DWM bounds")
	}

	var window windowRect
	if ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&window))); ret == 0 {
		return nil, errors.New("failed to get window rect")
	}

	left := int(frame.Left - window.Left)
	top := int(frame.Top - window.Top)
	right := int(frame.Right - window.Left)
	bottom := int(frame.Bottom - window.Top)

	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	bounds := img.Bounds()
	if right > bounds.Dx() {
		right = bounds.Dx()
	}
	if bottom > bounds.Dy() {
		bottom = bounds.Dy()
	}
	if right <= left || bottom <= top {
		return nil, errors.New("invalid DWM bounds")
	}

	crop := image.Rect(left, top, right, bottom)
	out := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	for y := 0; y < crop.Dy(); y++ {
		for x := 0; x < crop.Dx(); x++ {
			out.Set(x, y, img.At(crop.Min.X+x, crop.Min.Y+y))
		}
	}
	return out, nil
}

func getClientRectOnScreen(hwnd windows.Handle) (windowRect, error) {
	var client windowRect
	if ret, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&client))); ret == 0 {
		return windowRect{}, errors.New("failed to get client rect")
	}
	topLeft := point{X: 0, Y: 0}
	bottomRight := point{X: client.Right, Y: client.Bottom}
	if ret, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&topLeft))); ret == 0 {
		return windowRect{}, errors.New("failed to translate client top-left")
	}
	if ret, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&bottomRight))); ret == 0 {
		return windowRect{}, errors.New("failed to translate client bottom-right")
	}
	return windowRect{
		Left:   topLeft.X,
		Top:    topLeft.Y,
		Right:  bottomRight.X,
		Bottom: bottomRight.Y,
	}, nil
}

func isMostlyBlack(img image.Image) bool {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return true
	}

	samplesX := 10
	samplesY := 10
	if width < samplesX {
		samplesX = width
	}
	if height < samplesY {
		samplesY = height
	}

	blackCount := 0
	total := samplesX * samplesY
	for y := 0; y < samplesY; y++ {
		for x := 0; x < samplesX; x++ {
			px := bounds.Min.X + x*(width-1)/max(1, samplesX-1)
			py := bounds.Min.Y + y*(height-1)/max(1, samplesY-1)
			r, g, b, a := img.At(px, py).RGBA()
			if a > 0 && r < 0x0100 && g < 0x0100 && b < 0x0100 {
				blackCount++
			}
		}
	}
	return blackCount*100/total >= 95
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func findBestWindowForPID(pid uint32) (windows.Handle, error) {
	var best windows.Handle
	var bestArea int32

	selectBest := func(requireVisible bool) windows.Handle {
		best = 0
		bestArea = 0
		callback := windows.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
			var windowPID uint32
			procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
			if windowPID != pid {
				return 1
			}
			if requireVisible {
				visible, _, _ := procIsWindowVisible.Call(hwnd)
				if visible == 0 {
					return 1
				}
				iconic, _, _ := procIsIconic.Call(hwnd)
				if iconic != 0 {
					return 1
				}
			}
			var rect windowRect
			if ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); ret == 0 {
				return 1
			}
			width := rect.Right - rect.Left
			height := rect.Bottom - rect.Top
			if width <= 0 || height <= 0 {
				return 1
			}
			area := width * height
			if area > bestArea {
				bestArea = area
				best = windows.Handle(hwnd)
			}
			return 1
		})
		procEnumWindows.Call(callback, 0)
		return best
	}

	if handle := selectBest(true); handle != 0 {
		return handle, nil
	}
	if handle := selectBest(false); handle != 0 {
		return handle, nil
	}

	fg, _, _ := procGetForegroundWindow.Call()
	if fg != 0 {
		var windowPID uint32
		procGetWindowThreadPID.Call(fg, uintptr(unsafe.Pointer(&windowPID)))
		if windowPID == pid {
			return windows.Handle(fg), nil
		}
	}

	return 0, errors.New("window not found")
}

func listWindowCandidates(pid uint32, requireVisible bool, allowToolWindow bool, allowChild bool) []windows.Handle {
	handles := make([]windows.Handle, 0, 4)
	seen := map[windows.Handle]bool{}

	callback := windows.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var windowPID uint32
		procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if windowPID != pid {
			return 1
		}
		if requireVisible {
			visible, _, _ := procIsWindowVisible.Call(hwnd)
			if visible == 0 {
				return 1
			}
			iconic, _, _ := procIsIconic.Call(hwnd)
			if iconic != 0 {
				return 1
			}
		}
		style, _, _ := procGetWindowLongPtr.Call(hwnd, uintptr(int64(gwlStyle)))
		if !allowChild && style&wsChild != 0 {
			return 1
		}
		exStyle, _, _ := procGetWindowLongPtr.Call(hwnd, uintptr(int64(gwlExStyle)))
		if !allowToolWindow && exStyle&wsExToolWindow != 0 {
			return 1
		}
		var rect windowRect
		if ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); ret == 0 {
			return 1
		}
		width := rect.Right - rect.Left
		height := rect.Bottom - rect.Top
		if width <= 0 || height <= 0 {
			return 1
		}
		handle := windows.Handle(hwnd)
		if !seen[handle] {
			seen[handle] = true
			handles = append(handles, handle)
		}
		return 1
	})
	procEnumWindows.Call(callback, 0)
	return handles
}

type candidateMetrics struct {
	hwnd         windows.Handle
	visible      bool
	iconic       bool
	isRoot       bool
	isCloaked    bool
	isToolWindow bool
	isChild      bool
	area         int32
	width        int32
	height       int32
}

func rankWindowMetrics(metrics []candidateMetrics) []candidateMetrics {
	if len(metrics) <= 1 {
		return metrics
	}

	rankValue := func(m candidateMetrics) int64 {
		var score int64
		if m.visible {
			score += 1_000_000
		}
		if m.isRoot {
			score += 100_000
		}
		if !m.iconic {
			score += 10_000
		}
		if !m.isCloaked {
			score += 1_000
		}
		if !m.isToolWindow {
			score += 500
		}
		if !m.isChild {
			score += 500
		}
		if m.area >= 200*200 {
			score += 100
		}
		score += int64(m.area)
		return score
	}

	for i := 0; i < len(metrics); i++ {
		for j := i + 1; j < len(metrics); j++ {
			if rankValue(metrics[j]) > rankValue(metrics[i]) {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}

	return metrics
}

func rankWindowCandidates(handles []windows.Handle) []windows.Handle {
	if len(handles) <= 1 {
		return handles
	}

	metrics := buildCandidateMetrics(handles)
	if len(metrics) == 0 {
		return handles
	}
	metrics = rankWindowMetrics(metrics)

	ordered := make([]windows.Handle, 0, len(metrics))
	for _, m := range metrics {
		ordered = append(ordered, m.hwnd)
	}
	return ordered
}

func rankPidsForCapture(pids []int) []int {
	if len(pids) <= 1 {
		return pids
	}

	type pidMetrics struct {
		pid         int
		visibleRoot bool
		notIconic   bool
		notCloaked  bool
		area        int32
	}

	metrics := make([]pidMetrics, 0, len(pids))
	for _, pid := range pids {
		candidates := listWindowCandidates(uint32(pid), true, true, true)
		if len(candidates) == 0 {
			candidates = listWindowCandidates(uint32(pid), false, true, true)
		}
		ordered := rankWindowCandidates(candidates)
		best := windows.Handle(0)
		if len(ordered) > 0 {
			best = ordered[0]
		}
		m := pidMetrics{pid: pid}
		if best != 0 {
			visible, _, _ := procIsWindowVisible.Call(uintptr(best))
			iconic, _, _ := procIsIconic.Call(uintptr(best))
			root, _, _ := procGetAncestor.Call(uintptr(best), uintptr(gaRoot))
			m.visibleRoot = visible != 0 && root != 0 && root == uintptr(best)
			m.notIconic = iconic == 0
			m.notCloaked = !isWindowCloaked(best)
			var rect windowRect
			if ret, _, _ := procGetWindowRect.Call(uintptr(best), uintptr(unsafe.Pointer(&rect))); ret != 0 {
				width := rect.Right - rect.Left
				height := rect.Bottom - rect.Top
				if width > 0 && height > 0 {
					m.area = width * height
				}
			}
		}
		metrics = append(metrics, m)
	}

	rankValue := func(m pidMetrics) int64 {
		var score int64
		if m.visibleRoot {
			score += 1_000_000
		}
		if m.notIconic {
			score += 10_000
		}
		if m.notCloaked {
			score += 1_000
		}
		if m.area >= 200*200 {
			score += 100
		}
		score += int64(m.area)
		return score
	}

	for i := 0; i < len(metrics); i++ {
		for j := i + 1; j < len(metrics); j++ {
			if rankValue(metrics[j]) > rankValue(metrics[i]) {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}

	ordered := make([]int, 0, len(metrics))
	for _, m := range metrics {
		ordered = append(ordered, m.pid)
	}
	return ordered
}

func buildCandidateMetrics(handles []windows.Handle) []candidateMetrics {
	metrics := make([]candidateMetrics, 0, len(handles))
	for _, hwnd := range handles {
		if hwnd == 0 {
			continue
		}
		visible, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		iconic, _, _ := procIsIconic.Call(uintptr(hwnd))
		root, _, _ := procGetAncestor.Call(uintptr(hwnd), uintptr(gaRoot))
		isRoot := root != 0 && root == uintptr(hwnd)
		isCloaked := isWindowCloaked(hwnd)
		style, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), uintptr(int64(gwlStyle)))
		exStyle, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), uintptr(int64(gwlExStyle)))
		var rect windowRect
		var area int32
		if ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect))); ret != 0 {
			width := rect.Right - rect.Left
			height := rect.Bottom - rect.Top
			if width > 0 && height > 0 {
				area = width * height
			}
		}
		metrics = append(metrics, candidateMetrics{
			hwnd:         hwnd,
			visible:      visible != 0,
			iconic:       iconic != 0,
			isRoot:       isRoot,
			isCloaked:    isCloaked,
			isToolWindow: exStyle&wsExToolWindow != 0,
			isChild:      style&wsChild != 0,
			area:         area,
			width:        rect.Right - rect.Left,
			height:       rect.Bottom - rect.Top,
		})
	}
	return metrics
}

func describeMetrics(metrics []candidateMetrics) string {
	if len(metrics) == 0 {
		return ""
	}
	parts := make([]string, 0, len(metrics))
	for _, m := range metrics {
		className := getWindowClassName(m.hwnd)
		title := getWindowTitle(m.hwnd)
		parts = append(parts, fmt.Sprintf(
			"hwnd=%d cls=%q title=%q vis=%t root=%t iconic=%t cloaked=%t tool=%t child=%t area=%d (%dx%d)",
			m.hwnd, className, title, m.visible, m.isRoot, m.iconic, m.isCloaked, m.isToolWindow, m.isChild, m.area, m.width, m.height,
		))
	}
	return "candidates: " + strings.Join(parts, " | ")
}

func describeChildWindows(metrics []candidateMetrics) string {
	if len(metrics) == 0 {
		return ""
	}
	parts := make([]string, 0, len(metrics))
	for _, m := range metrics {
		children := listChildCandidates(m.hwnd, true)
		if len(children) == 0 {
			continue
		}
		childParts := make([]string, 0, len(children))
		for _, c := range children {
			childParts = append(childParts, describeWindowSummary(c))
		}
		parts = append(parts, fmt.Sprintf("children(hwnd=%d): %s", m.hwnd, strings.Join(childParts, " | ")))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " | ")
}

func describeForegroundCandidate(pid uint32) string {
	hwnd := foregroundWindowCandidate(pid)
	if hwnd == 0 {
		return ""
	}
	return "foreground: " + describeWindowSummary(hwnd)
}

func describeWindowSummary(hwnd windows.Handle) string {
	if hwnd == 0 {
		return "hwnd=0"
	}
	className := getWindowClassName(hwnd)
	title := getWindowTitle(hwnd)
	visible, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	iconic, _, _ := procIsIconic.Call(uintptr(hwnd))
	var rect windowRect
	var width, height int32
	if ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect))); ret != 0 {
		width = rect.Right - rect.Left
		height = rect.Bottom - rect.Top
	}
	return fmt.Sprintf("hwnd=%d cls=%q title=%q vis=%t iconic=%t size=%dx%d",
		hwnd, className, title, visible != 0, iconic != 0, width, height)
}

func getWindowClassName(hwnd windows.Handle) string {
	buf := make([]uint16, 256)
	ret, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:ret])
}

func getWindowTitle(hwnd windows.Handle) string {
	buf := make([]uint16, 512)
	ret, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:ret])
}

func listChildCandidates(hwnd windows.Handle, includeHidden bool) []windows.Handle {
	if hwnd == 0 {
		return nil
	}
	children := make([]windows.Handle, 0, 6)
	seen := map[windows.Handle]bool{}
	callback := windows.NewCallback(func(child uintptr, lparam uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(child)
		if !includeHidden && visible == 0 {
			return 1
		}
		var rect windowRect
		if ret, _, _ := procGetWindowRect.Call(child, uintptr(unsafe.Pointer(&rect))); ret == 0 {
			return 1
		}
		width := rect.Right - rect.Left
		height := rect.Bottom - rect.Top
		if width < 200 || height < 200 {
			return 1
		}
		handle := windows.Handle(child)
		if !seen[handle] {
			seen[handle] = true
			children = append(children, handle)
		}
		return 1
	})
	procEnumChildWindows.Call(uintptr(hwnd), callback, 0)
	return children
}

func collectChildCandidates(metrics []candidateMetrics) []windows.Handle {
	children := make([]windows.Handle, 0, 8)
	for _, m := range metrics {
		for _, child := range listChildCandidates(m.hwnd, false) {
			children = append(children, child)
		}
	}
	return children
}

func prependUniqueWindows(handles []windows.Handle, extras []windows.Handle) []windows.Handle {
	if len(extras) == 0 {
		return handles
	}
	seen := map[windows.Handle]bool{}
	for _, h := range handles {
		seen[h] = true
	}
	ordered := make([]windows.Handle, 0, len(extras)+len(handles))
	for _, h := range extras {
		if h == 0 || seen[h] {
			continue
		}
		seen[h] = true
		ordered = append(ordered, h)
	}
	ordered = append(ordered, handles...)
	return ordered
}

func prependUniqueWindow(handles []windows.Handle, hwnd windows.Handle) []windows.Handle {
	if hwnd == 0 {
		return handles
	}
	for _, h := range handles {
		if h == hwnd {
			return handles
		}
	}
	return append([]windows.Handle{hwnd}, handles...)
}

func foregroundWindowCandidate(pid uint32) windows.Handle {
	fg, _, _ := procGetForegroundWindow.Call()
	if fg == 0 {
		return 0
	}
	var fgPID uint32
	procGetWindowThreadPID.Call(fg, uintptr(unsafe.Pointer(&fgPID)))
	if fgPID != pid {
		return 0
	}
	return windows.Handle(fg)
}
