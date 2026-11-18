package kanban

import (
	"strings"
	"testing"
)

func TestNewIDIsAValidUUIDVersion4(t *testing.T) {
	id := NewID()

	if !IsValidID(id) {
		t.Fatalf("NewID() = %q, which IsValidID rejects", id)
	}
	if id[14] != '4' {
		t.Errorf("NewID() = %q, want version nibble 4 at index 14", id)
	}
	if !strings.ContainsRune("89ab", rune(id[19])) {
		t.Errorf("NewID() = %q, want variant nibble 8, 9, a or b at index 19", id)
	}
}

func TestNewIDDoesNotRepeat(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		id := NewID()
		if seen[id] {
			t.Fatalf("NewID() returned %q twice", id)
		}
		seen[id] = true
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"a lowercase canonical UUID", "0b7e3c1a-5d2f-4a8b-9c3d-1e2f3a4b5c6d", true},
		{"an empty string", "", false},
		{"a word", "board-one", false},
		{"too short", "0b7e3c1a-5d2f-4a8b-9c3d-1e2f3a4b5c6", false},
		{"too long", "0b7e3c1a-5d2f-4a8b-9c3d-1e2f3a4b5c6d0", false},
		{"dashes in the wrong places", "0b7e3c1a5-d2f-4a8b-9c3d-1e2f3a4b5c6d", false},
		{"non-hex characters", "0b7e3c1a-5d2f-4a8b-9c3d-1e2f3a4b5c6g", false},
		{"uppercase hex", "0B7E3C1A-5D2F-4A8B-9C3D-1E2F3A4B5C6D", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidID(tt.id); got != tt.want {
				t.Errorf("IsValidID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestChooseID(t *testing.T) {
	const wanted = "0b7e3c1a-5d2f-4a8b-9c3d-1e2f3a4b5c6d"
	unused := func(string) bool { return false }

	t.Run("keeps a valid unused ID", func(t *testing.T) {
		if got := ChooseID(wanted, unused); got != wanted {
			t.Errorf("ChooseID = %q, want %q", got, wanted)
		}
	})
	t.Run("generates one when the requested ID is already used", func(t *testing.T) {
		got := ChooseID(wanted, func(id string) bool { return id == wanted })
		if got == wanted || !IsValidID(got) {
			t.Errorf("ChooseID = %q, want a new valid ID", got)
		}
	})
	t.Run("generates one when the requested ID is not a UUID", func(t *testing.T) {
		got := ChooseID("nope", unused)
		if got == "nope" || !IsValidID(got) {
			t.Errorf("ChooseID = %q, want a new valid ID", got)
		}
	})
	t.Run("generates one when nothing was requested", func(t *testing.T) {
		if got := ChooseID("", unused); !IsValidID(got) {
			t.Errorf("ChooseID = %q, want a valid ID", got)
		}
	})
}
