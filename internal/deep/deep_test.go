package deep

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
		m.Context = []ContextItem{{OwnerRef: "r1", Kind: "decision", EntityID: "e1", BrainID: "brain", AudienceID: "owner", ProjectID: "project", ContentDigest: digest(ctxBody), Version: "v1", ExpiresAt: time.Now().Add(time.Hour), Body: ctxBody}}
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
	s, err := New(Config{Dir: dir, MaxBytes: 8 << 20, Engine: e, Validator: v})
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
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
	if _, err = Key(m); err == nil {
		t.Fatal("scope/reference mismatch accepted")
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls after invalid manifest=%d", calls.Load())
	}
}

func TestMemoryUnavailableHidesWithoutDeletingAndInvalidationRemovesBody(t *testing.T) {
	var vOK atomic.Bool
	vOK.Store(true)
	v := validatorFunc(func(context.Context, ContextScope, []ContextItem) (LineageValidation, error) {
		if !vOK.Load() {
			return LineageValidation{Available: false}, nil
		}
		return LineageValidation{Available: true, Valid: true}, nil
	})
	s, _ := service(t, engineFunc(func(context.Context, Manifest) (string, error) { return "memory answer", nil }), v)
	m := manifest(ContextMemory)
	r, err := s.Analyze(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	vOK.Store(false)
	if _, err = s.InspectForRepository(context.Background(), r.Key, m.RepositoryID); !errors.Is(err, ErrMemoryUnavailable) {
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
	if _, err = s.InspectForRepository(context.Background(), r.Key, m.RepositoryID); !errors.Is(err, ErrLineageInvalid) {
		t.Fatalf("invalid inspect err=%v", err)
	}
	stored, err = s.read(r.Key)
	if err != nil || stored.Answer != "" || stored.Status != "invalidated" || stored.Manifest.Context[0].Body != "" {
		t.Fatalf("invalidated body remains: %+v err=%v", stored, err)
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
