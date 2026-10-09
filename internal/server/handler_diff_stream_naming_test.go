package server

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/config"
	ssync "skillshare/internal/sync"
)

func TestComputeTargetDiff_NamingChangeReportsRename(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	for _, dir := range []string{filepath.Join(source, "_emil-design", "skills", "prototype"), filepath.Join(target, "prototype")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: prototype\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := &ssync.Manifest{Managed: map[string]string{"prototype": "x"}, Naming: map[string]string{"prototype": "standard"}}
	if err := ssync.WriteManifest(target, manifest); err != nil {
		t.Fatal(err)
	}
	discovered, err := ssync.DiscoverSourceSkills(source)
	if err != nil {
		t.Fatal(err)
	}

	dt := (&Server{}).computeTargetDiff("claude", config.TargetConfig{Skills: &config.ResourceTargetConfig{Path: target, Mode: "copy", TargetNaming: "prefixed"}}, discovered, "merge", source, nil)
	if len(dt.Items) != 1 || dt.Items[0].Skill != "emil-design-prototype" || dt.Items[0].Action != "update" || dt.Items[0].Reason != "renamed from prototype (target naming changed)" {
		t.Fatalf("items = %+v, want one update renaming prototype", dt.Items)
	}
}
