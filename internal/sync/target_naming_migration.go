package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"skillshare/internal/utils"
)

// selectActiveTargetNameForSync returns the entry name a skill syncs to. When
// the skill still has a managed entry under the name an earlier target naming
// gave it, that entry is renamed in place, unless the new name is taken.
func selectActiveTargetNameForSync(mode, targetPath string, skill ResolvedTargetSkill, manifest *Manifest, dryRun bool) (string, error) {
	desiredName := skill.TargetName
	legacyName, legacyNaming, err := findLegacyTargetEntry(mode, targetPath, skill, manifest)
	if err != nil || legacyName == "" {
		return desiredName, err
	}
	legacyPath := filepath.Join(targetPath, legacyName)

	desiredPath := filepath.Join(targetPath, desiredName)
	if _, err := os.Lstat(desiredPath); err == nil {
		fmt.Fprintf(DiagOutput,
			"Warning: kept legacy managed entry %s for target name %q because %s already exists\n",
			legacyName, desiredName, desiredName)
		return legacyName, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to inspect target entry %s: %w", desiredName, err)
	}

	if dryRun {
		fmt.Fprintf(DiagOutput, "[dry-run] Would rename managed target entry: %s -> %s\n", legacyName, desiredName)
		return legacyName, nil
	}

	if err := os.Rename(legacyPath, desiredPath); err != nil {
		return "", fmt.Errorf("failed to rename managed target entry %s -> %s: %w", legacyName, desiredName, err)
	}
	renameManifestEntry(manifest, legacyName, desiredName, legacyNaming)
	// Record the rename now: if a later skill fails, the manifest must still
	// name this entry, or the next sync would take it for a user folder.
	if err := WriteManifest(targetPath, manifest); err != nil {
		renameManifestEntry(manifest, desiredName, legacyName, legacyNaming)
		if rbErr := os.Rename(desiredPath, legacyPath); rbErr != nil {
			return "", fmt.Errorf("failed to record rename %s -> %s: %w (rename back also failed: %v)", legacyName, desiredName, err, rbErr)
		}
		return "", fmt.Errorf("failed to record rename %s -> %s: %w", legacyName, desiredName, err)
	}
	return desiredName, nil
}

// findLegacyTargetEntry returns the managed entry a skill holds under the name
// another target naming gave it, and that naming, or "" when there is none.
func findLegacyTargetEntry(mode, targetPath string, skill ResolvedTargetSkill, manifest *Manifest) (string, string, error) {
	for _, c := range targetNameCandidates(skill.Skill) {
		if c.name == "" || c.name == skill.TargetName {
			continue
		}
		legacyPath := filepath.Join(targetPath, c.name)
		info, err := os.Lstat(legacyPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", "", fmt.Errorf("failed to inspect legacy target entry %s: %w", c.name, err)
		}
		if isManagedLegacyTargetEntry(mode, legacyPath, info, skill, manifest, c) {
			return c.name, c.naming, nil
		}
	}
	return "", "", nil
}

func isManagedLegacyTargetEntry(mode, legacyPath string, info os.FileInfo, skill ResolvedTargetSkill, manifest *Manifest, c namedTarget) bool {
	switch mode {
	case "merge":
		return utils.IsSymlinkOrJunction(legacyPath) && isSymlinkToSource(legacyPath, skill.Skill.SourcePath)
	case "copy":
		if manifest == nil || !info.IsDir() || utils.IsSymlinkOrJunction(legacyPath) {
			return false
		}
		if _, managed := manifest.Managed[c.name]; !managed {
			return false
		}
		// A recorded naming says exactly which naming made the entry; older
		// manifests have none, so the first managed candidate wins.
		recorded := manifest.Naming[c.name]
		return recorded == "" || recorded == c.naming
	default:
		return false
	}
}

// renameManifestEntry moves an entry's records to newName. The naming record
// becomes oldNaming, the naming that produced the entry, so copy sync sees the
// naming change and refreshes the copy.
func renameManifestEntry(manifest *Manifest, oldName, newName, oldNaming string) {
	if manifest == nil || oldName == newName {
		return
	}
	if manifest.Managed == nil {
		manifest.Managed = make(map[string]string)
	}
	if manifest.Mtimes == nil {
		manifest.Mtimes = make(map[string]int64)
	}
	if manifest.Naming == nil {
		manifest.Naming = make(map[string]string)
	}

	if managed, ok := manifest.Managed[oldName]; ok {
		manifest.Managed[newName] = managed
		delete(manifest.Managed, oldName)
	}
	if mtime, ok := manifest.Mtimes[oldName]; ok {
		manifest.Mtimes[newName] = mtime
		delete(manifest.Mtimes, oldName)
	}
	delete(manifest.Naming, oldName)
	manifest.Naming[newName] = oldNaming
}
