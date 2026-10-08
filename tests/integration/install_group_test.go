//go:build !online

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skillshare/internal/install"
	"skillshare/internal/testutil"
)

func TestInstall_Into_RecordsGroupField(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	// Create a local skill
	localSkill := filepath.Join(sb.Root, "pdf-skill")
	os.MkdirAll(localSkill, 0755)
	os.WriteFile(filepath.Join(localSkill, "SKILL.md"), []byte("# PDF Skill"), 0644)

	// Install with --into frontend
	result := sb.RunCLI("install", localSkill, "--into", "frontend")
	result.AssertSuccess(t)

	// Verify skill was installed into subdirectory
	if !sb.FileExists(filepath.Join(sb.SourcePath, "frontend", "pdf-skill", "SKILL.md")) {
		t.Error("skill should be installed to source/frontend/pdf-skill/")
	}

	// Read centralized metadata and verify group field
	store, err := install.LoadMetadata(sb.SourcePath)
	if err != nil {
		t.Fatalf("failed to load metadata: %v", err)
	}
	// Full-path key: "frontend/pdf-skill" (not just basename "pdf-skill")
	entry := store.Get("frontend/pdf-skill")
	if entry == nil {
		t.Fatal("expected metadata entry for 'frontend/pdf-skill'")
	}
	if entry.Group != "frontend" {
		t.Errorf("metadata group = %q, want %q", entry.Group, "frontend")
	}
}

func TestInstall_Into_MultiLevel_RecordsGroupField(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	localSkill := filepath.Join(sb.Root, "ui-skill")
	os.MkdirAll(localSkill, 0755)
	os.WriteFile(filepath.Join(localSkill, "SKILL.md"), []byte("# UI Skill"), 0644)

	// Install with multi-level --into
	result := sb.RunCLI("install", localSkill, "--into", "frontend/vue")
	result.AssertSuccess(t)

	// Read centralized metadata and verify group field
	store, err := install.LoadMetadata(sb.SourcePath)
	if err != nil {
		t.Fatalf("failed to load metadata: %v", err)
	}
	// Full-path key: "frontend/vue/ui-skill"
	entry := store.Get("frontend/vue/ui-skill")
	if entry == nil {
		t.Fatal("expected metadata entry for 'frontend/vue/ui-skill'")
	}
	if entry.Group != "frontend/vue" {
		t.Errorf("metadata group = %q, want %q", entry.Group, "frontend/vue")
	}
}

func TestInstall_LegacySlashName_BackwardCompat(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	// Write config with legacy format (name contains slash, no group field)
	// This is the format that existed before the group field was added
	sourceSkill := filepath.Join(sb.Root, "source-pdf")
	os.MkdirAll(sourceSkill, 0755)
	os.WriteFile(filepath.Join(sourceSkill, "SKILL.md"), []byte("# PDF"), 0644)

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
skills:
  - name: frontend/pdf
    source: ` + sourceSkill + `
`)

	// Config-based install with legacy slash name should still work
	result := sb.RunCLI("install")
	result.AssertSuccess(t)

	// Verify skill was installed correctly
	if !sb.FileExists(filepath.Join(sb.SourcePath, "frontend", "pdf", "SKILL.md")) {
		t.Error("legacy slash-name install should place skill at frontend/pdf/")
	}
}

func TestInstallProject_Into_RecordsGroupField(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	projectRoot := sb.SetupProjectDir("claude")

	// Create a source skill
	sourceSkill := filepath.Join(sb.Root, "my-skill")
	os.MkdirAll(sourceSkill, 0755)
	os.WriteFile(filepath.Join(sourceSkill, "SKILL.md"), []byte("---\nname: my-skill\n---\n# My Skill"), 0644)

	// Install with --into in project mode
	result := sb.RunCLIInDir(projectRoot, "install", sourceSkill, "--into", "tools", "-p")
	result.AssertSuccess(t)

	// Read centralized metadata and verify group field
	store, err := install.LoadMetadata(filepath.Join(projectRoot, ".skillshare", "skills"))
	if err != nil {
		t.Fatalf("failed to load metadata: %v", err)
	}
	// Full-path key: "tools/my-skill"
	entry := store.Get("tools/my-skill")
	if entry == nil {
		t.Fatal("expected metadata entry for 'tools/my-skill'")
	}
	if entry.Group != "tools" {
		t.Errorf("metadata group = %q, want %q", entry.Group, "tools")
	}
}

// assertGroupedMetadata checks the skills root records name and that no stray
// metadata file sits in the group folder instead.
func assertGroupedMetadata(t *testing.T, skillsRoot, name string) {
	t.Helper()
	store, err := install.LoadMetadata(skillsRoot)
	if err != nil {
		t.Fatal(err)
	}
	if store.Get(name) == nil {
		t.Errorf("no metadata entry %q at the skills root, have %v", name, store.List())
	}
	stray := filepath.Join(skillsRoot, filepath.Dir(name), install.MetadataFileName)
	if _, err := os.Stat(stray); err == nil {
		t.Errorf("metadata was written into the group folder: %s", stray)
	}
}

func TestInstallProject_IntoGitSource_RecordedInConfig(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	source, _ := lockRemote(t, sb)
	project := sb.SetupProjectDir("claude")

	sb.RunCLIInDir(project, "install", source, "--into", "frontend", "-p").AssertSuccess(t)

	assertGroupedMetadata(t, filepath.Join(project, ".skillshare", "skills"), "frontend/demo")
	cfg, err := os.ReadFile(filepath.Join(project, ".skillshare", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "group: frontend") {
		t.Errorf("config.yaml does not list the grouped skill, so teammates never install it:\n%s", cfg)
	}
}

func TestInstallProject_FromConfig_GroupedSkill(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	source, _ := lockRemote(t, sb)
	project := sb.SetupProjectDir("claude")
	sb.WriteProjectConfig(project, "targets:\n  - claude\nskills:\n  - name: demo\n    source: "+source+"\n    group: frontend\n")

	sb.RunCLIInDir(project, "install", "-p").AssertSuccess(t)

	assertGroupedMetadata(t, filepath.Join(project, ".skillshare", "skills"), "frontend/demo")
}

func TestInstallGlobal_FromConfig_GroupedSkill(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()
	source, _ := lockRemote(t, sb)
	sb.WriteConfig("source: " + sb.SourcePath + "\ntargets: {}\nskills:\n  - name: demo\n    source: " + source + "\n    group: frontend\n")

	sb.RunCLI("install", "-g").AssertSuccess(t)

	assertGroupedMetadata(t, sb.SourcePath, "frontend/demo")
}
