package replacetextcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceText(t *testing.T) {
	tests := []struct {
		name            string
		contents        string
		search          string
		replacement     string
		expectedMatches int
		wantContents    string
		wantReplaced    int
		wantError       string
	}{
		{name: "replaces one match", contents: "hello world\n", search: "world", replacement: "there", expectedMatches: 1, wantContents: "hello there\n", wantReplaced: 1},
		{name: "replaces every match", contents: "one two one\n", search: "one", replacement: "three", expectedMatches: 2, wantContents: "three two three\n", wantReplaced: 2},
		{name: "supports multiline text", contents: "before\nold line\nafter\n", search: "old line\nafter", replacement: "new lines", expectedMatches: 1, wantContents: "before\nnew lines\n", wantReplaced: 1},
		{name: "allows an expected count of zero", contents: "unchanged\n", search: "missing", replacement: "new", expectedMatches: 0, wantContents: "unchanged\n", wantReplaced: 0},
		{name: "does not edit when count is too low", contents: "one two\n", search: "one", replacement: "changed", expectedMatches: 2, wantContents: "one two\n", wantError: "expected 2 matches, found 1"},
		{name: "does not edit when count is too high", contents: "one one\n", search: "one", replacement: "changed", expectedMatches: 1, wantContents: "one one\n", wantError: "expected 1 matches, found 2"},
		{name: "rejects empty search text", contents: "unchanged\n", search: "", replacement: "new", expectedMatches: 1, wantContents: "unchanged\n", wantError: "search text cannot be empty"},
		{name: "rejects negative expected matches", contents: "unchanged\n", search: "unchanged", replacement: "new", expectedMatches: -1, wantContents: "unchanged\n", wantError: "expected matches cannot be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "input.txt")
			require.NoError(t, os.WriteFile(filename, []byte(tt.contents), 0o600))

			replaced, err := replaceText(filename, tt.search, tt.replacement, tt.expectedMatches)

			if tt.wantError != "" {
				assert.EqualError(t, err, tt.wantError)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantReplaced, replaced)

			contents, readErr := os.ReadFile(filename)
			require.NoError(t, readErr)
			assert.Equal(t, tt.wantContents, string(contents))
		})
	}
}

func TestReplaceTextMissingFile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "missing.txt")

	replaced, err := replaceText(filename, "old", "new", 1)

	assert.Error(t, err)
	assert.Zero(t, replaced)
}

func TestExpectedMatchesDefault(t *testing.T) {
	flag := Cmd.Flags().Lookup("expected-matches")

	require.NotNil(t, flag)
	assert.Equal(t, "1", flag.DefValue)
}
