package observatory

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
}

func TestSelectedScanExcludesCredentialLikePaths(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"main.go": "package p", ".env": "TOKEN=secret", "id_rsa": "private", "private-key.pem": "private", "spec.test.ts": "test"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := New(root, t.TempDir(), "node", "bridge")
	files, _ := s.scanFiles(root)
	got := map[string]bool{}
	for _, f := range files {
		got[f.Path] = true
	}
	if !got["main.go"] || !got["spec.test.ts"] {
		t.Fatalf("expected code/test files in snapshot: %#v", got)
	}
	for _, name := range []string{".env", "id_rsa", "private-key.pem"} {
		if got[name] {
			t.Errorf("sensitive file included: %s", name)
		}
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
