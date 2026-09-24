// Package buildinfo provides the production-build fallback for revision metadata.
//go:build production

package buildinfo

func generatedRevision() string {
	return unknownRevision
}
