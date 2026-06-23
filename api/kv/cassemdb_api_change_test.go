package kv

import (
	"testing"

	"github.com/yeqown/cassem/api/concept"
)

func TestChangeMessagesImplementConceptChange(t *testing.T) {
	var _ concept.Change = (*Change)(nil)
	var _ concept.Change = (*ParentDirectoryChange)(nil)

	change := &Change{Key: "app/default/key"}
	if change.Topic() != "app/default/key" {
		t.Fatalf("Change.Topic() = %q, want %q", change.Topic(), "app/default/key")
	}
	if change.Type() != concept.ChangeType_KV {
		t.Fatalf("Change.Type() = %v, want %v", change.Type(), concept.ChangeType_KV)
	}

	dirChange := &ParentDirectoryChange{SpecificTopic: "app/default"}
	if dirChange.Topic() != "app/default" {
		t.Fatalf("ParentDirectoryChange.Topic() = %q, want %q", dirChange.Topic(), "app/default")
	}
	if dirChange.Type() != concept.ChangeType_DIR {
		t.Fatalf("ParentDirectoryChange.Type() = %v, want %v", dirChange.Type(), concept.ChangeType_DIR)
	}
}
