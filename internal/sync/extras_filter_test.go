package sync

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"skillshare/internal/config"
)

var filterTestFiles = []string{"draft.md", "guides/draft-2.md", "guides/setup.md", "images/logo.png", "index.md"}

func TestFilterExtraFiles_GitignoreSemantics(t *testing.T) {
	tests := []struct {
		name             string
		include, exclude []string
		want             []string
	}{
		{"no filters keeps all", nil, nil, filterTestFiles},
		{"bare name matches at any depth", []string{"setup.md"}, nil, []string{"guides/setup.md"}},
		{"glob without slash matches at any depth", nil, []string{"draft*"}, []string{"guides/setup.md", "images/logo.png", "index.md"}},
		{"trailing slash covers a folder", nil, []string{"images/"}, []string{"draft.md", "guides/draft-2.md", "guides/setup.md", "index.md"}},
		{"double star matches nested files", []string{"**/*.png"}, nil, []string{"images/logo.png"}},
		{"exclude applies after include", []string{"guides/"}, []string{"draft*"}, []string{"guides/setup.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterExtraFiles(filterTestFiles, tt.include, tt.exclude)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnmatchedIncludes_ReportsPatternWithNoFile(t *testing.T) {
	unmatched := UnmatchedIncludes(filterTestFiles, []string{"index.md", "index.mdd"})

	if !slices.Equal(unmatched, []string{"index.mdd"}) {
		t.Errorf("unmatched = %v, want [index.mdd]", unmatched)
	}
}

func filterTestExtra(tgt string, target config.ExtraTargetConfig) config.ExtraConfig {
	target.Path = tgt
	return config.ExtraConfig{Name: "docs", Targets: []config.ExtraTargetConfig{target}}
}

func TestRunExtraTargets_IncludeSyncsOnlyMatches(t *testing.T) {
	src, tgt := setupExtrasTest(t, map[string]string{"index.md": "i", "draft.md": "d"})

	run := RunExtraTargets(filterTestExtra(tgt, config.ExtraTargetConfig{Include: []string{"index.md"}}), src, runTestOptions())

	if r := run.Targets[0]; r.Err != nil || r.Result.Synced != 1 {
		t.Fatalf("expected one synced file, got %+v", r)
	}
	if _, err := os.Lstat(filepath.Join(tgt, "draft.md")); !os.IsNotExist(err) {
		t.Errorf("draft.md must not be synced, lstat err = %v", err)
	}
}

func TestRunExtraTargets_NarrowedFilterPrunesMergeLinks(t *testing.T) {
	src, tgt := setupExtrasTest(t, map[string]string{"index.md": "i", "draft.md": "d"})
	RunExtraTargets(filterTestExtra(tgt, config.ExtraTargetConfig{}), src, runTestOptions())

	run := RunExtraTargets(filterTestExtra(tgt, config.ExtraTargetConfig{Exclude: []string{"draft*"}}), src, runTestOptions())

	if r := run.Targets[0]; r.Err != nil || r.Result.Pruned != 1 {
		t.Fatalf("expected the draft.md link pruned, got %+v", r)
	}
}

func TestRunExtraTargets_CopyKeepsFilteredOutFiles(t *testing.T) {
	src, tgt := setupExtrasTest(t, map[string]string{"index.md": "i", "draft.md": "d"})
	RunExtraTargets(filterTestExtra(tgt, config.ExtraTargetConfig{Mode: "copy"}), src, runTestOptions())

	RunExtraTargets(filterTestExtra(tgt, config.ExtraTargetConfig{Mode: "copy", Exclude: []string{"draft*"}}), src, runTestOptions())

	if _, err := os.Stat(filepath.Join(tgt, "draft.md")); err != nil {
		t.Errorf("copy mode must keep draft.md, stat err = %v", err)
	}
}

func TestRunExtraTargets_WarnsUnmatchedInclude(t *testing.T) {
	src, tgt := setupExtrasTest(t, map[string]string{"index.md": "i"})

	run := RunExtraTargets(filterTestExtra(tgt, config.ExtraTargetConfig{Include: []string{"index.mdd"}}), src, runTestOptions())

	if r := run.Targets[0]; !slices.Contains(r.Result.Warnings, UnmatchedIncludeWarning("index.mdd")) {
		t.Errorf("expected unmatched include warning, got %v", r.Result.Warnings)
	}
}

func TestCollectExtraFiles_SkipsFilteredOutFiles(t *testing.T) {
	src, tgt := setupExtrasTest(t, nil)
	os.WriteFile(filepath.Join(tgt, "draft.md"), []byte("d"), 0644)

	result, err := CollectExtraFiles(src, tgt, "", false, false, false, "", nil, []string{"draft*"})

	if err != nil || result.Collected != 0 {
		t.Fatalf("expected nothing collected, got %+v, err %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(src, "draft.md")); !os.IsNotExist(err) {
		t.Errorf("draft.md must stay out of the source, stat err = %v", err)
	}
}
