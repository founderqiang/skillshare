package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skillshare/internal/config"
)

// newDocsExtraServer serves a docs extra holding index.md and draft.md with
// one merge target, and returns the server, the extra's folder and the target.
func newDocsExtraServer(t *testing.T, target config.ExtraTargetConfig) (*Server, string, string) {
	t.Helper()
	if target.Path == "" {
		target.Path = t.TempDir()
	}
	s, skillsDir := newTestServerWithExtras(t, []config.ExtraConfig{{Name: "docs", Targets: []config.ExtraTargetConfig{target}}}, "")
	docs := filepath.Join(filepath.Dir(skillsDir), "extras", "docs")
	os.WriteFile(filepath.Join(docs, "index.md"), []byte("i"), 0644)
	os.WriteFile(filepath.Join(docs, "draft.md"), []byte("d"), 0644)
	return s, docs, target.Path
}

func serveExtras(s *Server, method, path, body string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.handler.ServeHTTP(rr, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rr
}

func TestHandleExtras_ListsTargetFilterAndFileCount(t *testing.T) {
	s, _, _ := newDocsExtraServer(t, config.ExtraTargetConfig{Exclude: []string{"draft*"}})

	rr := serveExtras(s, http.MethodGet, "/api/extras", "")

	var resp struct {
		Extras []extrasListEntry `json:"extras"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if ti := resp.Extras[0].Targets[0]; ti.FileCount != 1 || len(ti.Exclude) != 1 {
		t.Errorf("target = %+v, want file_count 1 and the exclude pattern", ti)
	}
}

func TestHandleExtrasPreview_NamesTheExcludePattern(t *testing.T) {
	s, _, _ := newDocsExtraServer(t, config.ExtraTargetConfig{})

	rr := serveExtras(s, http.MethodPost, "/api/extras/docs/preview", `{"exclude":["draft*"],"include":["*.md","nope.md"]}`)

	body := rr.Body.String()
	if !strings.Contains(body, `{"file":"draft.md","status":"excluded","reason":"draft*"}`) || !strings.Contains(body, `"unmatched":["nope.md"]`) {
		t.Errorf("unexpected preview: %s", body)
	}
}

func TestHandleExtrasEditTarget_NewPathRemovesOldLinksAndSyncs(t *testing.T) {
	s, _, oldTgt := newDocsExtraServer(t, config.ExtraTargetConfig{})
	serveExtras(s, http.MethodPost, "/api/extras/sync", `{"name":"docs"}`)
	newTgt := t.TempDir()

	rr := serveExtras(s, http.MethodPut, "/api/extras/docs/targets", `{"path":"`+oldTgt+`","target":{"path":"`+newTgt+`"}}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Lstat(filepath.Join(oldTgt, "index.md")); !os.IsNotExist(err) {
		t.Errorf("old link should be gone, lstat err = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(newTgt, "index.md")); err != nil {
		t.Errorf("new target should be synced: %v", err)
	}
}

func TestHandleExtrasEditTarget_SymlinkDropsFilters(t *testing.T) {
	s, _, tgt := newDocsExtraServer(t, config.ExtraTargetConfig{Include: []string{"index.md"}})

	rr := serveExtras(s, http.MethodPut, "/api/extras/docs/targets", `{"path":"`+tgt+`","target":{"path":"`+tgt+`","mode":"symlink","include":["index.md"]}}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := s.cfg.Extras[0].Targets[0]; got.Mode != "symlink" || got.Include != nil {
		t.Errorf("target = %+v, want symlink with no include", got)
	}
}

func TestHandleExtrasEdit_RenameKeepsTheFolder(t *testing.T) {
	s, docs, _ := newDocsExtraServer(t, config.ExtraTargetConfig{})

	rr := serveExtras(s, http.MethodPatch, "/api/extras/docs", `{"name":"notes"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := s.cfg.Extras[0]; got.Name != "notes" || got.Source != docs {
		t.Errorf("extra = %+v, want notes with source %s", got, docs)
	}
}

func TestHandleExtrasEdit_RenameFromAgentsRejected(t *testing.T) {
	s, _ := newTestServerWithExtras(t, []config.ExtraConfig{{Name: "agents", Targets: []config.ExtraTargetConfig{{Path: t.TempDir()}}}}, "")

	rr := serveExtras(s, http.MethodPatch, "/api/extras/agents", `{"name":"helpers"}`)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleExtrasEdit_MissingSourceRejected(t *testing.T) {
	s, _, _ := newDocsExtraServer(t, config.ExtraTargetConfig{})

	rr := serveExtras(s, http.MethodPatch, "/api/extras/docs", `{"source":"`+filepath.Join(t.TempDir(), "nope")+`"}`)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleExtrasEdit_NewSourceRelinksTargets(t *testing.T) {
	s, _, tgt := newDocsExtraServer(t, config.ExtraTargetConfig{})
	serveExtras(s, http.MethodPost, "/api/extras/sync", `{"name":"docs"}`)
	newSrc := t.TempDir()
	os.WriteFile(filepath.Join(newSrc, "guide.md"), []byte("g"), 0644)

	rr := serveExtras(s, http.MethodPatch, "/api/extras/docs", `{"source":"`+newSrc+`"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Lstat(filepath.Join(tgt, "index.md")); !os.IsNotExist(err) {
		t.Errorf("link into the old folder should be gone, lstat err = %v", err)
	}
	if dest, err := os.Readlink(filepath.Join(tgt, "guide.md")); err != nil || !strings.HasPrefix(dest, newSrc) {
		t.Errorf("guide.md should link into the new folder, got %q, %v", dest, err)
	}
}

func TestHandleExtrasMode_SymlinkRejectedWhenTargetHasFilters(t *testing.T) {
	s, _, tgt := newDocsExtraServer(t, config.ExtraTargetConfig{Include: []string{"index.md"}})

	rr := serveExtras(s, http.MethodPatch, "/api/extras/docs/mode", `{"target":"`+tgt+`","mode":"symlink"}`)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleExtrasEdit_NewSourceRelinksSymlinkWhoseOldFolderIsGone(t *testing.T) {
	link := filepath.Join(t.TempDir(), "docs")
	s, docs, _ := newDocsExtraServer(t, config.ExtraTargetConfig{Path: link, Mode: "symlink"})
	if err := os.Symlink(docs, link); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(docs)
	newSrc := t.TempDir()

	rr := serveExtras(s, http.MethodPatch, "/api/extras/docs", `{"source":"`+newSrc+`"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if dest, err := os.Readlink(link); err != nil || dest != newSrc {
		t.Errorf("target should link to the new folder, got %q, %v", dest, err)
	}
}
