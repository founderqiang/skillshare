//go:build !online

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skillshare/internal/testutil"
)

// setupDocsExtra writes a docs extra with index.md and draft.md synced to one
// target, with extraTarget appended to the target entry, and returns the
// target directory.
func setupDocsExtra(t *testing.T, sb *testutil.Sandbox, extraTarget string) string {
	t.Helper()
	sb.CreateSkill("placeholder", map[string]string{"SKILL.md": "# Placeholder"})
	claude := sb.CreateTarget("claude")

	docsSource := filepath.Join(filepath.Dir(sb.SourcePath), "extras", "docs")
	os.MkdirAll(docsSource, 0755)
	os.WriteFile(filepath.Join(docsSource, "index.md"), []byte("# Index"), 0644)
	os.WriteFile(filepath.Join(docsSource, "draft.md"), []byte("# Draft"), 0644)

	docsTarget := filepath.Join(sb.Home, ".claude", "docs")
	os.MkdirAll(docsTarget, 0755)

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claude + `
extras:
  - name: docs
    targets:
      - path: ` + docsTarget + extraTarget + `
`)
	return docsTarget
}

func TestExtras_AddExclude_NextSyncPrunesExcludedLink(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	docsTarget := setupDocsExtra(t, sb, "")
	sb.RunCLI("sync", "extras").AssertSuccess(t)

	sb.RunCLI("extras", "docs", "--add-exclude", "draft*", "-g").AssertSuccess(t)
	sb.RunCLI("sync", "extras").AssertSuccess(t)

	if _, err := os.Lstat(filepath.Join(docsTarget, "draft.md")); !os.IsNotExist(err) {
		t.Errorf("draft.md link should be pruned, lstat err = %v", err)
	}
	if !sb.IsSymlink(filepath.Join(docsTarget, "index.md")) {
		t.Error("index.md should stay linked")
	}
}

func TestExtras_Init_IncludeWritesTargetFilter(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	setupDocsExtra(t, sb, "")
	notesTarget := filepath.Join(sb.Home, ".claude", "notes")

	sb.RunCLI("extras", "init", "notes", "--target", notesTarget, "--include", "index.md", "-g").AssertSuccess(t)

	if cfg := sb.ReadFile(sb.ConfigPath); !strings.Contains(cfg, "include:\n          - index.md") {
		t.Errorf("expected include in config, got:\n%s", cfg)
	}
}

func TestExtras_Mode_SymlinkRejectedWithFilters(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	setupDocsExtra(t, sb, "\n        include: [index.md]")

	result := sb.RunCLI("extras", "docs", "--mode", "symlink", "-g")

	result.AssertFailure(t)
	result.AssertAnyOutputContains(t, "symlink")
}

func TestSyncExtras_WarnsUnmatchedInclude(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	setupDocsExtra(t, sb, "\n        include: [index.mdd]")

	result := sb.RunCLI("sync", "extras")

	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, `include "index.mdd" matches no file`)
}

func TestExtras_FlattenWithFilterNeedsTargetOnMultiTargetExtra(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	setupDocsExtra(t, sb, "\n      - path: "+filepath.Join(sb.Home, ".cursor", "docs"))

	result := sb.RunCLI("extras", "docs", "--flatten", "--add-exclude", "draft*", "-g")

	result.AssertFailure(t)
	result.AssertAnyOutputContains(t, "--target")
}
