package install

import (
	"testing"
)

func TestMetadataSkillEntries_FullPathKeyWithGroup(t *testing.T) {
	store := NewMetadataStore()
	store.Set("team/_demo", &MetadataEntry{
		Source:  "file:///tmp/demo",
		Tracked: true,
		Group:   "team",
	})

	dtos := MetadataSkillEntries(store)
	if len(dtos) != 1 {
		t.Fatalf("expected 1 dto, got %d", len(dtos))
	}

	got := dtos[0]
	if got.Name != "_demo" {
		t.Fatalf("Name = %q, want %q", got.Name, "_demo")
	}
	if got.Group != "team" {
		t.Fatalf("Group = %q, want %q", got.Group, "team")
	}
	if got.FullName() != "team/_demo" {
		t.Fatalf("FullName() = %q, want %q", got.FullName(), "team/_demo")
	}
}

func TestMetadataSkillEntries_LegacyBasenameKeyWithGroup(t *testing.T) {
	store := NewMetadataStore()
	store.Set("_demo", &MetadataEntry{
		Source:  "file:///tmp/demo",
		Tracked: true,
		Group:   "team",
	})

	dtos := MetadataSkillEntries(store)
	if len(dtos) != 1 {
		t.Fatalf("expected 1 dto, got %d", len(dtos))
	}

	got := dtos[0]
	if got.Name != "_demo" {
		t.Fatalf("Name = %q, want %q", got.Name, "_demo")
	}
	if got.Group != "team" {
		t.Fatalf("Group = %q, want %q", got.Group, "team")
	}
	if got.FullName() != "team/_demo" {
		t.Fatalf("FullName() = %q, want %q", got.FullName(), "team/_demo")
	}
}
