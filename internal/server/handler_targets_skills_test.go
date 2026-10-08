package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"skillshare/internal/config"
)

// newSkillsOffServer configures universal (~/.agents/skills) and gemini
// (~/.gemini/skills) under a temp HOME, with alpha linked into gemini.
func newSkillsOffServer(t *testing.T) (s *Server, sourceDir, universal, gemini string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	universal = filepath.Join(home, ".agents", "skills")
	gemini = filepath.Join(home, ".gemini", "skills")
	s, sourceDir = newTestServerWithTargets(t, map[string]string{"universal": universal, "gemini": gemini})
	addSkill(t, sourceDir, "alpha")
	if err := os.Symlink(filepath.Join(sourceDir, "alpha"), filepath.Join(gemini, "alpha")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gemini, "mine"), 0755); err != nil {
		t.Fatal(err)
	}
	return s, sourceDir, universal, gemini
}

func serve(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	s.handler.ServeHTTP(rr, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rr
}

type skillsTargetItem struct {
	Name             string   `json:"name"`
	SkillsEnabled    bool     `json:"skillsEnabled"`
	SkillsReadFrom   []string `json:"skillsReadFrom"`
	SkillsAlsoReadBy []string `json:"skillsAlsoReadBy"`
	SkillsSharedWith []string `json:"skillsSharedWith"`
	LinkedCount      int      `json:"linkedCount"`
}

func listSkillsTargets(t *testing.T, s *Server) map[string]skillsTargetItem {
	t.Helper()
	rr := serve(t, s, http.MethodGet, "/api/targets", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Targets []skillsTargetItem `json:"targets"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	out := map[string]skillsTargetItem{}
	for _, item := range resp.Targets {
		out[item.Name] = item
	}
	return out
}

func TestSkillsOffPreview_ListsWithoutWriting(t *testing.T) {
	s, _, _, gemini := newSkillsOffServer(t)

	rr := serve(t, s, http.MethodGet, "/api/targets/gemini/skills-off-preview", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Remove     []string `json:"remove"`
		Keep       []string `json:"keep"`
		SharedWith string   `json:"sharedWith"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if !slices.Equal(resp.Remove, []string{"alpha"}) || !slices.Equal(resp.Keep, []string{"mine"}) || resp.SharedWith != "" {
		t.Errorf("preview = %+v", resp)
	}
	if _, err := os.Lstat(filepath.Join(gemini, "alpha")); err != nil {
		t.Error("preview must not remove links")
	}
}

func TestSkillsOffPreview_SharedFolder(t *testing.T) {
	shared := filepath.Join(t.TempDir(), "agents-skills")
	s, _ := newTestServerWithTargets(t, map[string]string{"universal": shared, "codex": shared})

	rr := serve(t, s, http.MethodGet, "/api/targets/codex/skills-off-preview", "")
	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["sharedWith"] != "universal" {
		t.Errorf("preview = %s", rr.Body.String())
	}
}

func TestListTargets_NamesTargetsSharingASkillsFolder(t *testing.T) {
	shared := filepath.Join(t.TempDir(), "agents", "skills")
	s, _ := newTestServerWithTargets(t, map[string]string{"universal": shared, "codex": shared, "claude": filepath.Join(t.TempDir(), "claude", "skills")})

	items := listSkillsTargets(t, s)
	if !slices.Equal(items["universal"].SkillsSharedWith, []string{"codex"}) || !slices.Equal(items["codex"].SkillsSharedWith, []string{"universal"}) {
		t.Errorf("universal = %+v, codex = %+v", items["universal"], items["codex"])
	}
	if len(items["claude"].SkillsSharedWith) != 0 {
		t.Errorf("claude = %+v, want no sharing", items["claude"])
	}
}

func TestUpdateTarget_SkillsOffDetachesAndPersists(t *testing.T) {
	s, _, _, gemini := newSkillsOffServer(t)

	rr := serve(t, s, http.MethodPatch, "/api/targets/gemini", `{"skills_enabled":false}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Detach  struct {
			Removed []string `json:"removed"`
			Kept    []string `json:"kept"`
		} `json:"detach"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if !resp.Success || !slices.Equal(resp.Detach.Removed, []string{"alpha"}) || !slices.Equal(resp.Detach.Kept, []string{"mine"}) {
		t.Errorf("patch = %s", rr.Body.String())
	}
	if _, err := os.Lstat(filepath.Join(gemini, "alpha")); err == nil {
		t.Error("alpha link should be removed")
	}
	data, _ := os.ReadFile(config.ConfigPath())
	if !strings.Contains(string(data), "enabled: false") {
		t.Errorf("config not persisted:\n%s", data)
	}

	items := listSkillsTargets(t, s)
	if items["gemini"].SkillsEnabled || !slices.Equal(items["gemini"].SkillsReadFrom, []string{"universal"}) {
		t.Errorf("gemini = %+v", items["gemini"])
	}
	if !items["universal"].SkillsEnabled || !slices.Equal(items["universal"].SkillsAlsoReadBy, []string{"gemini"}) {
		t.Errorf("universal = %+v", items["universal"])
	}
}

func TestUpdateTarget_SkillsOnOnlyPersists(t *testing.T) {
	s, _, _, gemini := newSkillsOffServer(t)
	serve(t, s, http.MethodPatch, "/api/targets/gemini", `{"skills_enabled":false}`)

	rr := serve(t, s, http.MethodPatch, "/api/targets/gemini", `{"skills_enabled":true}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "detach") {
		t.Errorf("turning on must not detach: %s", rr.Body.String())
	}
	if !listSkillsTargets(t, s)["gemini"].SkillsEnabled {
		t.Error("gemini should be enabled again")
	}
	if _, err := os.Lstat(filepath.Join(gemini, "alpha")); err == nil {
		t.Error("turning on only changes config")
	}
}

func TestSyncHandler_SkipsSkillsOffTarget(t *testing.T) {
	s, _, _, gemini := newSkillsOffServer(t)
	serve(t, s, http.MethodPatch, "/api/targets/gemini", `{"skills_enabled":false}`)

	rr := serve(t, s, http.MethodPost, "/api/sync", `{}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("sync: %d %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Lstat(filepath.Join(gemini, "alpha")); err == nil {
		t.Error("web sync must not relink a target with skills off")
	}
}

func TestAddTarget_SkillsDisabled(t *testing.T) {
	s, _ := newTestServer(t)
	path := filepath.Join(t.TempDir(), "gemini", "skills")

	rr := serve(t, s, http.MethodPost, "/api/targets", `{"name":"gemini","path":"`+path+`","skills_enabled":false}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("add: %d %s", rr.Code, rr.Body.String())
	}
	if listSkillsTargets(t, s)["gemini"].SkillsEnabled {
		t.Error("gemini should be added with skills off")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("no skills folder should be created")
	}
}

func TestAvailableTargets_ReadsFrom(t *testing.T) {
	s, _, _, _ := newSkillsOffServer(t)

	rr := serve(t, s, http.MethodGet, "/api/config/available-targets", "")
	var resp struct {
		Targets []struct {
			Name      string   `json:"name"`
			ReadsFrom []string `json:"readsFrom"`
		} `json:"targets"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	got := map[string][]string{}
	for _, item := range resp.Targets {
		got[item.Name] = item.ReadsFrom
	}
	// gemini is configured and enabled; it still reads universal's folder.
	if !slices.Equal(got["gemini"], []string{"universal"}) {
		t.Errorf("gemini readsFrom = %v", got["gemini"])
	}
	if slices.Contains(got["universal"], "universal") {
		t.Errorf("a target never reads from itself: %v", got["universal"])
	}
}

// A shared folder lists the tools that read it, so the add dialog can show whom
// universal would serve before it is configured.
func TestAvailableTargets_ReadBy(t *testing.T) {
	s, _, _, _ := newSkillsOffServer(t)

	rr := serve(t, s, http.MethodGet, "/api/config/available-targets", "")
	var resp struct {
		Targets []struct {
			Name   string   `json:"name"`
			ReadBy []string `json:"readBy"`
		} `json:"targets"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	for _, item := range resp.Targets {
		if item.Name == "universal" && !slices.Contains(item.ReadBy, "gemini") {
			t.Errorf("universal readBy = %v, want gemini (configured, scans ~/.agents/skills)", item.ReadBy)
		}
	}
}

func TestRemoveTarget_SkillsOffLeavesFolder(t *testing.T) {
	s, sourceDir, _, gemini := newSkillsOffServer(t)
	serve(t, s, http.MethodPatch, "/api/targets/gemini", `{"skills_enabled":false}`)
	// A link added by hand after skills were switched off is the user's business.
	if err := os.Symlink(filepath.Join(sourceDir, "alpha"), filepath.Join(gemini, "alpha")); err != nil {
		t.Fatal(err)
	}

	rr := serve(t, s, http.MethodDelete, "/api/targets/gemini", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("remove: %d %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Lstat(filepath.Join(gemini, "alpha")); err != nil {
		t.Error("removing a target with skills off must not touch its folder")
	}
}

func TestProjectMode_SkillsOffRoundTrip(t *testing.T) {
	s, projectRoot := newTestProjectServerWithExtras(t, nil)
	rr := serve(t, s, http.MethodPost, "/api/targets", `{"name":"claude"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("add: %d %s", rr.Code, rr.Body.String())
	}
	rr = serve(t, s, http.MethodPatch, "/api/targets/claude", `{"skills_enabled":false}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rr.Code, rr.Body.String())
	}
	loaded, err := config.LoadProject(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Targets[0].SkillsConfig().IsEnabled() {
		t.Error("project config should record skills off")
	}
	if listSkillsTargets(t, s)["claude"].SkillsEnabled {
		t.Error("GET /api/targets should report skills off")
	}
}
