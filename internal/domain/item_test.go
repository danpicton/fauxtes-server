package domain

import (
	"testing"
	"time"
)

func TestValidateItemHashUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		uuid    string
		wantErr bool
	}{
		{name: "valid_uuid", uuid: "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d", wantErr: false},
		{name: "empty_uuid", uuid: "", wantErr: true},
		{name: "too_short", uuid: "abc-123", wantErr: true},
		{name: "invalid_chars", uuid: "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz", wantErr: true},
		{name: "no_hyphens", uuid: "a4b8e3c21f6d4a3b9c7e8d2f5a6b3c4d", wantErr: false}, // UUID without hyphens is still valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := ItemHash{UUID: tt.uuid}
			err := h.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ItemHash{UUID: %q}.Validate() error = %v, wantErr %v", tt.uuid, err, tt.wantErr)
			}
		})
	}
}

func TestItemToHTTPRepresentation(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	nowMicro := now.UnixMicro()

	item := Item{
		UUID:                "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d",
		UserUUID:            "b5c9f4d3-2e7a-4b8c-ad6f-9e3a7b4c5d6e",
		Content:             strPtr("encrypted-content"),
		ContentType:         strPtr("Note"),
		EncItemKey:          strPtr("enc-key-data"),
		AuthHash:            nil,
		ItemsKeyID:          strPtr("items-key-123"),
		DuplicateOf:         nil,
		Deleted:             false,
		UpdatedWithSession:  strPtr("session-uuid"),
		KeySystemIdentifier: nil,
		SharedVaultUUID:     nil,
		CreatedAt:           now,
		UpdatedAt:           now,
		CreatedAtTimestamp:  nowMicro,
		UpdatedAtTimestamp:  nowMicro,
	}

	rep := item.ToHTTPRepresentation()

	if rep.UUID != item.UUID {
		t.Errorf("UUID = %q, want %q", rep.UUID, item.UUID)
	}
	if rep.ContentType != "Note" {
		t.Errorf("ContentType = %q, want %q", rep.ContentType, "Note")
	}
	if rep.Deleted != false {
		t.Errorf("Deleted = %v, want false", rep.Deleted)
	}
	if rep.CreatedAtTimestamp != nowMicro {
		t.Errorf("CreatedAtTimestamp = %d, want %d", rep.CreatedAtTimestamp, nowMicro)
	}
	if rep.UpdatedAtTimestamp != nowMicro {
		t.Errorf("UpdatedAtTimestamp = %d, want %d", rep.UpdatedAtTimestamp, nowMicro)
	}
	if deref(rep.Content) != "encrypted-content" {
		t.Errorf("Content = %v, want %q", rep.Content, "encrypted-content")
	}
	if deref(rep.EncItemKey) != "enc-key-data" {
		t.Errorf("EncItemKey = %v, want %q", rep.EncItemKey, "enc-key-data")
	}
	if rep.AuthHash != nil {
		t.Errorf("AuthHash = %v, want nil", rep.AuthHash)
	}
}

func TestItemHashContentSize(t *testing.T) {
	t.Parallel()

	h := ItemHash{
		UUID:        "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d",
		Content:     strPtr("some encrypted content here"),
		ContentType: strPtr("Note"),
	}

	size := h.ContentSize()
	if size <= 0 {
		t.Errorf("ContentSize() = %d, want > 0", size)
	}

	// A hash with no content should have a smaller size
	empty := ItemHash{UUID: "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d"}
	emptySize := empty.ContentSize()
	if emptySize >= size {
		t.Errorf("empty ContentSize() = %d, should be < %d", emptySize, size)
	}
}

// Test helpers

func strPtr(s string) *string {
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
