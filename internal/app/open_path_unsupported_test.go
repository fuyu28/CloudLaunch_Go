//go:build !windows && !darwin && !linux

package app

import (
	"errors"
	"testing"
)

func TestOpenPathUnsupported(t *testing.T) {
	t.Parallel()

	err := openPath("/tmp/any")
	if !errors.Is(err, errOpenPathUnsupported) {
		t.Fatalf("openPath() error = %v, want errOpenPathUnsupported", err)
	}
}
