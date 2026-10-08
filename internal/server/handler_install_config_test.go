package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"skillshare/internal/install"
)

func TestHandleMissingConfigEntries_ListsAbsentEntries(t *testing.T) {
	s, src := newTestServer(t)
	store := install.LoadMetadataOrNew(src)
	store.Set("_team-skills", &install.MetadataEntry{Source: "https://github.com/team/skills.git", Tracked: true, Branch: "main"})
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	rr := httptest.NewRecorder()
	s.handleMissingConfigEntries(rr, httptest.NewRequest(http.MethodGet, "/api/install/missing", nil))
	var resp struct {
		Entries []configEntryInfo `json:"entries"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	want := configEntryInfo{Name: "_team-skills", Source: "https://github.com/team/skills.git", Tracked: true, Branch: "main"}
	if len(resp.Entries) != 1 || resp.Entries[0] != want {
		t.Fatalf("entries = %+v, want [%+v]", resp.Entries, want)
	}
}

func TestHandleInstallFromConfig_NothingMissing(t *testing.T) {
	s, _ := newTestServer(t)
	rr := httptest.NewRecorder()
	s.handleInstallFromConfig(rr, httptest.NewRequest(http.MethodPost, "/api/install/from-config", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}
