package deep

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type engineFunc func(context.Context, Manifest) (string, error)

func (f engineFunc) Complete(c context.Context, m Manifest) (string, error) { return f(c, m) }

type validatorFunc func(context.Context, ContextScope, []ContextItem) (LineageValidation, error)

func (f validatorFunc) ValidateLineage(c context.Context, s ContextScope, r []ContextItem) (LineageValidation, error) {
	return f(c, s, r)
}

func manifest(mode ContextMode) Manifest {
	p := "selected task plan"
	body := "selected source"
	c := CodeInput{Path: "src/main.go", Body: body, SHA256: digest(body)}
	cd := c.Path + "\x00" + c.SHA256 + "\n"
	m := Manifest{RepositoryID: "repo-1", TaskRef: "plans/work.md#T1", Question: "How does this work?", PlanDigest: digest(p), CodeDigest: digest(cd), PlanBody: p, Code: []CodeInput{c}, PromptVersion: "p1", AnalyzerVersion: "a1", Model: Model, Settings: []byte(`{"max_tokens":128}`), ContextMode: mode}
	if mode == ContextMemory {
		ctxBody := "displayed context"
		m.ContextScope = ContextScope{RepositoryID: m.RepositoryID, BrainID: "brain", AudienceID: "owner", ProjectID: "project", EntityIDs: []string{"e1"}, ReferenceIDs: []string{"r1"}}
		m.Context = []ContextItem{{OwnerRef: "r1", Kind: "decision", EntityID: "e1", BrainID: "brain", AudienceID: "owner", ProjectID: "project", ContentDigest: "sha256:" + digest(ctxBody), Version: "v1", ExpiresAt: time.Now().Add(time.Hour), Body: ctxBody}}
	}
	return m
}
func service(t *testing.T, e Engine, v LineageValidator) (*Service, string) {
	t.Helper()
	dir := filepath.Join(os.Getenv("TMPDIR"), "wazi-deep-test-"+strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	s, err := New(Config{Dir: dir, MaxBytes: 8 << 20, Engine: e, Validator: v, MemoryPersistenceQualified: v != nil})
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

func TestCompletedPlanCodeCanBeReadOffline(t *testing.T) {
	dir := filepath.Join(os.Getenv("TMPDIR"), "wazi-deep-offline-"+strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	m := manifest(ContextPlanCode)
	m.Code = nil
	m.CodeDigest = ComputeCodeDigest(nil)
	if _, err := Key(m); err != nil {
		t.Fatalf("plan-only manifest rejected: %v", err)
	}
	first, err := New(Config{Dir: dir, MaxBytes: 8 << 20, Engine: engineFunc(func(context.Context, Manifest) (string, error) { return "saved", nil })})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = first.Analyze(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	offline, err := New(Config{Dir: dir, MaxBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	r, err := offline.Analyze(context.Background(), m)
	if err != nil || r.Answer != "saved" || !r.Cached {
		t.Fatalf("offline cache=%+v err=%v", r, err)
	}
}

func TestAnalyzeCachesOnlyCompletedResult(t *testing.T) {
	var calls atomic.Int32
	s, _ := service(t, engineFunc(func(context.Context, Manifest) (string, error) { calls.Add(1); return "answer", nil }), nil)
	m := manifest(ContextPlanCode)
	first, err := s.Analyze(context.Background(), m)
	if err != nil || first.Answer != "answer" {
		t.Fatalf("Analyze=%+v err=%v", first, err)
	}
	second, err := s.Analyze(context.Background(), m)
	if err != nil || second.Answer != "answer" {
		t.Fatalf("cached Analyze=%+v err=%v", second, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls=%d", calls.Load())
	}
}

func TestUnknownReceiptPreventsAutomaticResend(t *testing.T) {
	var calls atomic.Int32
	s, _ := service(t, engineFunc(func(context.Context, Manifest) (string, error) {
		calls.Add(1)
		return "", errors.New("connection lost")
	}), nil)
	if _, err := s.Analyze(context.Background(), manifest(ContextPlanCode)); err == nil {
		t.Fatal("expected dispatch failure")
	}
	if _, err := s.Analyze(context.Background(), manifest(ContextPlanCode)); !errors.Is(err, ErrUnknownOutcome) {
		t.Fatalf("second Analyze err=%v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls=%d", calls.Load())
	}
}

func TestKernelLockReleasedWhenOwnerProcessExits(t *testing.T) {
	if os.Getenv("WAZI_LOCK_CRASH_HELPER") == "1" {
		s := &Service{dir: os.Getenv("WAZI_LOCK_CRASH_DIR")}
		if _, err := s.acquire(context.Background(), "lifecycle"); err != nil {
			os.Exit(2)
		}
		// Intentionally exit without unlocking; the kernel must release flock.
		os.Exit(0)
	}
	dir := filepath.Join(os.Getenv("TMPDIR"), "wazi-deep-lock-crash-"+strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	cmd := exec.Command(os.Args[0], "-test.run=^TestKernelLockReleasedWhenOwnerProcessExits$")
	cmd.Env = append(os.Environ(), "WAZI_LOCK_CRASH_HELPER=1", "WAZI_LOCK_CRASH_DIR="+dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("lock owner helper failed: %v: %s", err, output)
	}
	s := &Service{dir: dir}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	unlock, err := s.acquire(ctx, "lifecycle")
	if err != nil {
		t.Fatalf("kernel lock remained stuck after owner process exit: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, ".lifecycle.lock"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("persistent lock file mode: info=%v err=%v", info, err)
	}
	unlock()
}

func TestPersistentDispatchingReceiptRequiresExplicitRegenerate(t *testing.T) {
	var calls atomic.Int32
	dir := filepath.Join(os.Getenv("TMPDIR"), "wazi-deep-restart-"+strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	m := manifest(ContextPlanCode)
	key, err := Key(m)
	if err != nil {
		t.Fatal(err)
	}
	first, err := New(Config{Dir: dir, MaxBytes: 8 << 20, Engine: engineFunc(func(context.Context, Manifest) (string, error) { calls.Add(1); return "unexpected", nil })})
	if err != nil {
		t.Fatal(err)
	}
	if err = first.write(receipt{Key: key, Status: "dispatching", Manifest: metadataManifest(m), UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	restarted, err := New(Config{Dir: dir, MaxBytes: 8 << 20, Engine: engineFunc(func(context.Context, Manifest) (string, error) { calls.Add(1); return "regenerated", nil })})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = restarted.Analyze(context.Background(), m); !errors.Is(err, ErrUnknownOutcome) {
		t.Fatalf("restart Analyze err=%v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("unknown receipt auto-dispatched %d times", calls.Load())
	}
	result, err := restarted.Regenerate(context.Background(), m, CostDisclosure{Summary: "explicit OpenRouter cost disclosure", Acknowledged: true})
	if err != nil || result.Answer != "regenerated" {
		t.Fatalf("explicit regenerate=%+v err=%v", result, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("explicit regeneration calls=%d", calls.Load())
	}
}

func TestReceiptsNeverPersistRequestBodies(t *testing.T) {
	dir := filepath.Join(os.Getenv("TMPDIR"), "wazi-deep-redaction-"+strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	m := manifest(ContextMemory)
	m.Question = "private-question-marker"
	m.PlanBody = "private-plan-marker"
	m.PlanDigest = digest(m.PlanBody)
	m.Code[0].Body = "private-code-marker"
	m.Code[0].SHA256 = digest(m.Code[0].Body)
	m.CodeDigest = ComputeCodeDigest(m.Code)
	m.Context[0].Body = "private-context-marker"
	m.Context[0].ContentDigest = digest(m.Context[0].Body)
	key, err := Key(m)
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan bool, 1), make(chan struct{})
	engine := engineFunc(func(_ context.Context, received Manifest) (string, error) {
		started <- received.Question == m.Question && received.PlanBody == m.PlanBody && received.Code[0].Body == m.Code[0].Body && received.Context[0].Body == m.Context[0].Body
		<-release
		return "answer without source", nil
	})
	validator := validatorFunc(func(context.Context, ContextScope, []ContextItem) (LineageValidation, error) {
		return LineageValidation{Available: true, Valid: true, ValidUntil: time.Now().Add(time.Hour)}, nil
	})
	s, err := New(Config{Dir: dir, MaxBytes: 8 << 20, Engine: engine, Validator: validator, MemoryPersistenceQualified: true})
	if err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { _, analyzeErr := s.Analyze(context.Background(), m); finished <- analyzeErr }()
	if !<-started {
		close(release)
		t.Fatal("engine did not receive the original request bodies")
	}
	assertRedacted := func() {
		t.Helper()
		raw, e := os.ReadFile(s.filename(key))
		if e != nil {
			t.Fatal(e)
		}
		for _, marker := range []string{"private-question-marker", "private-plan-marker", "private-code-marker", "private-context-marker"} {
			if bytes.Contains(raw, []byte(marker)) {
				t.Fatalf("receipt persisted request body marker %q", marker)
			}
		}
		saved, e := s.read(key)
		if e != nil {
			t.Fatal(e)
		}
		if saved.Manifest.Question != "" || saved.Manifest.PlanBody != "" || saved.Manifest.Code[0].Body != "" || saved.Manifest.Context[0].Body != "" {
			t.Fatalf("receipt kept source bodies: %+v", saved.Manifest)
		}
	}
	assertRedacted()
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	assertRedacted()
}

func TestCapacityAndManifestScopePreflightBeforeProvider(t *testing.T) {
	var calls atomic.Int32
	engine := engineFunc(func(context.Context, Manifest) (string, error) { calls.Add(1); return "answer", nil })
	dir := filepath.Join(os.Getenv("TMPDIR"), "wazi-deep-capacity-"+strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	s, err := New(Config{Dir: dir, MaxBytes: 4096, Engine: engine})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Analyze(context.Background(), manifest(ContextPlanCode)); !errors.Is(err, ErrCapacity) {
		t.Fatalf("capacity error=%v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls before capacity preflight=%d", calls.Load())
	}
	m := manifest(ContextMemory)
	m.ContextScope.ReferenceIDs = []string{"not-the-displayed-ref"}
	m.ContextScope.EntityIDs = []string{"not-the-displayed-entity"}
	if _, err = Key(m); err == nil {
		t.Fatal("scope/reference mismatch accepted")
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls after invalid manifest=%d", calls.Load())
	}
}

func TestManifestScopeAllowsReferenceOrEntityQualification(t *testing.T) {
	t.Run("entity-only", func(t *testing.T) {
		m := manifest(ContextMemory)
		m.ContextScope.ReferenceIDs = nil
		if _, err := Key(m); err != nil {
			t.Fatalf("mapped entity scope rejected: %v", err)
		}
	})
	t.Run("reference-only", func(t *testing.T) {
		m := manifest(ContextMemory)
		m.ContextScope.EntityIDs = nil
		m.Context[0].EntityID = ""
		if _, err := Key(m); err != nil {
			t.Fatalf("mapped reference scope rejected: %v", err)
		}
	})
	t.Run("neither", func(t *testing.T) {
		m := manifest(ContextMemory)
		m.ContextScope.EntityIDs = []string{"unmapped-entity"}
		m.ContextScope.ReferenceIDs = []string{"unmapped-reference"}
		if _, err := Key(m); err == nil {
			t.Fatal("unmapped reference and entity were accepted")
		}
	})
}

func TestMemoryUnavailableHidesWithoutDeletingAndInvalidationRemovesBody(t *testing.T) {
	var vOK atomic.Bool
	vOK.Store(true)
	v := validatorFunc(func(context.Context, ContextScope, []ContextItem) (LineageValidation, error) {
		if !vOK.Load() {
			return LineageValidation{Available: false}, nil
		}
		return LineageValidation{Available: true, Valid: true, ValidUntil: time.Now().Add(time.Hour)}, nil
	})
	s, _ := service(t, engineFunc(func(context.Context, Manifest) (string, error) { return "memory answer", nil }), v)
	m := manifest(ContextMemory)
	r, err := s.Analyze(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	vOK.Store(false)
	if _, err = s.InspectForManifest(context.Background(), m); !errors.Is(err, ErrMemoryUnavailable) {
		t.Fatalf("offline inspect err=%v", err)
	}
	stored, err := s.read(r.Key)
	if err != nil || stored.Answer != "memory answer" {
		t.Fatalf("offline read removed body: %+v err=%v", stored, err)
	}
	vOK.Store(true)
	v = validatorFunc(func(context.Context, ContextScope, []ContextItem) (LineageValidation, error) {
		return LineageValidation{Available: true, Valid: false}, nil
	})
	s.validator = v
	if _, err = s.InspectForManifest(context.Background(), m); !errors.Is(err, ErrLineageInvalid) {
		t.Fatalf("invalid inspect err=%v", err)
	}
	stored, err = s.read(r.Key)
	if err != nil || stored.Answer != "" || stored.Status != "invalidated" || stored.Manifest.Context[0].Body != "" {
		t.Fatalf("invalidated body remains: %+v err=%v", stored, err)
	}
}

func TestMemoryExpiryDuringEnginePreventsDisplay(t *testing.T) {
	var expired atomic.Bool
	validator := validatorFunc(func(context.Context, ContextScope, []ContextItem) (LineageValidation, error) {
		if expired.Load() {
			return LineageValidation{Available: true, Valid: false, ValidUntil: time.Now().Add(-time.Second)}, nil
		}
		return LineageValidation{Available: true, Valid: true, ValidUntil: time.Now().Add(time.Hour)}, nil
	})
	started, release := make(chan struct{}), make(chan struct{})
	s, _ := service(t, engineFunc(func(context.Context, Manifest) (string, error) {
		close(started)
		<-release
		return "must not display", nil
	}), validator)
	m := manifest(ContextMemory)
	key, err := Key(m)
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		result Result
		err    error
	}
	done := make(chan outcome, 1)
	go func() { r, e := s.Analyze(context.Background(), m); done <- outcome{result: r, err: e} }()
	<-started
	expired.Store(true)
	close(release)
	got := <-done
	if !errors.Is(got.err, ErrLineageInvalid) || got.result.Answer != "" {
		t.Fatalf("expired result surfaced: %+v err=%v", got.result, got.err)
	}
	saved, err := s.read(key)
	if err != nil || saved.Status != "invalidated" || saved.Answer != "" {
		t.Fatalf("expired result remains cached: %+v err=%v", saved, err)
	}
}

func TestOpenRouterContractWithHTTPTestServer(t *testing.T) {
	var gotAuth, gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var in chatRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Errorf("decode: %v", err)
		}
		gotModel = in.Model
		if len(in.Messages) != 1 || !strings.Contains(in.Messages[0].Content, "selected task plan") || !strings.Contains(in.Messages[0].Content, "displayed context") {
			t.Errorf("request omitted selected input: %+v", in)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"local answer"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	client := &http.Client{Timeout: time.Second, Transport: rewriteTransport{target: server.URL}}
	engine := newOpenRouterForTest("test-token", client)
	m := manifest(ContextMemory)
	answer, err := engine.Complete(context.Background(), m)
	if err != nil || answer != "local answer" || gotAuth != "Bearer test-token" || gotModel != Model {
		t.Fatalf("answer=%q auth=%q model=%q err=%v", answer, gotAuth, gotModel, err)
	}
}

type rewriteTransport struct{ target string }

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	copy.URL = mustParse(r.target)
	copy.Host = copy.URL.Host
	return http.DefaultTransport.RoundTrip(copy)
}
func mustParse(s string) *url.URL {
	u, e := url.Parse(s)
	if e != nil {
		panic(e)
	}
	return u
}
