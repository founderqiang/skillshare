//go:build !online

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/testutil"
)

func TestSyncProject_CustomSkillsSource(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")

	// Create skill in custom source directory (not .skillshare/skills)
	customSkillsDir := filepath.Join(projectRoot, "docs", "skills")
	os.MkdirAll(filepath.Join(customSkillsDir, "custom-skill"), 0755)
	os.WriteFile(filepath.Join(customSkillsDir, "custom-skill", "SKILL.md"), []byte("# Custom"), 0644)

	sb.WriteProjectConfig(projectRoot, `sources:
  skills: ./docs/skills
targets:
  - claude
`)

	result := sb.RunCLIInDir(projectRoot, "sync", "-p")
	result.AssertSuccess(t)

	link := filepath.Join(projectRoot, ".claude", "skills", "custom-skill")
	if !sb.IsSymlink(link) {
		t.Error("sync should create symlink from custom skills source")
	}
}

func TestStatusProject_CustomSkillsSource(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")

	customSkillsDir := filepath.Join(projectRoot, "docs", "skills")
	os.MkdirAll(filepath.Join(customSkillsDir, "my-skill"), 0755)
	os.WriteFile(filepath.Join(customSkillsDir, "my-skill", "SKILL.md"), []byte("# My Skill"), 0644)

	sb.WriteProjectConfig(projectRoot, `sources:
  skills: ./docs/skills
targets:
  - claude
`)

	result := sb.RunCLIInDir(projectRoot, "status", "-p")
	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "docs/skills")
}

// TestUninstallProject_MalformedConfig_FailsClosed verifies that with a
// malformed project config, uninstall returns an error instead of silently
// falling back to .skillshare/skills (which could trash the wrong directory
// when sources.skills is configured).
func TestUninstallProject_MalformedConfig_FailsClosed(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")
	sb.CreateProjectSkill(projectRoot, "stay-put", map[string]string{
		"SKILL.md": "# Stay",
	})
	// Malformed YAML — targets must be a list, not a scalar
	sb.WriteProjectConfig(projectRoot, "targets: not-a-list\n")

	result := sb.RunCLIInDir(projectRoot, "uninstall", "stay-put", "--force", "-p")
	result.AssertFailure(t)
	result.AssertAnyOutputContains(t, "failed to load project config")

	// Skill must still exist — uninstall should have aborted
	if !sb.FileExists(filepath.Join(projectRoot, ".skillshare", "skills", "stay-put", "SKILL.md")) {
		t.Error("skill should remain when uninstall fails closed on malformed config")
	}
}

// TestNewProject_MalformedConfig_FailsClosed verifies that `new` aborts
// rather than creating a skill in the default directory when project config
// is malformed.
func TestNewProject_MalformedConfig_FailsClosed(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")
	sb.WriteProjectConfig(projectRoot, "skills: bad-not-list\n")

	result := sb.RunCLIInDir(projectRoot, "new", "should-not-exist", "-p")
	result.AssertFailure(t)
	result.AssertAnyOutputContains(t, "failed to load project config")

	if sb.FileExists(filepath.Join(projectRoot, ".skillshare", "skills", "should-not-exist")) {
		t.Error("new should not create skill when project config fails to load")
	}
}

// TestCheckProject_MalformedConfig_FailsClosed verifies that `check -p`
// returns an error on malformed config rather than scanning the default
// .skillshare/skills directory.
func TestCheckProject_MalformedConfig_FailsClosed(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")
	sb.WriteProjectConfig(projectRoot, "targets: bad\n")

	result := sb.RunCLIInDir(projectRoot, "check", "-p")
	result.AssertFailure(t)
	result.AssertAnyOutputContains(t, "failed to load project config")
}

// TestSyncProject_SourceAliasesTarget_Rejected verifies that the validator
// rejects a config where sources.skills resolves to a target path — without
// this guard, `sync --force` could delete the configured source directory.
func TestSyncProject_SourceAliasesTarget_Rejected(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")
	// Pre-create the alias directory so source existence check passes
	os.MkdirAll(filepath.Join(projectRoot, ".claude", "skills"), 0755)
	sb.WriteProjectConfig(projectRoot, `sources:
  skills: .claude/skills
targets:
  - claude
`)

	result := sb.RunCLIInDir(projectRoot, "sync", "-p")
	result.AssertFailure(t)
	result.AssertAnyOutputContains(t, "overlaps")
}

func TestListProject_CustomSkillsSource(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")

	customSkillsDir := filepath.Join(projectRoot, "docs", "skills")
	os.MkdirAll(filepath.Join(customSkillsDir, "listed-skill"), 0755)
	os.WriteFile(filepath.Join(customSkillsDir, "listed-skill", "SKILL.md"), []byte("---\nname: listed-skill\ndescription: test\n---\n# Listed"), 0644)

	sb.WriteProjectConfig(projectRoot, `sources:
  skills: ./docs/skills
targets:
  - claude
`)

	result := sb.RunCLIInDir(projectRoot, "list", "-p", "--no-tui")
	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "listed-skill")
}
