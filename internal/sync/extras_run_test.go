package sync

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/config"
)

func runTestOptions() ExtraRunOptions {
	return ExtraRunOptions{
		ResolvePath: func(p string) string { return p },
		ResolveExtension: func(ext string) (*ExtensionSpec, error) {
			return nil, errors.New("no extension " + ext)
		},
	}
}

func TestRunExtraTargets_MissingSourceSkip(t *testing.T) {
	src := filepath.Join(t.TempDir(), "missing")
	extra := config.ExtraConfig{Name: "rules", Targets: []config.ExtraTargetConfig{{Path: t.TempDir()}}}

	run := RunExtraTargets(extra, src, runTestOptions())

	if !run.SourceMissing || len(run.Targets) != 0 {
		t.Fatalf("expected skipped extra, got %+v", run)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source must not be created, stat err = %v", err)
	}
}

func TestRunExtraTargets_SkipsAgentOverlap(t *testing.T) {
	src, tgt := setupExtrasTest(t, map[string]string{"a.md": "a"})
	extra := config.ExtraConfig{Name: "agents", Targets: []config.ExtraTargetConfig{{Path: tgt}}}
	opts := runTestOptions()
	opts.AgentTargetPaths = map[string]bool{filepath.Clean(tgt): true}

	run := RunExtraTargets(extra, src, opts)

	if got := run.Targets[0]; got.SkippedBy != "agents" || got.Result != nil {
		t.Fatalf("expected target skipped by agents, got %+v", got)
	}
}

// Only the "agents" extra's target at an agents sync path is skipped.
func TestRunExtraTargets_AgentOverlapIsPerTargetAndPerName(t *testing.T) {
	src, overlap := setupExtrasTest(t, map[string]string{"a.md": "a"})
	other := t.TempDir()
	opts := runTestOptions()
	opts.AgentTargetPaths = map[string]bool{filepath.Clean(overlap): true}
	targets := []config.ExtraTargetConfig{{Path: overlap}, {Path: other}}

	agents := RunExtraTargets(config.ExtraConfig{Name: "agents", Targets: targets}, src, opts).Targets
	if agents[0].SkippedBy != "agents" || agents[1].SkippedBy != "" || agents[1].Result == nil {
		t.Errorf("agents extra: want only the overlapping target skipped, got %+v", agents)
	}
	rules := RunExtraTargets(config.ExtraConfig{Name: "rules", Targets: targets[:1]}, src, opts).Targets
	if rules[0].SkippedBy != "" || rules[0].Result == nil {
		t.Errorf("rules extra at an agents path must sync, got %+v", rules[0])
	}
}

func TestAgentTargetPaths(t *testing.T) {
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "agents")

	if got := AgentTargetPaths(source, []string{target}); got != nil {
		t.Errorf("no agents in the source: want nil, got %v", got)
	}

	if err := os.WriteFile(filepath.Join(source, "helper.md"), []byte("# Helper"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := AgentTargetPaths(source, []string{target + "/", ""})
	if len(got) != 1 || !got[target] {
		t.Errorf("want only the cleaned %s, got %v", target, got)
	}
}

func TestRunExtraTargets_ReportsModeAndExtensionErrors(t *testing.T) {
	src, tgt := setupExtrasTest(t, map[string]string{"a.md": "a"})
	extra := config.ExtraConfig{Name: "rules", Targets: []config.ExtraTargetConfig{{Path: tgt, Mode: "merge", Extension: "x"}}}

	got := RunExtraTargets(extra, src, runTestOptions()).Targets[0]

	if got.ModeErr == nil || got.ExtensionErr == nil || got.Result != nil || got.Mode != "merge" {
		t.Fatalf("expected both errors, no sync and configured mode, got %+v", got)
	}
}
