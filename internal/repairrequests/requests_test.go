package repairrequests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testDir(t *testing.T) string {
	t.Helper()
	base := "/Volumes/BuildOffload/worktrees"
	dir, err := os.MkdirTemp(base, "repairrequests-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func testKey(n byte) string { return strings.Repeat(string([]byte{n}), 64) }

func TestRunDurablePrivateReceiptAndExactReuse(t *testing.T) {
	dir := filepath.Join(testDir(t), "cache")
	key := testKey('a')
	inputMarker, output := []byte("private input marker"), []byte("private response marker")
	var calls atomic.Int32
	got, err := Run(context.Background(), dir, key, func(context.Context) ([]byte, error) {
		calls.Add(1)
		b, err := os.ReadFile(filepath.Join(dir, key+".json"))
		if err != nil {
			t.Fatalf("receipt missing before generate: %v", err)
		}
		var r receipt
		if json.Unmarshal(b, &r) != nil || r.State != "dispatching" {
			t.Fatalf("receipt not durably dispatching before callback: %s", b)
		}
		if strings.Contains(string(b), string(inputMarker)) {
			t.Fatal("receipt contains input body")
		}
		return output, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(output) {
		t.Fatalf("got %q", got)
	}
	for _, p := range []string{dir, filepath.Join(dir, key+".json"), filepath.Join(dir, key+".body"), filepath.Join(dir, ".lock"), filepath.Join(dir, ".key-"+key)} {
		st, e := os.Stat(p)
		if e != nil {
			t.Fatal(e)
		}
		want := os.FileMode(0600)
		if p == dir {
			want = 0700
		}
		if st.Mode().Perm() != want {
			t.Fatalf("%s mode %o, want %o", filepath.Base(p), st.Mode().Perm(), want)
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, key+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), string(inputMarker)) || strings.Contains(string(b), string(output)) {
		t.Fatal("receipt contains a body")
	}
	got, err = Run(context.Background(), dir, key, func(context.Context) ([]byte, error) { calls.Add(1); return nil, errors.New("must not call") })
	if err != nil || string(got) != string(output) || calls.Load() != 1 {
		t.Fatalf("cached reuse got=%q err=%v calls=%d", got, err, calls.Load())
	}
}

func TestRunKeyNamespaceAndValidation(t *testing.T) {
	dir := filepath.Join(testDir(t), "cache")
	if _, err := Run(context.Background(), dir, strings.Repeat("A", 64), func(context.Context) ([]byte, error) { t.Fatal("called"); return nil, nil }); !errors.Is(err, ErrBadKey) {
		t.Fatalf("bad key: %v", err)
	}
	for _, key := range []string{testKey('b'), testKey('c')} {
		if _, err := Run(context.Background(), dir, key, func(context.Context) ([]byte, error) { return []byte(key[:1]), nil }); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, testKey('b')+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, testKey('c')+".json")); err != nil {
		t.Fatal(err)
	}
}

func TestRunUnknownNeverResends(t *testing.T) {
	dir, key := filepath.Join(testDir(t), "cache"), testKey('d')
	var calls atomic.Int32
	_, err := Run(context.Background(), dir, key, func(context.Context) ([]byte, error) { calls.Add(1); return nil, errors.New("timeout") })
	if !errors.Is(err, ErrUnknown) {
		t.Fatalf("first error %v", err)
	}
	_, err = Run(context.Background(), dir, key, func(context.Context) ([]byte, error) { calls.Add(1); return []byte("retry"), nil })
	if !errors.Is(err, ErrUnknown) || calls.Load() != 1 {
		t.Fatalf("automatic resend: err=%v calls=%d", err, calls.Load())
	}
}

func TestRunStrandedReceiptAndMissingOrCorruptBodyFailClosed(t *testing.T) {
	dir, key := filepath.Join(testDir(t), "cache"), testKey('e')
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writePrivate(filepath.Join(dir, key+".json"), mustReceipt(t, receipt{Key: key, State: "dispatching", CreatedAt: time.Now().UTC()})); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	if _, err := Run(context.Background(), dir, key, func(context.Context) ([]byte, error) { calls.Add(1); return nil, nil }); !errors.Is(err, ErrUnknown) || calls.Load() != 0 {
		t.Fatalf("stranded resend err=%v calls=%d", err, calls.Load())
	}
	key2 := testKey('f')
	if _, err := Run(context.Background(), dir, key2, func(context.Context) ([]byte, error) { return []byte("body"), nil }); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, key2+".body"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), dir, key2, func(context.Context) ([]byte, error) { calls.Add(1); return nil, nil }); !errors.Is(err, ErrCorrupt) || calls.Load() != 0 {
		t.Fatalf("corrupt body resent err=%v calls=%d", err, calls.Load())
	}
}

func TestRunExpiryAndCapacity(t *testing.T) {
	dir := filepath.Join(testDir(t), "cache")
	key := testKey('1')
	if _, err := Run(context.Background(), dir, key, func(context.Context) ([]byte, error) { return []byte("old"), nil }); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, key+".json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var r receipt
	if err = json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	r.ExpiresAt = time.Now().Add(-time.Second)
	if err = os.WriteFile(p, mustReceipt(t, r), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	if _, err = Run(context.Background(), dir, key, func(context.Context) ([]byte, error) { calls.Add(1); return []byte("new"), nil }); err != nil || calls.Load() != 1 {
		t.Fatalf("expiry regenerate err=%v calls=%d", err, calls.Load())
	}
	for i := 2; i < maxOccupied+1; i++ {
		k := fmt.Sprintf("%064x", i)
		if _, err = Run(context.Background(), dir, k, func(context.Context) ([]byte, error) { return []byte("x"), nil }); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = Run(context.Background(), dir, testKey('f'), func(context.Context) ([]byte, error) { calls.Add(1); return []byte("no"), nil }); !errors.Is(err, ErrFull) || calls.Load() != 1 {
		t.Fatalf("capacity err=%v calls=%d", err, calls.Load())
	}
}

func TestRunSameKeyConcurrentDeduplicates(t *testing.T) {
	dir, key := filepath.Join(testDir(t), "cache"), testKey('9')
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	var wg sync.WaitGroup
	wg.Add(2)
	results := make([][]byte, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = Run(context.Background(), dir, key, func(context.Context) ([]byte, error) {
				if calls.Add(1) == 1 {
					close(entered)
				}
				<-release
				return []byte("one"), nil
			})
		}(i)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("generator did not start")
	}
	close(release)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
	for i := range results {
		if errs[i] != nil || string(results[i]) != "one" {
			t.Fatalf("result %d: %q %v", i, results[i], errs[i])
		}
	}
}

func TestRunContextCancelsWaitingForCrossProcessLock(t *testing.T) {
	dir, key := filepath.Join(testDir(t), "cache"), testKey('8')
	if _, err := Run(context.Background(), dir, testKey('7'), func(context.Context) ([]byte, error) { return []byte("x"), nil }); err != nil {
		t.Fatal(err)
	}
	unlock, err := lockNamed(context.Background(), dir, ".key-"+key)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, dir, key, func(context.Context) ([]byte, error) { t.Fatal("called"); return nil, nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestRunRejectsSymlinkAndOversizeResponse(t *testing.T) {
	base := testDir(t)
	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), link, testKey('6'), func(context.Context) ([]byte, error) { t.Fatal("called"); return nil, nil }); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("symlink err=%v", err)
	}
	dir := filepath.Join(base, "cache")
	if _, err := Run(context.Background(), dir, testKey('5'), func(context.Context) ([]byte, error) { return make([]byte, maxResponse+1), nil }); !errors.Is(err, ErrFailed) {
		t.Fatalf("oversize err=%v", err)
	}
}

func mustReceipt(t *testing.T, r receipt) []byte {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRunDefinitiveFailureIsRetainedWithoutResend(t *testing.T) {
	dir, key := filepath.Join(testDir(t), "cache"), testKey('4')
	var calls atomic.Int32
	_, err := Run(context.Background(), dir, key, func(context.Context) ([]byte, error) {
		calls.Add(1)
		return nil, Failed(errors.New("provider rejected request"))
	})
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("first error %v", err)
	}
	_, err = Run(context.Background(), dir, key, func(context.Context) ([]byte, error) { calls.Add(1); return []byte("retry"), nil })
	if !errors.Is(err, ErrFailed) || calls.Load() != 1 {
		t.Fatalf("resend: err=%v calls=%d", err, calls.Load())
	}
}
