package kanban

import "testing"

func TestMemoryStore(t *testing.T) {
	testStore(t, func(*testing.T) Store { return NewMemory() })
}
