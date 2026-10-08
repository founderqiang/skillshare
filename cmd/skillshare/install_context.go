package main

import (
	"skillshare/internal/config"
	"skillshare/internal/install"
)

// Compile-time interface satisfaction checks.
var (
	_ install.InstallContext = (*globalInstallContext)(nil)
	_ install.InstallContext = (*projectInstallContext)(nil)
)

// ---------------------------------------------------------------------------
// globalInstallContext
// ---------------------------------------------------------------------------

// globalInstallContext implements install.InstallContext for global mode.
type globalInstallContext struct {
	cfg   *config.Config
	store *install.MetadataStore
}

func (g *globalInstallContext) SourcePath() string { return g.cfg.EffectiveSkillsSource() }
func (g *globalInstallContext) ConfigSkills() []install.SkillEntryDTO {
	return install.MetadataSkillEntries(g.store)
}
func (g *globalInstallContext) Reconcile() error {
	return config.ReconcileGlobalSkills(g.cfg, g.store)
}
func (g *globalInstallContext) PostInstallSkill(string) error { return nil }
func (g *globalInstallContext) Mode() string                  { return "global" }
func (g *globalInstallContext) GitLabHosts() []string         { return g.cfg.EffectiveGitLabHosts() }
func (g *globalInstallContext) AzureHosts() []string          { return g.cfg.EffectiveAzureHosts() }
func (g *globalInstallContext) CNBHosts() []string            { return g.cfg.EffectiveCNBHosts() }
func (g *globalInstallContext) GiteaHosts() []string          { return g.cfg.EffectiveGiteaHosts() }

// ---------------------------------------------------------------------------
// projectInstallContext
// ---------------------------------------------------------------------------

// projectInstallContext implements install.InstallContext for project mode.
type projectInstallContext struct {
	runtime *projectRuntime
}

func (p *projectInstallContext) SourcePath() string { return p.runtime.sourcePath }
func (p *projectInstallContext) ConfigSkills() []install.SkillEntryDTO {
	return config.ProjectSkillEntries(p.runtime.config.Skills)
}
func (p *projectInstallContext) Reconcile() error {
	return reconcileProjectRemoteSkills(p.runtime)
}
func (p *projectInstallContext) PostInstallSkill(displayName string) error {
	gitDir, prefix := config.ProjectGitignoreTarget(p.runtime.root, p.runtime.sourcePath)
	if gitDir == "" {
		return nil
	}
	return install.UpdateGitIgnore(gitDir, prefix+"/"+displayName)
}
func (p *projectInstallContext) Mode() string { return "project" }
func (p *projectInstallContext) GitLabHosts() []string {
	return p.runtime.config.EffectiveGitLabHosts()
}
func (p *projectInstallContext) AzureHosts() []string {
	return p.runtime.config.EffectiveAzureHosts()
}
func (p *projectInstallContext) CNBHosts() []string {
	return p.runtime.config.EffectiveCNBHosts()
}
func (p *projectInstallContext) GiteaHosts() []string {
	return p.runtime.config.EffectiveGiteaHosts()
}
