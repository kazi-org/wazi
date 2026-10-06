package repairai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLoadConfigProcessEnvironmentWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chosen.env")
	if err := os.WriteFile(path, []byte("EXPLABS_API_KEY='file-secret'\nEXPLABS_BASE_URL=https://api.experientiallabs.ai/v1/\nEXPLABS_MODEL=model-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EXPLABS_API_KEY", "process-secret")
	t.Setenv("EXPLABS_BASE_URL", "https://api.experientiallabs.ai/v1/")
	t.Setenv("EXPLABS_MODEL", "process-model")
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.APIKey != "process-secret" || c.Model != "process-model" || c.BaseURL != "https://api.experientiallabs.ai/v1/" {
		t.Fatalf("process environment did not win: %#v", c)
	}
}

func TestLoadConfigRejectsUnsafeEnvFileAndEndpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("EXPLABS_API_KEY=key\nEXPLABS_MODEL=model\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EXPLABS_API_KEY", "")
	t.Setenv("EXPLABS_MODEL", "")
	t.Setenv("EXPLABS_BASE_URL", "")
	for _, key := range relevantEnv {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("accepted group-readable env file")
	}
	for _, endpoint := range []string{"http://api.experientiallabs.ai/v1", "https://u:p@api.experientiallabs.ai/v1", "https://api.experientiallabs.ai:444/v1", "https://api.experientiallabs.ai/v1?x=y", "https://api.experientiallabs.ai/v2"} {
		if validateBaseURL(endpoint) == nil {
			t.Errorf("accepted endpoint %q", endpoint)
		}
	}
}

func TestLoadConfigRejectsDuplicateRelevantVariables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "private.env")
	if err := os.WriteFile(path, []byte("EXPLABS_API_KEY=first\nEXPLABS_API_KEY=second\nEXPLABS_MODEL=model\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EXPLABS_API_KEY", "")
	t.Setenv("EXPLABS_MODEL", "")
	t.Setenv("EXPLABS_BASE_URL", "")
	for _, key := range relevantEnv {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("accepted duplicate credential definition")
	}
	for _, model := range []string{"model\nsecret", "api_key=abcdefghijklmnop", "sk-abcdefghijklmnop", "contains space"} {
		if validModel(model, "api-secret") {
			t.Errorf("accepted unsafe model value %q", model)
		}
	}
}

func TestValidateOnlyPermittedSyntax(t *testing.T) {
	source := []byte("- [ ]T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [x] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n")
	candidate := []byte("- [ ] T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [x] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n")
	if err := Validate(source, candidate); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{
		[]byte("- [x] T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [x] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n"),
		[]byte("- [ ] T1.0\n  Owner: changed\n  Stage: verify\n  Acceptance: [ok]\n- [x] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n"),
		[]byte("-[x] T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [x] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n"),
	} {
		if err := Validate(source, bad); err == nil {
			t.Errorf("accepted invalid proposal %q", bad)
		}
	}
	if err := Validate([]byte("-[x] T1.0\n"), []byte("- [x] T1.0\n")); err == nil {
		t.Fatal("missing metadata passed pinned profile")
	}
}

func TestValidateRepairsJoinedSpacingAndPreservesProtectedRegions(t *testing.T) {
	source := []byte("-[ ]T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n<!--\n- [x] T9.0\n-->\n```md\n- [x] T9.1\n```\n")
	want := []byte("- [ ] T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n<!--\n- [x] T9.0\n-->\n```md\n- [x] T9.1\n```\n")
	if err := Validate(source, want); err != nil {
		t.Fatal(err)
	}
	bad := bytes.Replace(want, []byte("<!--\n- [x] T9.0"), []byte("<!--\n- [x] T9.9"), 1)
	if err := Validate(source, bad); err == nil {
		t.Fatal("proposal changed comment bytes")
	}
	if _, err := normalizeAI([]byte("-[?] unclear\n")); err == nil {
		t.Fatal("accepted ambiguous joined checkbox")
	}
}

