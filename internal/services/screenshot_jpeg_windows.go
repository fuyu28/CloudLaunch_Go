//go:build windows

package services

import (
	"image"
	"image/jpeg"
	"os"
)

func (service *ScreenshotService) convertFileToJpeg(sourcePath string, destPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			service.logger.Warn("スクリーンショットの保存に失敗", "error", closeErr)
		}
	}()

	quality := normalizeJpegQuality(service.jpegQuality)
	return jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
}

func normalizeJpegQuality(value int) int {
	if value < 1 || value > 100 {
		return 85
	}
	return value
}
