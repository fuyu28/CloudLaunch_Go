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
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	biRGB                  = 0
	dibRGBColors           = 0
	srccopy                = 0x00CC0020
	pwClientOnly           = 0x00000001
	dwmExtendedFrameBounds = 9
	dwmCloaked             = 14
	gaRoot                 = 2
	gaRootOwner            = 3
	gwlStyle               = -16
	gwlExStyle             = -20
	wsChild                = 0x40000000
	wsExToolWindow         = 0x00000080
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

func captureWindowImageByPID(pid int, clientOnly bool) (image.Image, error) {
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

	if len(failures) == 0 {
		return nil, errors.New("failed to capture window")
	}
	return nil, errors.Join(failures...)
}

func captureWindowWithWGC(pid int, outputPath string, clientOnly bool) (bool, error) {
	helperPath, err := wgcHelperPath()
	if err != nil || helperPath == "" {
		return false, nil
	}

	candidates := listWindowCandidates(uint32(pid), true)
	if len(candidates) == 0 {
		candidates = listWindowCandidates(uint32(pid), false)
	}
	if len(candidates) == 0 {
		return false, errors.New("window not found")
	}
	candidates = rankWindowCandidates(candidates)

	var failures []error
	for _, hwnd := range candidates {
		hwnd = normalizeRootWindow(hwnd)
		if isWindowCloaked(hwnd) {
			failures = append(failures, fmt.Errorf("hwnd=%d is cloaked", hwnd))
			continue
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
				failures = append(failures, fmt.Errorf("hwnd=%d: %w: %s", hwnd, err, message))
			} else {
				failures = append(failures, fmt.Errorf("hwnd=%d: %w", hwnd, err))
			}
			continue
		}
		return true, nil
	}

	if len(failures) == 0 {
		return false, errors.New("WGC capture failed")
	}
	return false, errors.Join(failures...)
}

func wgcHelperPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "wgc_screenshot.exe"), nil
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

func listWindowCandidates(pid uint32, requireVisible bool) []windows.Handle {
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
		style, _, _ := procGetWindowLongPtr.Call(hwnd, uintptr(gwlStyle))
		if style&wsChild != 0 {
			return 1
		}
		exStyle, _, _ := procGetWindowLongPtr.Call(hwnd, uintptr(gwlExStyle))
		if exStyle&wsExToolWindow != 0 {
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
	hwnd      windows.Handle
	visible   bool
	iconic    bool
	isRoot    bool
	isCloaked bool
	area      int32
}

func rankWindowCandidates(handles []windows.Handle) []windows.Handle {
	if len(handles) <= 1 {
		return handles
	}

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
			hwnd:      hwnd,
			visible:   visible != 0,
			iconic:    iconic != 0,
			isRoot:    isRoot,
			isCloaked: isCloaked,
			area:      area,
		})
	}

	if len(metrics) == 0 {
		return handles
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
		candidates := listWindowCandidates(uint32(pid), true)
		if len(candidates) == 0 {
			candidates = listWindowCandidates(uint32(pid), false)
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
