package converter

import (
	"strings"
)

// expandCommand replaces template variables in a command template string
// and returns the result split into tokens for safe execution.
//
// Supported variables:
//
//	{{input}}      — absolute input file path
//	{{output}}     — absolute output file path
//	{{stem}}       — input filename without extension
//	{{dir}}        — input file's parent directory
//	{{output_dir}} — converter's OutputDir (sync-root relative)
func expandCommand(template, input, output, outputDir string) []string {
	replacer := strings.NewReplacer(
		"{{input}}", input,
		"{{output}}", output,
		"{{stem}}", stemOf(input),
		"{{dir}}", dirOf(input),
		"{{output_dir}}", outputDir,
	)
	s := replacer.Replace(template)
	return tokenize(s)
}

// stemOf returns the filename without its last extension.
func stemOf(path string) string {
	s := path
	// Take the last segment
	if idx := strings.LastIndexByte(s, '/'); idx >= 0 {
		s = s[idx+1:]
	}
	// Remove extension
	if idx := strings.LastIndexByte(s, '.'); idx >= 0 {
		return s[:idx]
	}
	return s
}

// dirOf returns the absolute path of the directory containing the input file.
func dirOf(path string) string {
	if idx := strings.LastIndexByte(path, '/'); idx >= 0 {
		return path[:idx]
	}
	return "."
}

// tokenize splits a command string into tokens respecting shell-like quoting.
// This is a simple implementation: it splits on whitespace and handles
// double-quoted and single-quoted segments.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, r := range s {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == ' ' || r == '\t':
			if inSingle || inDouble {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}
