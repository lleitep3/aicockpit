package bot

import (
	"testing"
)

func TestNewIDGeneratesUniqueIDs(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		id, err := NewID()
		if err != nil {
			t.Fatalf("NewID() error = %v", err)
		}
		if len(id) == 0 {
			t.Fatalf("NewID() returned empty id")
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate bot id generated: %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestNewIDReturnsStablePrefix(t *testing.T) {
	id, err := NewID()
	if err != nil {
		t.Fatalf("NewID() error = %v", err)
	}
	if len(id) < len("bot_") {
		t.Fatalf("id %q shorter than expected prefix", id)
	}
	if id[:4] != "bot_" {
		t.Fatalf("id %q does not start with bot_", id)
	}
}
