package services

import "errors"

// ErrNoNewScreenshot はSnipping Toolで新規画像が取得されなかったことを示す。
var ErrNoNewScreenshot = errors.New("no new screenshot")
