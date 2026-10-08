package sync

import (
	"fmt"

	"skillshare/internal/config"
	"skillshare/internal/skillignore"
)

// extraFilter decides which source files (paths relative to the extra's
// source) a target syncs. include and exclude use .gitignore syntax: a pattern
// without "/" matches at any depth and "dir/" covers a folder.
type extraFilter struct{ inc, exc *skillignore.Matcher }

func newExtraFilter(include, exclude []string) extraFilter {
	return extraFilter{skillignore.Compile(include), skillignore.Compile(exclude)}
}

func (f extraFilter) included(rel string) bool {
	return !f.inc.HasRules() || f.inc.Match(rel, false)
}

func (f extraFilter) keeps(rel string) bool {
	return f.included(rel) && !f.exc.Match(rel, false)
}

// FilterExtraFiles keeps the source files that pass include and exclude.
func FilterExtraFiles(files, include, exclude []string) []string {
	if len(include) == 0 && len(exclude) == 0 {
		return files
	}
	f := newExtraFilter(include, exclude)
	var kept []string
	for _, rel := range files {
		if f.keeps(rel) {
			kept = append(kept, rel)
		}
	}
	return kept
}

// UnmatchedIncludes returns the include patterns that select no file at all.
func UnmatchedIncludes(files, include []string) []string {
	var unmatched []string
	for _, p := range include {
		m := skillignore.Compile([]string{p})
		if !m.HasRules() {
			continue // blank or comment
		}
		if !matchesAnyFile(m, files) {
			unmatched = append(unmatched, p)
		}
	}
	return unmatched
}

// ExtraTargetFiles narrows an extra's source files to the ones target syncs.
func ExtraTargetFiles(files []string, target config.ExtraTargetConfig) []string {
	return FilterExtraFiles(files, target.Include, target.Exclude)
}

// ExtraFilterEntry is one source file in a filter preview. Status is
// "synced", "not_included", or "excluded"; Reason names the exclude pattern
// that dropped an excluded file.
type ExtraFilterEntry struct {
	File   string `json:"file"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// PreviewExtraFilter reports what include and exclude do to each source file.
func PreviewExtraFilter(files, include, exclude []string) []ExtraFilterEntry {
	f := newExtraFilter(include, exclude)
	excludes := make([]*skillignore.Matcher, len(exclude))
	for i, p := range exclude {
		excludes[i] = skillignore.Compile([]string{p})
	}
	entries := make([]ExtraFilterEntry, 0, len(files))
	for _, rel := range files {
		e := ExtraFilterEntry{File: rel, Status: "synced"}
		switch {
		case !f.included(rel):
			e.Status = "not_included"
		case !f.keeps(rel):
			e.Status = "excluded"
			for i, m := range excludes {
				if m.Match(rel, false) {
					e.Reason = exclude[i]
					break
				}
			}
		}
		entries = append(entries, e)
	}
	return entries
}

// UnmatchedIncludeWarning is the sync warning for an include pattern that
// selects no file.
func UnmatchedIncludeWarning(pattern string) string {
	return fmt.Sprintf("include %q matches no file", pattern)
}

func matchesAnyFile(m *skillignore.Matcher, files []string) bool {
	for _, rel := range files {
		if m.Match(rel, false) {
			return true
		}
	}
	return false
}

// discoverExtraTargetFiles lists the source files one target syncs, with a
// warning for each include pattern that selects nothing.
func discoverExtraTargetFiles(sourcePath string, include, exclude []string) ([]string, []string, error) {
	files, err := DiscoverExtraFiles(sourcePath)
	if err != nil {
		return nil, nil, err
	}
	kept := FilterExtraFiles(files, include, exclude)
	unmatched := UnmatchedIncludes(files, include)
	warnings := make([]string, 0, len(unmatched))
	for _, p := range unmatched {
		warnings = append(warnings, UnmatchedIncludeWarning(p))
	}
	return kept, warnings, nil
}
