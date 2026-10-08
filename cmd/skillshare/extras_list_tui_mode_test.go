package main

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/config"
)

func TestExtrasListTUI_SetModeRejectsSymlinkForFilteredTarget(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".skillshare"), 0755)
	os.WriteFile(filepath.Join(dir, ".skillshare", "config.yaml"), []byte(`extras:
  - name: docs
    targets:
      - path: .claude/docs
        include: [index.md]
`), 0644)
	m := extrasListTUIModel{projCfg: &config.ProjectConfig{}, cwd: dir}

	if _, err := m.doSetMode("docs", ".claude/docs", "symlink"); err == nil {
		t.Fatal("expected symlink to be refused for a target with filters")
	}
}
