//go:build !online

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/testutil"
)

func TestInstall_Global_FromConfig_SkipsExisting(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	// Pre-create the skill directory so it should be skipped
	sb.CreateSkill("my-skill", map[string]string{
		"SKILL.md": "---\nname: my-skill\n---\n# My Skill",
	})

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
skills:
  - name: my-skill
    source: github.com/user/repo
`)

	result := sb.RunCLI("install", "--global")

	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "1 skipped")
}

func TestInstall_Global_FromConfig_DryRun_TrackedRespectsQuiet(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	repoPath := filepath.Join(sb.Root, "tracked-src")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "SKILL.md"), []byte("# tracked"), 0644); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repoPath)

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	registryPath := filepath.Join(sb.SourcePath, "registry.yaml")
	if err := os.WriteFile(registryPath, []byte(`skills:
  - name: dryrun-track
    source: file://`+repoPath+`
    tracked: true
`), 0644); err != nil {
		t.Fatal(err)
	}

	result := sb.RunCLI("install", "--global", "--dry-run")

	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "Dry run — nothing was written")
	result.AssertOutputNotContains(t, "would clone")
}

func TestInstall_Global_NoSource_IncompatibleFlags(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets: {}
`)

	tests := []struct {
		name string
		args []string
	}{
		{"name flag", []string{"install", "--global", "--name", "foo"}},
		{"into flag", []string{"install", "--global", "--into", "sub"}},
		{"track flag", []string{"install", "--global", "--track"}},
		{"skill flag", []string{"install", "--global", "--skill", "x"}},
		{"exclude flag", []string{"install", "--global", "--exclude", "x"}},
		{"all flag", []string{"install", "--global", "--all"}},
		{"yes flag", []string{"install", "--global", "--yes"}},
		{"update flag", []string{"install", "--global", "--update"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sb.RunCLI(tc.args...)
			result.AssertFailure(t)
			result.AssertAnyOutputContains(t, "require a source argument")
		})
	}
}
