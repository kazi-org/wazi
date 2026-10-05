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
