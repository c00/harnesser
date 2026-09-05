package redact

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/lastpersonlabs/goredact"
)

func RedactBytes(ctx context.Context, data []byte) ([]byte, error) {
	engine, err := goredact.New(goredact.Config{
		Profile: goredact.ProfileBalanced,
		CustomRules: []goredact.CustomRule{
			{
				ID:            "firecrawl-api-key",
				Triggers:      []string{"fc-"},
				MaxLookbehind: 1,
				MaxLookahead:  33,
				Confidence:    goredact.ConfidenceHigh,
				Validate:      firecrawlValidation,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create redactor: %w", err)
	}

	writer := bytes.NewBuffer([]byte{})
	reader := bytes.NewReader(data)

	stats, err := engine.Redact(ctx, writer, reader)
	if err != nil {
		return nil, fmt.Errorf("cannot redact stream: %w", err)
	}

	slog.Info("Redacting buffer", "stats", stats)

	return writer.Bytes(), nil
}

func firecrawlValidation(src []byte, start, end int) (int, int, bool) {
	// 1. Prefix Boundary Check:
	// Ensure we aren't inside another identifier (e.g., "myfc-123...")
	if start > 0 {
		if isIdent(src[start-1]) {
			return 0, 0, false
		}
	}

	// 2. Length Check:
	// Firecrawl keys have exactly 32 trailing hex characters.
	const keyBodyLen = 32
	if len(src)-end < keyBodyLen {
		return 0, 0, false
	}

	// 3. Hexadecimal Validation:
	for i := 0; i < keyBodyLen; i++ {
		if !isHex(src[end+i]) {
			return 0, 0, false
		}
	}

	newEnd := end + keyBodyLen

	// 4. Suffix Boundary Check:
	// Ensure the key doesn't continue with more identifier characters.
	if len(src) > newEnd {
		if isIdent(src[newEnd]) {
			return 0, 0, false
		}
	}

	return start, newEnd, true
}

// Helper: check if a byte is a common identifier character [a-zA-Z0-9_-]
func isIdent(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

// Helper: check if a byte is hexadecimal [0-9a-fA-F]
func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
