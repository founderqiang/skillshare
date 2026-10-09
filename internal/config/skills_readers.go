package config

import (
	"path/filepath"
	"sort"
)

// SkillsAlsoReadBy names the configured targets with skills switched off whose
// tool still reads the skills folder of target name, as its own default folder
// or one it also scans (universal's ~/.agents/skills is read by gemini and pi).
// Only built-in tools are known readers; custom targets never appear. It
// returns nil when name has skills off itself. projectRoot is "" in global
// mode; in project mode, target paths are absolute (ResolveProjectTargets).
func SkillsAlsoReadBy(targets map[string]TargetConfig, name, projectRoot string) []string {
	tc, ok := targets[name]
	if !ok || !tc.SkillsConfig().IsEnabled() {
		return nil
	}
	folder := skillsFolder(tc)
	if folder == "" {
		return nil
	}
	var names []string
	for other, otc := range targets {
		if other == name || otc.SkillsConfig().IsEnabled() {
			continue
		}
		if toolReadsSkills(other, folder, projectRoot) {
			names = append(names, other)
		}
	}
	sort.Strings(names)
	return names
}

// SkillsSharedWith names the other targets with skills on that write to the
// same skills folder as target name, or nil when name has skills off itself.
func SkillsSharedWith(targets map[string]TargetConfig, name string) []string {
	tc, ok := targets[name]
	if !ok || !tc.SkillsConfig().IsEnabled() {
		return nil
	}
	folder := skillsFolder(tc)
	if folder == "" {
		return nil
	}
	var names []string
	for other, otc := range targets {
		if other != name && otc.SkillsConfig().IsEnabled() && skillsFolder(otc) == folder {
			names = append(names, other)
		}
	}
	sort.Strings(names)
	return names
}

// SkillsReadFrom names the configured targets with skills on whose skills
// folder the built-in tool name reads. name need not be configured, so the
// add-target list can say a tool already sees another target's skills.
// A folder name itself syncs skills into is not another source.
func SkillsReadFrom(targets map[string]TargetConfig, name, projectRoot string) []string {
	var own string
	if tc, ok := targets[name]; ok && tc.SkillsConfig().IsEnabled() {
		own = skillsFolder(tc)
	}
	var names []string
	for other, otc := range targets {
		if other == name || !otc.SkillsConfig().IsEnabled() {
			continue
		}
		if folder := skillsFolder(otc); folder != "" && folder != own && toolReadsSkills(name, folder, projectRoot) {
			names = append(names, other)
		}
	}
	sort.Strings(names)
	return names
}

func skillsFolder(tc TargetConfig) string {
	raw := tc.SkillsConfig().Path
	if raw == "" {
		return ""
	}
	return filepath.Clean(ExpandPath(raw))
}

// toolReadsSkills reports whether the built-in tool name reads folder.
func toolReadsSkills(name, folder, projectRoot string) bool {
	for _, p := range RuntimeScanPaths(name, projectRoot != "") {
		if projectRoot != "" && !filepath.IsAbs(p) {
			p = filepath.Join(projectRoot, p)
		}
		if filepath.Clean(p) == folder {
			return true
		}
	}
	return false
}
