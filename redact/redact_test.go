package redact

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactBytes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		secrets   []string
		preserved []string
	}{
		{
			name:  "plain text",
			input: "nothing sensitive here",
		},
		{
			name:      "Firecrawl API key",
			input:     "before FIRECRAWL_API_KEY=fc-6e6f74617265616c7468696e67776f77 after",
			secrets:   []string{"fc-6e6f74617265616c7468696e67776f77"},
			preserved: []string{"before", "after"},
		},
		{
			name:      "OpenAI API key",
			input:     "before OPENAI_API_KEY=sk-aB3dE5fG7hJ9kL2mN4pQ6rS8tV1wX3yZ5cD7eF9gH2jK4mN6 after",
			secrets:   []string{"sk-aB3dE5fG7hJ9kL2mN4pQ6rS8tV1wX3yZ5cD7eF9gH2jK4mN6"},
			preserved: []string{"before", "after"},
		},
		{
			name:      "GitHub personal access token",
			input:     "before GITHUB_TOKEN=ghp_aB3dE5fG7hJ9kL2mN4pQ6rS8tV1wX3yZ5cD7 after",
			secrets:   []string{"ghp_aB3dE5fG7hJ9kL2mN4pQ6rS8tV1wX3yZ5cD7e"},
			preserved: []string{"before", "after"},
		},
		{
			name:      "AWS access key",
			input:     "before AWS_ACCESS_KEY_ID=AKIAQ7N5Q2W6R3T4Y5UP after",
			secrets:   []string{"AKIAQ7N5Q2W6R3T4Y5UP"},
			preserved: []string{"before", "after"},
		},
		{
			name:  "multiple API keys",
			input: "firecrawl=fc-6e6f74617265616c7468696e67776f77 github=ghp_aB3dE5fG7hJ9kL2mN4pQ6rS8tV1wX3yZ5cD7",
			secrets: []string{
				"fc-6e6f74617265616c7468696e67776f77",
				"ghp_aB3dE5fG7hJ9kL2mN4pQ6rS8tV1wX3yZ5cD7e",
			},
			preserved: []string{"firecrawl=", "github="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RedactBytes(t.Context(), []byte(tt.input))
			require.NoError(t, err)

			for _, secret := range tt.secrets {
				assert.NotContains(t, string(got), secret)
			}
			for _, preserved := range tt.preserved {
				assert.Contains(t, string(got), preserved)
			}
			if len(tt.secrets) == 0 {
				assert.Equal(t, tt.input, string(got))
			}
		})
	}
}
