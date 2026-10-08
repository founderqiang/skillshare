//go:build !online

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/install"
	"skillshare/internal/testutil"
)

func TestSyncProject_MultipleTargets(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	projectRoot := sb.SetupProjectDir("claude", "cursor")
	sb.CreateProjectSkill(projectRoot, "shared", map[string]string{
		"SKILL.md": "# Shared",
	})

	result := sb.RunCLIInDir(projectRoot, "sync", "-p")
	result.AssertSuccess(t)

	if !sb.IsSymlink(filepath.Join(projectRoot, ".claude", "skills", "shared")) {
		t.Error("symlink in claude target missing")
	}
	if !sb.IsSymlink(filepath.Join(projectRoot, ".agents", "skills", "shared")) {
		t.Error("symlink in cursor target missing")
	}
}

func TestSyncProject_PrunesOrphanLinks(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	projectRoot := sb.SetupProjectDir("claude")
	sb.CreateProjectSkill(projectRoot, "skill-a", map[string]string{"SKILL.md": "# A"})
	sb.CreateProjectSkill(projectRoot, "skill-b", map[string]string{"SKILL.md": "# B"})

	// First sync
	sb.RunCLIInDir(projectRoot, "sync", "-p").AssertSuccess(t)

	// Remove skill-b from source
	os.RemoveAll(filepath.Join(projectRoot, ".skillshare", "skills", "skill-b"))

	// Second sync prunes
	result := sb.RunCLIInDir(projectRoot, "sync", "-p")
	result.AssertSuccess(t)

	if sb.FileExists(filepath.Join(projectRoot, ".claude", "skills", "skill-b")) {
		t.Error("skill-b should be pruned")
	}
	if !sb.IsSymlink(filepath.Join(projectRoot, ".claude", "skills", "skill-a")) {
		t.Error("skill-a should remain")
	}
}

func TestSyncProject_DryRun_NoChanges(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	projectRoot := sb.SetupProjectDir("claude")
	sb.CreateProjectSkill(projectRoot, "test", map[string]string{"SKILL.md": "# Test"})

	result := sb.RunCLIInDir(projectRoot, "sync", "-p", "--dry-run")
	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "Dry run")

	if sb.IsSymlink(filepath.Join(projectRoot, ".claude", "skills", "test")) {
		t.Error("dry-run should not create symlinks")
	}
}

func TestSyncProject_PreservesRegistryEntries(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	projectRoot := sb.SetupProjectDir("claude")

	// Create a skill on disk that sync will discover
	sb.CreateProjectSkill(projectRoot, "local-skill", map[string]string{
		"SKILL.md": "# Local Skill",
	})

	// Write metadata with a remote-installed skill that has NO files on disk.
	// Sync must NOT prune this entry — the metadata is the source of truth for installations.
	skillsDir := filepath.Join(projectRoot, ".skillshare", "skills")
	store := install.NewMetadataStore()
	store.Set("remote-tool", &install.MetadataEntry{Source: "github.com/someone/remote-tool"})
	store.Save(skillsDir)

	result := sb.RunCLIInDir(projectRoot, "sync", "-p")
	result.AssertSuccess(t)

	// Verify metadata still contains the remote-tool entry
	store2, err := install.LoadMetadata(skillsDir)
	if err != nil {
		t.Fatalf("failed to load metadata: %v", err)
	}
	if !store2.Has("remote-tool") {
		t.Errorf("sync should preserve metadata entry for installed skill without local files")
	}
	entry := store2.Get("remote-tool")
	if entry == nil || entry.Source != "github.com/someone/remote-tool" {
		t.Errorf("sync should preserve source in metadata entry")
	}
}

func TestSyncProject_AutoDetectsMode(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	projectRoot := sb.SetupProjectDir("claude")
	sb.CreateProjectSkill(projectRoot, "auto", map[string]string{"SKILL.md": "# Auto"})

	// No -p flag; auto-detects from .skillshare/config.yaml
	result := sb.RunCLIInDir(projectRoot, "sync")
	result.AssertSuccess(t)

	if !sb.IsSymlink(filepath.Join(projectRoot, ".claude", "skills", "auto")) {
		t.Error("auto-detect should trigger project mode sync")
	}
}

func TestSyncProject_RelativeSymlinks(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	projectRoot := sb.SetupProjectDir("claude")
	sb.CreateProjectSkill(projectRoot, "my-skill", map[string]string{
		"SKILL.md": "# My Skill\n\nDescription here.",
	})

	result := sb.RunCLIInDir(projectRoot, "sync", "-p")
	result.AssertSuccess(t)

	link := filepath.Join(projectRoot, ".claude", "skills", "my-skill")
	if !sb.IsSymlink(link) {
		t.Fatal("skill should be a symlink")
	}

	// Project-mode symlinks must be relative (not absolute)
	target := sb.SymlinkTarget(link)
	if filepath.IsAbs(target) {
		t.Errorf("project-mode symlink should be relative, got absolute: %q", target)
	}

	// Verify the symlink resolves to the correct skill directory
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatalf("symlink should resolve: %v", err)
	}
	expected, _ := filepath.EvalSymlinks(filepath.Join(projectRoot, ".skillshare", "skills", "my-skill"))
	if resolved != expected {
		t.Errorf("resolved symlink = %q, want %q", resolved, expected)
	}
}
