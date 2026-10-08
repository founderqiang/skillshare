//go:build !online

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/install"
	"skillshare/internal/testutil"
)

func TestInstall_Discovery_DryRun_ShowsSkills(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	// Create a monorepo with multiple skills
	gitRepoPath := filepath.Join(sb.Root, "dry-run-repo")
	skill1Path := filepath.Join(gitRepoPath, "skill-one")
	skill2Path := filepath.Join(gitRepoPath, "skill-two")

	os.MkdirAll(skill1Path, 0755)
	os.MkdirAll(skill2Path, 0755)
	os.WriteFile(filepath.Join(skill1Path, "SKILL.md"), []byte("# One"), 0644)
	os.WriteFile(filepath.Join(skill2Path, "SKILL.md"), []byte("# Two"), 0644)

	initGitRepo(t, gitRepoPath)

	// Use file:// protocol to test git discovery with local repo
	result := sb.RunCLI("install", "file://"+gitRepoPath, "--dry-run")

	result.AssertSuccess(t)
	result.AssertOutputContains(t, "Found")
	result.AssertOutputContains(t, "skill-one")
	result.AssertOutputContains(t, "skill-two")
	result.AssertOutputContains(t, "dry-run")

	// Verify nothing was installed
	if sb.FileExists(filepath.Join(sb.SourcePath, "skill-one")) {
		t.Error("skill should not be installed in dry-run mode")
	}
}

// TestInstall_Discovery_HiddenDirs_FindsSkills tests that skills inside hidden
// directories (like .curated/, .system/) are discovered, while .git is skipped.

func TestInstall_Discovery_HiddenDirs_FindsSkills(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	// Create a repo with skills inside hidden directories (like openai/skills)
	gitRepoPath := filepath.Join(sb.Root, "hidden-dir-repo")
	curatedSkill := filepath.Join(gitRepoPath, ".curated", "pdf")
	systemSkill := filepath.Join(gitRepoPath, ".system", "figma")

	os.MkdirAll(curatedSkill, 0755)
	os.MkdirAll(systemSkill, 0755)
	os.WriteFile(filepath.Join(curatedSkill, "SKILL.md"), []byte("# PDF"), 0644)
	os.WriteFile(filepath.Join(systemSkill, "SKILL.md"), []byte("# Figma"), 0644)

	initGitRepo(t, gitRepoPath)

	// Use file:// protocol to test git discovery with local repo
	result := sb.RunCLI("install", "file://"+gitRepoPath, "--dry-run")

	result.AssertSuccess(t)
	result.AssertOutputContains(t, "Found")
	result.AssertOutputContains(t, "pdf")
	result.AssertOutputContains(t, "figma")
}

// TestInstall_Discovery_RootSkill tests that a repo with SKILL.md only at the
// root is correctly discovered (fixes issue #8).

func TestInstall_Discovery_RootSkill(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	// Create a git repo with SKILL.md at root only (no child skills)
	gitRepoPath := filepath.Join(sb.Root, "root-skill-repo")
	os.MkdirAll(gitRepoPath, 0755)
	os.WriteFile(filepath.Join(gitRepoPath, "SKILL.md"), []byte("---\nname: root-skill\n---\n# Root Skill"), 0644)
	os.WriteFile(filepath.Join(gitRepoPath, "README.md"), []byte("# Repo with root skill only"), 0644)

	initGitRepo(t, gitRepoPath)

	// Test discovery via the internal API directly
	source, err := install.ParseSource("file://" + gitRepoPath)
	if err != nil {
		t.Fatalf("ParseSource() error = %v", err)
	}

	discovery, err := install.DiscoverFromGit(source)
	if err != nil {
		t.Fatalf("DiscoverFromGit() error = %v", err)
	}
	defer install.CleanupDiscovery(discovery)

	// Should find exactly 1 skill (the root)
	if len(discovery.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d: %+v", len(discovery.Skills), discovery.Skills)
	}

	skill := discovery.Skills[0]
	if skill.Path != "." {
		t.Errorf("skill Path = %q, want %q", skill.Path, ".")
	}
	// Name should be derived from repo name, not temp dir
	if skill.Name != "root-skill-repo" {
		t.Errorf("skill Name = %q, want %q", skill.Name, "root-skill-repo")
	}
}

