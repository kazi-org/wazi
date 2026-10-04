package observatory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSafeReadRejectsTraversalAndExternalSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ok.go"), []byte("package ok\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.go"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.go"), filepath.Join(root, "link.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := safeRead(root, "../secret.go", 1024); err == nil {
		t.Fatal("accepted traversal")
	}
	if _, err := safeRead(root, "link.go", 1024); err == nil {
		t.Fatal("accepted external symlink")
	}
	got, err := safeRead(root, "ok.go", 1024)
	if err != nil || !strings.Contains(string(got), "package ok") {
		t.Fatalf("safe read failed: %q, %v", got, err)
	}
	if _, err := safeRead(root, "ok.go", 4); err == nil {
		t.Fatal("read exceeded its byte limit")
	}
}

func TestIdentityLocatorUsesPrivateRegistryKeyAndGitCommonDir(t *testing.T) {
	base := t.TempDir()
	first := filepath.Join(base, "first")
	clone := filepath.Join(base, "clone")
	for _, root := range []string{first, clone} {
		if err := os.MkdirAll(filepath.Join(root, ".git"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	loc1, err := identityLocator(first)
	if err != nil {
		t.Fatal(err)
	}
	loc2, err := identityLocator(clone)
	if err != nil {
		t.Fatal(err)
	}
	if loc1 == loc2 {
		t.Fatal("independent clone roots shared a private registry locator")
	}
	worktrees := filepath.Join(base, "worktrees")
	common := filepath.Join(base, "common.git")
	gitdir := filepath.Join(worktrees, "one.git")
	for _, dir := range []string{worktrees, common, gitdir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(gitdir, "commondir"), []byte("../../first/.git\n"), 0600); err != nil {
		t.Fatal(err)
	}
	worktree := filepath.Join(base, "worktree")
	if err := os.Mkdir(worktree, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: ../worktrees/one.git\n"), 0600); err != nil {
		t.Fatal(err)
	}
	wid, err := identityLocator(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if wid != loc1 {
		t.Fatalf("worktree locator = %q, want common-dir locator %q", wid, loc1)
	}
}

func TestLockFileReleasesOnCloseWithoutDeletingLockPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	release, err := lockFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lockFile(path); err == nil {
		t.Fatal("concurrent lock was granted")
	}
	release()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock file was removed: %v", err)
	}
	releaseAgain, err := lockFile(path)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	releaseAgain()
}

func TestLineDerivedTasksAreNotStableBindingSubjects(t *testing.T) {
	if stableTaskID(Task{SourceID: "line-9"}) {
		t.Fatal("accepted a line-derived task ID")
	}
	if !stableTaskID(Task{SourceID: "T1.1"}) {
		t.Fatal("rejected an authored plan-local task ID")
	}
}

func TestBoundedBufferCapsRetainedProcessOutput(t *testing.T) {
	b := boundedBuffer{limit: 4}
	if n, err := b.Write([]byte("abcdef")); err != nil || n != 6 {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if b.String() != "abcd" || !b.exceeded {
		t.Fatalf("bounded output = %q, exceeded=%v", b.String(), b.exceeded)
	}
}

func TestTaskIdentityAndBindingsRefreshAfterPlanEdit(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(repo, "docs", "plan.md")
	plan := "# Project plan\n\n- [ ] T1.1 Retain record\n"
	if err := os.WriteFile(planPath, []byte(plan), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "retain.go"), []byte("package retain\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bridge, err := filepath.Abs(filepath.Join("..", "..", "scripts", "host-bridge.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is unavailable")
	}
	dataDir := filepath.Join(t.TempDir(), "app-data")
	svc := New(root, dataDir, "node", bridge)
	scan, err := svc.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Projects) != 1 {
		t.Fatalf("discovered %d projects", len(scan.Projects))
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "wazi-repository-id")); !os.IsNotExist(err) {
		t.Fatal("discovery wrote repository identity into the source checkout")
	}
	projectID := scan.Projects[0].ID
	if len(scan.Projects[0].Plans) == 0 || len(scan.Projects[0].Plans[0].Tasks) != 1 {
		t.Fatal("initial task was not parsed")
	}
	task := scan.Projects[0].Plans[0].Tasks[0]
	if task.ID != "T1.1" {
		t.Fatalf("internal task ID = %q, want source ID T1.1", task.ID)
	}
	snapshot, err := svc.Snapshot(context.Background(), projectID)
	if err != nil {
		t.Fatal(err)
	}
	var candidate *Suggestion
	for i := range snapshot.Suggestions {
		if snapshot.Suggestions[i].TaskID == task.ID {
			candidate = &snapshot.Suggestions[i]
			break
		}
	}
	if candidate == nil {
		t.Fatal("expected local filename suggestion")
	}
	if _, err := svc.WriteBinding(projectID, candidate.PlanPath, task.ID, candidate.Target, candidate.Kind, candidate.BasisDigest, snapshot.SidecarDigest, "confirm"); err != nil {
		t.Fatalf("confirm initial task: %v", err)
	}
	duplicate := "# Project plan\n\n- [ ] T1.1 Retain record\n- [ ] T1.1 Retain record\n"
	if err := os.WriteFile(planPath, []byte(duplicate), 0600); err != nil {
		t.Fatal(err)
	}
	duplicated, err := svc.Snapshot(context.Background(), projectID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.WriteBinding(projectID, candidate.PlanPath, task.ID, candidate.Target, candidate.Kind, candidate.BasisDigest, duplicated.SidecarDigest, "confirm"); err == nil {
		t.Fatal("confirmed an ambiguous duplicate task ID")
	}
	if err := os.WriteFile(planPath, []byte("# Project plan\n\n- [ ] T1.2 Renamed record\n"), 0600); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.Snapshot(context.Background(), projectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Suggestions) != 0 {
		t.Fatalf("removed task still has suggestions: %#v", updated.Suggestions)
	}
	if _, err := svc.ResolveTask(updated, candidate.PlanPath, task.ID); err == nil {
		t.Fatal("resolved a task removed from the source plan")
	}
	if _, err := svc.WriteBinding(projectID, candidate.PlanPath, task.ID, candidate.Target, candidate.Kind, candidate.BasisDigest, updated.SidecarDigest, "confirm"); err == nil {
		t.Fatal("confirmed a task removed from the source plan")
	}
	if len(updated.Bindings) != 1 || updated.Bindings[0].Freshness != "stale" {
		t.Fatalf("old authored binding was not retained as stale: %#v", updated.Bindings)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "repositories.json"), []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Discover(context.Background()); err == nil {
		t.Fatal("corrupt private identity registry was silently replaced")
	}
	if b, err := os.ReadFile(filepath.Join(dataDir, "repositories.json")); err != nil || string(b) != "[]" {
		t.Fatalf("corrupt registry changed: %q %v", b, err)
	}
}

func TestRevokeBindingPreservesOtherLinksAndSource(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is unavailable")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "project")
	for _, dir := range []string{filepath.Join(repo, "docs"), filepath.Join(repo, "src"), filepath.Join(repo, ".git")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	planPath := filepath.Join(repo, "docs", "plan.md")
	plan := []byte("# Project plan\n\n- [ ] T1.1 Retain record\n")
	code := []byte("package retain\n")
	if err := os.WriteFile(planPath, plan, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "retain.go"), code, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "other.go"), []byte("package other\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bridge, err := filepath.Abs(filepath.Join("..", "..", "scripts", "host-bridge.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(t.TempDir(), "private")
	svc := New(root, dataDir, "node", bridge)
	scan, err := svc.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Projects) != 1 {
		t.Fatalf("projects=%d", len(scan.Projects))
	}
	projectID := scan.Projects[0].ID
	snap, err := svc.Snapshot(context.Background(), projectID)
	if err != nil {
		t.Fatal(err)
	}
	var suggestion *Suggestion
	for i := range snap.Suggestions {
		if snap.Suggestions[i].Target == "src/retain.go" {
			suggestion = &snap.Suggestions[i]
			break
		}
	}
	if suggestion == nil {
		t.Fatal("no candidate for primary file")
	}
	snap, err = svc.WriteBinding(projectID, suggestion.PlanPath, suggestion.TaskID, suggestion.Target, suggestion.Kind, suggestion.BasisDigest, snap.SidecarDigest, "confirm")
	if err != nil {
		t.Fatal(err)
	}
	snap, err = svc.WriteBinding(projectID, suggestion.PlanPath, suggestion.TaskID, "src/other.go", "code", snap.SnapshotDigest, snap.SidecarDigest, "manual")
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Bindings) != 2 {
		t.Fatalf("before revoke bindings=%d", len(snap.Bindings))
	}
	snap, err = svc.WriteBinding(projectID, suggestion.PlanPath, suggestion.TaskID, suggestion.Target, suggestion.Kind, snap.SnapshotDigest, snap.SidecarDigest, "revoke")
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Bindings) != 1 || snap.Bindings[0].Target != "src/other.go" {
		t.Fatalf("revoke did not preserve other binding: %#v", snap.Bindings)
	}
	if got, err := os.ReadFile(planPath); err != nil || string(got) != string(plan) {
		t.Fatalf("plan changed: %q %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(repo, "src", "retain.go")); err != nil || string(got) != string(code) {
		t.Fatalf("source changed: %q %v", got, err)
	}
	if _, err = svc.WriteBinding(projectID, suggestion.PlanPath, suggestion.TaskID, suggestion.Target, suggestion.Kind, snap.SnapshotDigest, snap.SidecarDigest, "revoke"); err == nil {
		t.Fatal("revoked a binding that was already absent")
	}
}

func TestDiscoveryCancellationReapsParserProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process cancellation assertion requires Unix signals")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "pid")
	node := filepath.Join(dir, "node")
	script := "#!/bin/sh\nprintf '%s' $$ > '" + marker + "'\nexec /bin/sleep 60\n"
	if err := os.WriteFile(node, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	bridge := filepath.Join(dir, "bridge.mjs")
	if err := os.WriteFile(bridge, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	svc := New(dir, filepath.Join(dir, "data"), node, bridge)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := svc.Discover(ctx); done <- err }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("parser process did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	pidBytes, err := os.ReadFile(marker)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled discovery succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancelled discovery did not return")
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("parser process %d remained alive after discovery cancellation", pid)
}

func TestSnapshotRejectsPlanEditedAfterParserRead(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(repo, "docs", "plan.md")
	parsedBytes := []byte("# Race plan\n\n- [ ] T1.1 Parsed task\n")
	changedBytes := []byte("# Race plan\n\n- [ ] T1.2 Changed task\n")
	if err := os.WriteFile(planPath, parsedBytes, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is unavailable")
	}
	original, err := filepath.Abs(filepath.Join("..", "..", "scripts", "host-bridge.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(root, "race-bridge.mjs")
	cache := filepath.Join(root, "bridge-cache.json")
	script := fmt.Sprintf(`import {spawnSync} from 'node:child_process'; import fs from 'node:fs'; const cache=%q; if(fs.existsSync(cache)){process.stdout.write(fs.readFileSync(cache));}else{const r=spawnSync('node',[%q,process.argv[2]],{encoding:'utf8'});if(r.status!==0)process.exit(r.status||1);fs.writeFileSync(cache,r.stdout);process.stdout.write(r.stdout);fs.writeFileSync(%q,%q);}`, cache, original, planPath, string(changedBytes))
	if err := os.WriteFile(wrapper, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	svc := New(root, filepath.Join(t.TempDir(), "app-data"), "node", wrapper)
	scan, err := svc.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Projects) != 1 {
		t.Fatalf("discovered projects: %d", len(scan.Projects))
	}
	if _, err := svc.Snapshot(context.Background(), scan.Projects[0].ID); err == nil || !strings.Contains(err.Error(), "changed after its task graph was parsed") {
		t.Fatalf("snapshot accepted a mixed parser/source view: %v", err)
	}
}

func TestSelectedScanExcludesCredentialLikePaths(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"main.go": "package p", "module.mjs": "export {}", "widget.spec.cjs": "module.exports = {}", "cache.testenv": "sensitive", "test.env": "TOKEN=secret", "backup.test.txt": "backup", "picture.test.png": "binary", ".hidden.test.go": "hidden", ".env": "TOKEN=secret", "id_rsa": "private", "private-key.pem": "private", "spec.test.ts": "test"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "huge.test.mjs"), make([]byte, (128<<10)+1), 0600); err != nil {
		t.Fatal(err)
	}
	s := New(root, t.TempDir(), "node", "bridge")
	files, _ := s.scanFiles(root)
	got := map[string]bool{}
	for _, f := range files {
		got[f.Path] = true
	}
	if !got["main.go"] || !got["spec.test.ts"] || !got["module.mjs"] || !got["widget.spec.cjs"] {
		t.Fatalf("expected code/test files in snapshot: %#v", got)
	}
	for _, name := range []string{".env", "test.env", "id_rsa", "private-key.pem", "cache.testenv", "backup.test.txt", "picture.test.png", ".hidden.test.go", "huge.test.mjs"} {
		if got[name] {
			t.Errorf("sensitive file included: %s", name)
		}
	}
}

func TestSidecarReadRejectsSymlinkParentsAndOversizedFiles(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(outside, ".wazi"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, ".wazi", "links.json"), []byte(`{"version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, ".wazi"), filepath.Join(root, ".wazi")); err != nil {
		t.Fatal(err)
	}
	s := New(root, t.TempDir(), "node", "bridge")
	if _, _, err := s.loadSidecar(root, "repo-0123456789abcdef01234567"); err == nil {
		t.Fatal("followed symlinked sidecar directory")
	}
	if err := os.Remove(filepath.Join(root, ".wazi")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".wazi"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".wazi", "links.json"), make([]byte, (1<<20)+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.loadSidecar(root, "repo-0123456789abcdef01234567"); err == nil {
		t.Fatal("read oversized sidecar")
	}
}

func TestHostRequiresLoopbackAndSessionCapabilityForWrites(t *testing.T) {
	h, err := NewHost(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	bad := httptest.NewRecorder()
	h.Handler().ServeHTTP(bad, httptest.NewRequest(http.MethodGet, "http://example.test/api/session", nil))
	bad.Result().Body.Close()
	if bad.Code != http.StatusForbidden {
		t.Fatalf("remote host status %d", bad.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8891/api/anything", strings.NewReader(`{}`))
	req.Host = "127.0.0.1:8891"
	denied := httptest.NewRecorder()
	h.Handler().ServeHTTP(denied, req)
	denied.Result().Body.Close()
	if denied.Code != http.StatusForbidden {
		t.Fatalf("write without capability status %d", denied.Code)
	}
}
