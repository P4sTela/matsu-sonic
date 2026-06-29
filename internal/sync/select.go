package sync

import (
	"path"
	"path/filepath"
	"strings"
)

// IsSelectedBy reports whether relPath is included by the given select patterns.
// Empty (or all-blank) patterns select everything, mirroring sync behaviour.
func IsSelectedBy(patterns []string, relPath string) bool {
	hasPattern := false
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		hasPattern = true
		if matchSelectPattern(pattern, relPath) {
			return true
		}
	}
	return !hasPattern
}

// matchSelectPattern reports whether relPath (a forward-slash relative path
// from the sync root) is selected by pattern.
//
// Matching rules:
//   - A pattern without wildcard characters is treated as a path prefix:
//     "videos/2024" matches "videos/2024" itself and anything under it
//     ("videos/2024/clip.mp4"), but not "videos/2024-old".
//   - A pattern with wildcards is matched segment by segment:
//     "*" matches a single path segment (does not cross "/"),
//     "**" matches zero or more segments.
//     e.g. "videos/*"   -> direct children of videos
//          "videos/**"  -> everything under videos
//          "**/*.mp4"   -> any .mp4 at any depth
func matchSelectPattern(pattern, relPath string) bool {
	pattern = strings.Trim(filepath.ToSlash(pattern), "/")
	relPath = strings.Trim(filepath.ToSlash(relPath), "/")

	if pattern == "" {
		return true
	}

	// No wildcards -> prefix match.
	if !strings.ContainsAny(pattern, "*?[") {
		return relPath == pattern || strings.HasPrefix(relPath, pattern+"/")
	}

	return matchSegments(strings.Split(pattern, "/"), strings.Split(relPath, "/"))
}

// matchSegments matches pattern segments against path segments, where "**"
// matches any number of segments (including zero).
func matchSegments(pat, name []string) bool {
	if len(pat) == 0 {
		return len(name) == 0
	}

	if pat[0] == "**" {
		// Trailing "**" means "everything under here": requires at least one
		// remaining segment, so "videos/**" matches "videos/a" but not "videos".
		if len(pat) == 1 {
			return len(name) > 0
		}
		for i := 0; i <= len(name); i++ {
			if matchSegments(pat[1:], name[i:]) {
				return true
			}
		}
		return false
	}

	if len(name) == 0 {
		return false
	}
	if ok, _ := path.Match(pat[0], name[0]); !ok {
		return false
	}
	return matchSegments(pat[1:], name[1:])
}