func TestProposeUsesSingleConstrainedRequestAndSanitizes(t *testing.T) {
	source := []byte("- [ ]T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n")
	candidate := "- [ ] T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n"
	cfg := Config{APIKey: "secret-test-value", BaseURL: productionBaseURL, Model: "model-test"}
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPost || r.URL.String() != productionBaseURL+"/chat/completions" || r.Header.Get("Authorization") != "Bearer secret-test-value" {
			t.Errorf("bad request metadata")
		}
		b, _ := io.ReadAll(r.Body)
		body := string(b)
		for _, frag := range []string{`"max_tokens":16384`, `"max_total_attempts":1`, `"max_attempts_per_route":1`, `"allow_fallbacks":false`, `"stream":false`, quote(string(source))} {
			if !strings.Contains(body, frag) {
				t.Errorf("request missing required request value")
			}
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"choices":[{"finish_reason":"stop","message":{"content":` + quote(candidate) + `}}]}`))}, nil
	})}
	got, err := propose(context.Background(), cfg, source, client)
	if err != nil || string(got) != candidate || calls != 1 {
		t.Fatalf("proposal=%q calls=%d err=%v", got, calls, err)
	}
}

func TestValidateRejectsProfileChangesToCommentOpeningClosingAndInlineLines(t *testing.T) {
	tests := []struct{ name, source string }{
		{"closing line", "- [ ] T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n<!--\n- [X] T9.0 -->\n"},
		{"mixed inline comment", "- [X] T1.0 <!-- preserve this comment -->\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			if err := Validate(src, src); err == nil {
				t.Fatal("accepted source whose pinned profile changes protected comment bytes")
			}
		})
	}
}

func TestProposeRefusesCredentialsAndInvalidResponses(t *testing.T) {
	cfg := Config{APIKey: "secret", BaseURL: productionBaseURL, Model: "model"}
	called := false
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { called = true; return nil, nil })}
	for _, input := range []string{"API_KEY=abcdefghijklmnop", "EXPLABS_API_KEY=longsentinelvalue", "Authorization: Bearer longbearertokensentinel", "sk-abcdefghijklmnop"} {
		if _, err := propose(context.Background(), cfg, []byte(input), client); err == nil || called {
			t.Fatalf("credential-like selected text was sent: %q", input)
		}
	}
	for _, body := range []string{
		`{"choices":[{"finish_reason":"length","message":{"content":"x"}}]}`,
		`{"choices":[{"finish_reason":"stop","message":{"content":"x","refusal":"no"}}]}`,
		`{"choices":[{"finish_reason":"stop","message":{"content":"x","tool_calls":[]}}]}`,
		`{"choices":[],"choices":[]}`,
		`{"choices":[{"finish_reason":"stop","message":{"content":"x"}}],"x-experiential-ignored-parameters":["max_tokens"]}`,
	} {
		if _, err := responseText([]byte(body)); err == nil {
			t.Errorf("accepted response %s", body)
		}
	}
}

func TestProposeMarksUnknownTransportAndServerOutcomes(t *testing.T) {
	source := []byte("- [ ] T1.0\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.1\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n- [ ] T1.2\n  Owner: team\n  Stage: verify\n  Acceptance: [ok]\n")
	cfg := Config{APIKey: "private-secret", BaseURL: productionBaseURL, Model: "model"}
	for _, tc := range []struct {
		name string
		rt   roundTripFunc
	}{
		{"transport", func(*http.Request) (*http.Response, error) { return nil, errors.New("private-secret transport detail") }},
		{"server", func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("private-secret response detail"))}, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { calls++; return tc.rt(r) })}
			_, err := propose(context.Background(), cfg, source, client)
			if !errors.Is(err, ErrUncertain) || calls != 1 || strings.Contains(err.Error(), "private-secret") {
				t.Fatalf("uncertain classification/sanitization failed: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestIdentityBindsNonsecretInputs(t *testing.T) {
	c := Config{APIKey: "one", BaseURL: productionBaseURL, Model: "m"}
	base := Identity(c, "plan.md", []byte("source"))
	c.APIKey = "two"
	otherKey := Identity(c, "plan.md", []byte("source"))
	if otherKey == base || strings.Contains(otherKey, c.APIKey) {
		t.Fatal("identity did not bind API-key scope safely")
	}
	c.Model = "other"
	if Identity(c, "plan.md", []byte("source")) == otherKey {
		t.Fatal("model did not influence identity")
	}
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }
