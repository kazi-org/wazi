package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kazi-org/wazi/internal/deep"
	"github.com/kazi-org/wazi/internal/observatory"
)

type countEngine struct{ calls atomic.Int32 }

func (e *countEngine) Complete(_ context.Context, m deep.Manifest) (string, error) {
	e.calls.Add(1)
	return "checked " + m.TaskRef, nil
}
func setup(t *testing.T) (*Routes, observatory.Snapshot, *countEngine) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "sample")
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "plan.md"), []byte("# Sample\n\n- [ ] T1.1 Inspect alpha kind: agent stage: implement acc: [alpha.go reviewed]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "alpha.go"), []byte("package alpha\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bridge, err := filepath.Abs("../../scripts/host-bridge.mjs")
	if err != nil {
		t.Fatal(err)
	}
	service := observatory.New(root, t.TempDir(), "node", bridge)
	scan, err := service.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Projects) != 1 {
		t.Fatalf("projects: %+v", scan)
	}
	snapshot, err := service.Snapshot(context.Background(), scan.Projects[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	engine := &countEngine{}
	storage, err := deep.New(deep.Config{Dir: t.TempDir(), MaxBytes: 8 << 20, Engine: engine})
	if err != nil {
		t.Fatal(err)
	}
	return &Routes{Observatory: service, Deep: storage}, snapshot, engine
}
func send(t *testing.T, app *Routes, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	if !app.Handle(w, r) {
		t.Fatal("route not handled")
	}
	return w
}
func TestSelectionAndUnavailableContextNeverDispatchModel(t *testing.T) {
	app, snapshot, engine := setup(t)
	contextResponse := send(t, app, "/api/context", selection{ProjectID: snapshot.ProjectID})
	if contextResponse.Code != 200 || !strings.Contains(contextResponse.Body.String(), "unavailable") {
		t.Fatalf("context: %d %s", contextResponse.Code, contextResponse.Body.String())
	}
	request := selection{ProjectID: snapshot.ProjectID, PlanPath: "docs/plan.md", TaskID: "T1.1", SnapshotDigest: snapshot.SnapshotDigest, Question: "Next check?", ContextMode: "with-context"}
	response := send(t, app, "/api/deep/answer", request)
	if response.Code != 409 {
		t.Fatalf("expected unavailable context rejection, got %d: %s", response.Code, response.Body.String())
	}
	if engine.calls.Load() != 0 {
		t.Fatal("context selection dispatched a model")
	}
	request.ContextMode = "plan-code-only"
	response = send(t, app, "/api/deep/answer", request)
	if response.Code != 200 {
		t.Fatalf("explicit request: %d %s", response.Code, response.Body.String())
	}
	response = send(t, app, "/api/deep/answer", request)
	if response.Code != 200 || engine.calls.Load() != 1 {
		t.Fatalf("identical request did not reuse: %d %s calls %d", response.Code, response.Body.String(), engine.calls.Load())
	}
	request.SnapshotDigest = "stale"
	response = send(t, app, "/api/deep/answer", request)
	if response.Code != 409 || engine.calls.Load() != 1 {
		t.Fatal("stale source reached model")
	}
}
func TestScopeAndMalformedRequestsRejectBeforeModel(t *testing.T) {
	app, snapshot, engine := setup(t)
	for _, path := range []string{"/api/deep/answer", "/api/deep/inspect", "/api/deep/delete"} {
		response := send(t, app, path, selection{ProjectID: "unknown", Key: strings.Repeat("a", 64)})
		if response.Code != 409 {
			t.Fatalf("%s scope status %d", path, response.Code)
		}
	}
	if engine.calls.Load() != 0 {
		t.Fatal("unknown scope dispatched")
	}
	r := httptest.NewRequest(http.MethodPost, "/api/deep/answer", strings.NewReader(`{"projectId":"`+snapshot.ProjectID+`","code":"untrusted body"}`))
	w := httptest.NewRecorder()
	app.Handle(w, r)
	if w.Code != 400 {
		t.Fatalf("browser body accepted: %d", w.Code)
	}
}
func TestLineageAdapterFailsClosedWithoutOwner(t *testing.T) {
	validator := LineageAdapter{}
	got, err := validator.ValidateLineage(context.Background(), deep.ContextScope{}, nil)
	if got.Available || !errorsIsMemoryUnavailable(err) {
		t.Fatal("unqualified owner passed lineage")
	}
}
func errorsIsMemoryUnavailable(err error) bool { return err == deep.ErrMemoryUnavailable }
