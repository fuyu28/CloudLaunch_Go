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
	hwnd, err := findBestWindowForPID(uint32(pid))
	if err != nil {
		return false, err
	}
	helperPath, err := wgcHelperPath()
	if err != nil || helperPath == "" {
		return false, nil
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
			return false, fmt.Errorf("WGC capture failed: %w: %s", err, message)
		}
		return false, fmt.Errorf("WGC capture failed: %w", err)
	}
	return true, nil
}

func wgcHelperPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "wgc_screenshot.exe"), nil
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
