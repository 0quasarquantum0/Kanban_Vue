package kanban

import (
	"crypto/rand"
	"fmt"
)

// NewID returns a random version 4 UUID in lowercase canonical form, like crypto.randomUUID()
// in the browser. The standard library has no UUID type, and 16 random bytes are all it takes.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing means the operating system has no randomness; nothing sensible
		// can continue from here.
		panic(fmt.Sprintf("kanban: reading random bytes: %v", err))
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // variant 10xx
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// IsValidID reports whether id is a UUID in lowercase canonical form (8-4-4-4-12 hex digits).
func IsValidID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i := range len(id) {
		c := id[i]
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
				return false
			}
		}
	}
	return true
}

// ChooseID returns requested if it is a valid UUID that taken says is unused, and a freshly
// generated ID otherwise. This is how the server accepts IDs made by the client while keeping
// them unique.
func ChooseID(requested string, taken func(id string) bool) string {
	if IsValidID(requested) && !taken(requested) {
		return requested
	}
	for {
		if id := NewID(); !taken(id) {
			return id
		}
	}
}
