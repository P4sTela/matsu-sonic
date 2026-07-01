package converter

import (
	"testing"
)

func TestStemOf(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/videos/intro.mp4", "intro"},
		{"intro.mp4", "intro"},
		{"/videos/intro", "intro"},
		{"file.tar.gz", "file.tar"},
	}
	for _, tt := range tests {
		got := stemOf(tt.path)
		if got != tt.want {
			t.Errorf("stemOf(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDirOf(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/videos/intro.mp4", "/videos"},
		{"intro.mp4", "."},
	}
	for _, tt := range tests {
		got := dirOf(tt.path)
		if got != tt.want {
			t.Errorf("dirOf(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestExpandCommand(t *testing.T) {
	input := "/sync/videos/intro.mp4"
	output := "/sync/converted/hap/videos/intro.mov"
	outputDir := "converted/hap"

	tokens := expandCommand("ffmpeg -y -i {{input}} -c:v hap {{output}}", input, output, outputDir)
	if len(tokens) != 7 {
		t.Fatalf("expected 7 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[3] != input {
		t.Errorf("token[3] = %q, want %q", tokens[3], input)
	}
	if tokens[6] != output {
		t.Errorf("token[6] = %q, want %q", tokens[6], output)
	}
}

func TestExpandCommandWithQuotes(t *testing.T) {
	input := "/sync/videos/intro.mp4"
	output := "/sync/converted/hap/videos/intro.mov"

	template := `ffmpeg -y -i "{{input}}" -metadata title="My Video" {{output}}`
	tokens := expandCommand(template, input, output, "")

	if len(tokens) < 4 {
		t.Fatalf("expected at least 4 tokens, got %d: %v", len(tokens), tokens)
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		wantLen  int
		wantLast string
	}{
		{"echo hello world", 3, "world"},
		{`echo "hello world"`, 2, "hello world"},
		{`echo 'hello world'`, 2, "hello world"},
		{`echo hello\ world`, 2, "hello world"},
	}
	for _, tt := range tests {
		tokens := tokenize(tt.input)
		if len(tokens) != tt.wantLen {
			t.Errorf("tokenize(%q) = %d tokens, want %d: %v", tt.input, len(tokens), tt.wantLen, tokens)
			continue
		}
		if tokens[len(tokens)-1] != tt.wantLast {
			t.Errorf("tokenize(%q) last = %q, want %q", tt.input, tokens[len(tokens)-1], tt.wantLast)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"*.mp4", "intro.mp4", true},
		{"*.mp4", "intro.mov", false},
		{"*.mp4", "intro.mp4.txt", false},
	}
	for _, tt := range tests {
		got := matchPattern(tt.pattern, tt.name)
		if got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}
