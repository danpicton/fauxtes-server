package domain

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestEncodeSyncToken(t *testing.T) {
	t.Parallel()

	t.Run("produces_base64_v2_timestamp", func(t *testing.T) {
		t.Parallel()
		got := EncodeSyncToken(1234567890)
		decoded, err := base64.StdEncoding.DecodeString(got)
		if err != nil {
			t.Fatalf("EncodeSyncToken returned invalid base64: %v", err)
		}
		want := "2:1234567890"
		if string(decoded) != want {
			t.Errorf("decoded sync token = %q, want %q", string(decoded), want)
		}
	})
}

func TestDecodeSyncToken(t *testing.T) {
	t.Parallel()

	encode := func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}

	tests := []struct {
		name    string
		token   string
		want    int64
		wantErr bool
	}{
		{
			name:  "v1_string_date",
			token: encode("1:2023-01-15T10:30:00.000Z"),
			want:  1673778600000000, // 2023-01-15T10:30:00Z in microseconds
		},
		{
			name:  "v2_numeric_timestamp",
			token: encode("2:1673777400000000"),
			want:  1673777400000000,
		},
		{
			name:  "empty_returns_zero",
			token: "",
			want:  0,
		},
		{
			name:    "invalid_base64",
			token:   "!!!not-base64!!!",
			wantErr: true,
		},
		{
			name:    "malformed_no_colon",
			token:   encode("justgarbage"),
			wantErr: true,
		},
		{
			name:    "malformed_invalid_version",
			token:   encode("99:12345"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := DecodeSyncToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeSyncToken(%q) error = %v, wantErr %v", tt.token, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DecodeSyncToken(%q) = %d, want %d", tt.token, got, tt.want)
			}
		})
	}
}

func TestSyncTokenRoundtrip(t *testing.T) {
	t.Parallel()

	timestamps := []int64{0, 1, 1673777400000000, 9999999999999999}
	for _, ts := range timestamps {
		t.Run(fmt.Sprintf("timestamp_%d", ts), func(t *testing.T) {
			t.Parallel()
			encoded := EncodeSyncToken(ts)
			decoded, err := DecodeSyncToken(encoded)
			if err != nil {
				t.Fatalf("roundtrip failed: encode(%d) = %q, decode error: %v", ts, encoded, err)
			}
			if decoded != ts {
				t.Errorf("roundtrip: encode(%d) -> decode = %d", ts, decoded)
			}
		})
	}
}
