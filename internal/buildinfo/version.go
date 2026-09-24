// Package buildinfo exposes metadata recorded in the compiled application.
package buildinfo

import "runtime/debug"

const unknownRevision = "不明"

// Revision returns the commit that was checked out when the application was built.
func Revision() string {
	if revision := generatedRevision(); revision != unknownRevision {
		return revision
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return unknownRevision
	}

	return revisionFromSettings(info.Settings)
}

func revisionFromSettings(settings []debug.BuildSetting) string {
	for _, setting := range settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			return setting.Value
		}
	}

	return unknownRevision
}
