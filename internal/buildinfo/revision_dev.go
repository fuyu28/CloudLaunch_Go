//go:build !production

// Package buildinfo provides the development-build fallback for generated revision metadata.
package buildinfo

func generatedRevision() string {
	return unknownRevision
}
