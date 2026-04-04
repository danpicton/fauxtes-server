package domain

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// EncodeSyncToken creates a V2 sync token from a microsecond timestamp.
// Format: base64("2:<microseconds>")
func EncodeSyncToken(timestampMicro int64) string {
	raw := fmt.Sprintf("2:%d", timestampMicro)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeSyncToken decodes a sync token and returns the microsecond timestamp.
// Returns 0 for empty tokens (first sync).
// Supports V1 (string date) and V2 (numeric microseconds) formats.
func DecodeSyncToken(token string) (int64, error) {
	if token == "" {
		return 0, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("invalid sync token encoding: %w", err)
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("malformed sync token: expected version:value, got %q", string(decoded))
	}

	version := parts[0]
	value := parts[1]

	switch version {
	case "1":
		t, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			// Try alternate format without timezone
			t, err = time.Parse("2006-01-02T15:04:05.000Z", value)
			if err != nil {
				return 0, fmt.Errorf("invalid V1 sync token date: %w", err)
			}
		}
		return t.UnixMicro(), nil
	case "2":
		ts, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid V2 sync token timestamp: %w", err)
		}
		return ts, nil
	default:
		return 0, fmt.Errorf("unsupported sync token version: %s", version)
	}
}
