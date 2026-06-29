package sync

import "testing"

func TestMatchSelectPattern(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		// Prefix matching (no wildcards)
		{"prefix exact dir", "videos", "videos", true},
		{"prefix under dir", "videos", "videos/clip.mp4", true},
		{"prefix nested", "videos/2024", "videos/2024/sub/clip.mp4", true},
		{"prefix no partial segment", "videos/2024", "videos/2024-old/clip.mp4", false},
		{"prefix mismatch", "videos", "audio/song.mp3", false},
		{"prefix exact file", "docs/readme.txt", "docs/readme.txt", true},
		{"prefix leading slash normalized", "/videos", "videos/a.mp4", true},

		// Single star: one segment, does not cross /
		{"star direct child", "videos/*", "videos/a.mp4", true},
		{"star not deep", "videos/*", "videos/sub/a.mp4", false},
		{"star ext top level", "*.mp4", "a.mp4", true},
		{"star ext not nested", "*.mp4", "videos/a.mp4", false},

		// Double star: any number of segments
		{"globstar everything", "videos/**", "videos/a/b/c.mp4", true},
		{"globstar zero segments", "videos/**", "videos", false},
		{"globstar ext any depth", "**/*.mp4", "videos/2024/a.mp4", true},
		{"globstar ext top", "**/*.mp4", "a.mp4", true},
		{"globstar ext mismatch", "**/*.mp4", "videos/a.mov", false},
		{"globstar lone", "**", "anything/at/all.bin", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := matchSelectPattern(c.pattern, c.path); got != c.want {
				t.Errorf("matchSelectPattern(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
			}
		})
	}
}
