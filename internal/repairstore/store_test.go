package repairstore

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func fixture(t *testing.T) (string, string, string, Source, []byte) {
	t.Helper()
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(repo, "plan.md")
	original := []byte("# plan\n")
	if err := os.WriteFile(p, original, 0640); err != nil {
		t.Fatal(err)
	}
	neighbor := filepath.Join(repo, "neighbor.md")
	if err := os.WriteFile(neighbor, []byte("untouched\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := ReadSource(p)
	if err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(base, "private", "store")
	candidate := []byte("# repaired\n")
	return base, repo, store, s, candidate
}

func saved(t *testing.T) (string, string, string, Source, []byte, Manifest) {
	t.Helper()
	base, repo, store, s, c := fixture(t)
	m, err := Save(store, s, c, "test-profile")
	if err != nil {
		t.Fatal(err)
	}
	return base, repo, store, s, c, m
}

func TestReadSourceRejectsSymlinkComponentsAndInvalidBytes(t *testing.T) {
	_, repo, _, s, _, _ := saved(t)
	link := filepath.Join(filepath.Dir(repo), "linked")
	if err := os.Symlink(repo, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSource(filepath.Join(link, "plan.md")); err == nil {
		t.Fatal("accepted parent symlink")
	}
	if err := os.WriteFile(s.Path, []byte("bad\x00"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSource(s.Path); err == nil {
		t.Fatal("accepted NUL")
	}
}

func TestSaveRequiresPrivateOutsideStoreAndIsIdempotent(t *testing.T) {
	_, repo, store, s, c := fixture(t)
	if _, err := Save(filepath.Join(repo, "private"), s, c, "p"); err == nil {
		t.Fatal("accepted in-repo store")
	}
	if _, err := os.Stat(filepath.Join(repo, "private")); !os.IsNotExist(err) {
		t.Fatal("refused in-repo save created directories")
	}
	if err := os.MkdirAll(store, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(store, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(store, s, c, "p"); err == nil {
		t.Fatal("accepted unsafe store permissions")
	}
	if err := os.Chmod(store, 0700); err != nil {
		t.Fatal(err)
	}
	m, err := Save(store, s, c, "p")
	if err != nil {
		t.Fatal(err)
	}
	m2, err := Save(store, s, c, "p")
	if err != nil || m != m2 {
		t.Fatalf("repeat save: %v, equal=%v", err, m == m2)
	}
}

func TestCanonicalStoreCreationSyncsEachParentAndStopsOnFailure(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "one", "two", "store")
	var parents []string
	wantFailure := errors.New("injected directory sync failure")
	_, err := canonicalPathCreateWithSync(path, func(parent string) error {
		parents = append(parents, parent)
		if len(parents) == 2 {
			return wantFailure
		}
		return nil
	})
	if !errors.Is(err, wantFailure) {
		t.Fatalf("want injected sync error, got %v", err)
	}
	want := []string{base, filepath.Join(base, "one")}
	if len(parents) != len(want) {
		t.Fatalf("synced parents=%v", parents)
	}
	for i := range want {
		if parents[i] != want[i] {
			t.Fatalf("synced parents=%v", parents)
		}
	}
	if _, err := os.Stat(filepath.Join(base, "one", "two")); err != nil {
		t.Fatalf("expected second directory to exist before its parent-sync failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "one", "two", "store")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("created child after sync failure: %v", err)
	}
}

func TestSaveStopsBeforeCandidateFilesWhenEntryParentSyncFails(t *testing.T) {
	_, _, store, source, candidate := fixture(t)
	root, err := filepath.Abs(store)
	if err != nil {
		t.Fatal(err)
	}
	wantFailure := errors.New("injected parent sync failure")
	_, err = saveWithSync(store, source, candidate, "test-profile", func(parent string) error {
		if parent == root {
			return wantFailure
		}
		return syncDir(parent)
	})
	if !errors.Is(err, wantFailure) {
		t.Fatalf("want injected sync error, got %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("store root should have been created before entry sync: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected only candidate directory, entries=%v err=%v", entries, err)
	}
	files, err := os.ReadDir(filepath.Join(root, entries[0].Name()))
	if err != nil || len(files) != 0 {
		t.Fatalf("candidate files appeared before entry parent sync: files=%v err=%v", files, err)
	}
}

func TestApplyExactBackupPermissionsAndNeighbor(t *testing.T) {
	_, repo, store, s, c, m := saved(t)
	backup, err := Apply(store, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(c) {
		t.Fatalf("source=%q", got)
	}
	old, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != string(s.Bytes) {
		t.Fatalf("backup=%q", old)
	}
	info, err := os.Stat(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
	neighbor, err := os.ReadFile(filepath.Join(repo, "neighbor.md"))
	if err != nil || string(neighbor) != "untouched\n" {
		t.Fatalf("neighbor=%q err=%v", neighbor, err)
	}
}

func TestApplyReturnsBackupWhenPostRenameDirectorySyncFails(t *testing.T) {
	_, _, store, source, candidate, manifest := saved(t)
	wantFailure := errors.New("injected parent directory sync failure")
	syncCalls := 0
	backup, err := applyWithSync(store, manifest.ID, func(path string) error {
		syncCalls++
		if syncCalls == 1 {
			return syncDir(path)
		} // backup entry is durable first
		if syncCalls == 2 {
			return wantFailure
		} // source rename has completed
		return syncDir(path)
	})
	if !errors.Is(err, ErrApplyUncertain) || !errors.Is(err, wantFailure) {
		t.Fatalf("want uncertain apply wrapping injected failure, got backup=%q err=%v", backup, err)
	}
	if backup == "" {
		t.Fatal("uncertain apply omitted the known backup path")
	}
	gotBackup, readErr := os.ReadFile(backup)
	if readErr != nil || string(gotBackup) != string(source.Bytes) {
		t.Fatalf("backup=%q err=%v", gotBackup, readErr)
	}
	gotSource, readErr := os.ReadFile(source.Path)
	if readErr != nil || string(gotSource) != string(candidate) {
		t.Fatalf("source=%q err=%v", gotSource, readErr)
	}
}

func TestApplyStaleAndCorruptFailClosed(t *testing.T) {
	_, _, store, s, _, m := saved(t)
	if err := os.WriteFile(s.Path, []byte("externally changed\n"), 0640); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(store, m.ID); !errors.Is(err, ErrStale) {
		t.Fatalf("want stale, got %v", err)
	}
	_, _, store2, _, _, m2 := saved(t)
	if err := os.WriteFile(filepath.Join(store2, m2.ID, "candidate.md"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(store2, m2.ID); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("want corrupt, got %v", err)
	}
}

func TestConcurrentApplyOnlyOneWins(t *testing.T) {
	_, _, store, _, _, m := saved(t)
	start := make(chan struct{})
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; _, errs[i] = Apply(store, m.ID) }(i)
	}
	close(start)
	wg.Wait()
	success, stale := 0, 0
	for _, err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrStale) {
			stale++
		} else {
			t.Fatalf("unexpected apply error: %v", err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("success=%d stale=%d errors=%v", success, stale, errs)
	}
}

func TestConcurrentDifferentCandidatesForSameSourceOnlyOneWins(t *testing.T) {
	_, _, store, source, candidateA := fixture(t)
	candidateB := []byte("# another proposal\n")
	first, err := Save(store, source, candidateA, "profile-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Save(store, source, candidateB, "profile-b")
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make([]error, 2)
	ids := []string{first.ID, second.ID}
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; _, errs[i] = Apply(store, ids[i]) }(i)
	}
	close(start)
	wg.Wait()
	success, stale := 0, 0
	for _, err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrStale) {
			stale++
		} else {
			t.Fatalf("unexpected apply error: %v", err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("success=%d stale=%d errors=%v", success, stale, errs)
	}
}

func TestApplyRejectsSourceSymlinkAndSpecialMode(t *testing.T) {
	_, repo, store, s, _, m := saved(t)
	move := filepath.Join(repo, "real.md")
	if err := os.Rename(s.Path, move); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(move, s.Path); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(store, m.ID); err == nil {
		t.Fatal("accepted source symlink")
	}
	_, _, store2, s2, _, m2 := saved(t)
	if err := os.Chmod(s2.Path, os.ModeSetuid|0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(store2, m2.ID); err == nil {
		t.Fatal("accepted special permission bits")
	}
}

func TestApplyDoesNotReplaceIfFinalCASChanged(t *testing.T) {
	_, _, store, s, _, m := saved(t)
	if err := os.WriteFile(s.Path, []byte("different"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(store, m.ID); !errors.Is(err, ErrStale) {
		t.Fatalf("got %v", err)
	}
	got, err := os.ReadFile(s.Path)
	if err != nil || string(got) != "different" {
		t.Fatalf("source=%q err=%v", got, err)
	}
}

func TestBackupSyncFailureLeavesOriginalAndRecoveryRestoresBytes(t *testing.T) {
	_, _, store, source, _, manifest := saved(t)
	failure := errors.New("backup sync failed")
	if _, err := applyWithSync(store, manifest.ID, func(string) error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("expected backup sync failure: %v", err)
	}
	got, err := os.ReadFile(source.Path)
	if err != nil || string(got) != string(source.Bytes) {
		t.Fatalf("source changed before durable backup: %v", err)
	}
	backup, err := Apply(store, manifest.ID)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(source.Path, original, source.Mode.Perm()); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(source.Path)
	if err != nil || string(restored) != string(source.Bytes) {
		t.Fatal("explicit backup restoration did not restore exact bytes", err)
	}
}

func TestNoGitPlanStillRejectsCandidateStorageInSourceFolder(t *testing.T) {
	folder := t.TempDir()
	path := filepath.Join(folder, "plan.md")
	if err := os.WriteFile(path, []byte("original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source, err := ReadSource(path)
	if err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(folder, "private")
	if _, err = Save(store, source, []byte("candidate\n"), "profile"); err == nil {
		t.Fatal("accepted store in standalone plan folder")
	}
	if _, err = os.Stat(store); !os.IsNotExist(err) {
		t.Fatal("rejected store created directory")
	}
}