func TestInstall_FileURLDot_DryRun_UsesRepoName(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	gitRepoPath := filepath.Join(sb.Root, "root-skill-repo")
	os.MkdirAll(gitRepoPath, 0755)
	os.WriteFile(filepath.Join(gitRepoPath, "SKILL.md"), []byte("# Root Skill"), 0644)

	initGitRepo(t, gitRepoPath)

	result := sb.RunCLI("install", "file://"+gitRepoPath+"/.", "--dry-run")
	result.AssertSuccess(t)
	result.AssertOutputContains(t, "root-skill-repo")
	result.AssertOutputNotContains(t, "── . ──")
}

func TestInstall_Discovery_SingleSkill_CustomName(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	gitRepoPath := filepath.Join(sb.Root, "root-skill-repo")
	os.MkdirAll(gitRepoPath, 0755)
	os.WriteFile(filepath.Join(gitRepoPath, "SKILL.md"), []byte("# Root Skill"), 0644)

	initGitRepo(t, gitRepoPath)

	result := sb.RunCLI("install", "file://"+gitRepoPath, "--name", "haha")
	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "haha")

	if !sb.FileExists(filepath.Join(sb.SourcePath, "haha", "SKILL.md")) {
		t.Fatal("expected skill to be installed with custom name")
	}

	if sb.FileExists(filepath.Join(sb.SourcePath, "root-skill-repo")) {
		t.Fatal("repo-derived name should not be installed when --name is provided")
	}
}

func TestInstall_Discovery_MultipleSkills_NameFlagErrors(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	gitRepoPath := filepath.Join(sb.Root, "multi-skill-repo")
	skillAPath := filepath.Join(gitRepoPath, "skill-a")
	skillBPath := filepath.Join(gitRepoPath, "skill-b")
	os.MkdirAll(skillAPath, 0755)
	os.MkdirAll(skillBPath, 0755)
	os.WriteFile(filepath.Join(skillAPath, "SKILL.md"), []byte("# Skill A"), 0644)
	os.WriteFile(filepath.Join(skillBPath, "SKILL.md"), []byte("# Skill B"), 0644)

	initGitRepo(t, gitRepoPath)

	result := sb.RunCLI("install", "file://"+gitRepoPath, "--name", "renamed")
	result.AssertFailure(t)
	result.AssertAnyOutputContains(t, "--name can only be used when exactly one skill is discovered")
}

// createMultiSkillGitRepo creates a local git repo with multiple skills for testing.
// Returns the git repo path (usable with file:// protocol).

// TestInstall_SubdirFuzzyResolve_Discovery tests fuzzy resolution through the
// DiscoverFromGitSubdir path (multi-skill subdir with discovery).

func TestInstall_SubdirFuzzyResolve_Discovery(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + "\ntargets: {}\n")

	// Create a git repo: skills/frontend/SKILL.md + skills/frontend/child-a/SKILL.md
	gitRepoPath := filepath.Join(sb.Root, "fuzzy-discover-repo")
	parentPath := filepath.Join(gitRepoPath, "skills", "frontend")
	childPath := filepath.Join(parentPath, "child-a")
	os.MkdirAll(parentPath, 0755)
	os.MkdirAll(childPath, 0755)
	os.WriteFile(filepath.Join(parentPath, "SKILL.md"), []byte("# Frontend"), 0644)
	os.WriteFile(filepath.Join(childPath, "SKILL.md"), []byte("# Child A"), 0644)

	initGitRepo(t, gitRepoPath)

	// Construct source with subdir "frontend" (simulates GitHub URL like owner/repo/frontend)
	source := &install.Source{
		Type:     install.SourceTypeGitHTTPS,
		Raw:      "file://" + gitRepoPath + "/frontend",
		CloneURL: "file://" + gitRepoPath,
		Subdir:   "frontend",
		Name:     "frontend",
	}

	discovery, err := install.DiscoverFromGitSubdir(source)
	if err != nil {
		t.Fatalf("DiscoverFromGitSubdir() error = %v", err)
	}
	defer install.CleanupDiscovery(discovery)

	if len(discovery.Skills) == 0 {
		t.Fatal("expected at least 1 skill discovered")
	}

	// Should find the root skill of the subdir + child
	var foundRoot bool
	for _, sk := range discovery.Skills {
		if sk.Path == "." {
			foundRoot = true
		}
	}
	if !foundRoot {
		t.Error("expected root skill of resolved subdir to be discovered")
	}
}

// --- Feature #17: .skillignore ---
