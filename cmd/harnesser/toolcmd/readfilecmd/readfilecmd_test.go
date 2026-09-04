package readfilecmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFile(t *testing.T) {
	tests := []struct {
		name          string
		contents      string
		startLine     int
		maxLines      int
		maxBytes      int
		includeLineNr bool
		want          string
	}{
		{name: "entire file", contents: "first\nsecond\nthird\n", startLine: 1, maxLines: 100, maxBytes: 1024, want: "first\nsecond\nthird\n"},
		{name: "starts at requested line", contents: "first\nsecond\nthird", startLine: 2, maxLines: 100, maxBytes: 1024, want: "second\nthird"},
		{name: "limited by lines", contents: "first\nsecond\nthird\n", startLine: 1, maxLines: 2, maxBytes: 1024, want: "first\nsecond\n"},
		{name: "limited by bytes", contents: "first\nsecond\n", startLine: 1, maxLines: 100, maxBytes: 8, want: "first\nse\n[output truncated: maximum byte count reached]\n"},
		{name: "aligned line numbers", contents: "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\nten\n", startLine: 9, maxLines: 2, maxBytes: 1024, includeLineNr: true, want: " 9 nine\n10 ten\n"},
		{name: "zero lines", contents: "first\nsecond\n", startLine: 1, maxLines: 0, maxBytes: 1024, want: ""},
		{name: "zero bytes", contents: "first\n", startLine: 1, maxLines: 100, maxBytes: 0, want: "[output truncated: maximum byte count reached]\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "input.txt")
			require.NoError(t, os.WriteFile(filename, []byte(tt.contents), 0o600))

			got, err := readFile(filename, tt.startLine, tt.maxLines, tt.maxBytes, tt.includeLineNr)

			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestReadFileErrors(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "input.txt")
	emptyFile := filepath.Join(tempDir, "empty.txt")
	require.NoError(t, os.WriteFile(existingFile, []byte("one\ntwo\n"), 0o600))
	require.NoError(t, os.WriteFile(emptyFile, nil, 0o600))

	tests := []struct {
		name      string
		filename  string
		startLine int
		maxLines  int
		maxBytes  int
	}{
		{name: "missing file", filename: filepath.Join(tempDir, "missing.txt"), startLine: 1, maxLines: 1, maxBytes: 1},
		{name: "empty file", filename: emptyFile, startLine: 1, maxLines: 1, maxBytes: 1},
		{name: "start beyond end", filename: existingFile, startLine: 3, maxLines: 1, maxBytes: 1},
		{name: "zero start", filename: existingFile, startLine: 0, maxLines: 1, maxBytes: 1},
		{name: "negative lines", filename: existingFile, startLine: 1, maxLines: -1, maxBytes: 1},
		{name: "negative bytes", filename: existingFile, startLine: 1, maxLines: 1, maxBytes: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readFile(tt.filename, tt.startLine, tt.maxLines, tt.maxBytes, false)
			assert.Error(t, err)
		})
	}
}
