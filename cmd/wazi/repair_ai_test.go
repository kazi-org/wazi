package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kazi-org/wazi/internal/deep"
	"github.com/kazi-org/wazi/internal/repairai"
)

func TestAIRepairProposalCacheAndSeparateApply(t *testing.T) {
	t.Setenv("EXPLABS_API_KEY", "test-private-key")
	t.Setenv("EXPLABS_BASE_URL", "https://api.experientiallabs.ai/v1")
	t.Setenv("EXPLABS_MODEL", "test-model")
	old, oldExclude := proposeRepairAI, excludeRepairCache
	defer func() { proposeRepairAI = old; excludeRepairCache = oldExclude }()
	excludeRepairCache = func(context.Context, string) (deep.BackupStatus, error) { return deep.BackupStatus{}, nil }
	calls := 0
	source := []byte("# Sample\n-[X] T1.0 Meaning Owner: owner kind: agent stage: implement acc: [observable]\n")
	candidate := []byte("# Sample\n- [x] T1.0 Meaning Owner: owner kind: agent stage: implement acc: [observable]\n")
	proposeRepairAI = func(ctx context.Context, cfg repairai.Config, b []byte) ([]byte, error) {
		calls++
		if !bytes.Equal(b, source) {
			t.Fatal("wrong selected input")
		}
		return candidate, nil
	}
	file := filepath.Join(t.TempDir(), "plan.md")
	if e := os.WriteFile(file, source, 0600); e != nil {
		t.Fatal(e)
	}
	data := filepath.Join(t.TempDir(), "repair")
	var savedID string
	for i := 0; i < 2; i++ {
		var out, errs bytes.Buffer
		if c := runRepair([]string{"--ai", "--save-candidate", "--data", data, file}, &out, &errs); c != 0 {
			t.Fatalf("%d: %s", c, errs.String())
		}
		if strings.Contains(out.String()+errs.String(), "test-private-key") {
			t.Fatal("secret leaked")
		}
		if !strings.Contains(out.String(), "account-dependent content retention") {
			t.Fatal("missing disclosure")
		}
		got, e := os.ReadFile(file)
		if e != nil || !bytes.Equal(got, source) {
			t.Fatal("AI automatically wrote source")
		}
		for _, line := range strings.Split(out.String(), "\n") {
			if strings.HasPrefix(line, "Apply explicitly with:") {
				savedID = line[strings.LastIndex(line, " ")+1:]
			}
		}
	}
	if calls != 1 || savedID == "" {
		t.Fatalf("calls=%d, candidate=%q", calls, savedID)
	}
	var out, errs bytes.Buffer
	if c := runRepair([]string{"--data", data, "--apply-candidate", savedID}, &out, &errs); c != 0 {
		t.Fatal(c, errs.String())
	}
	got, e := os.ReadFile(file)
	if e != nil || !bytes.Equal(got, candidate) {
		t.Fatal("explicit apply mismatch")
	}
	if calls != 1 {
		t.Fatal("apply called AI")
	}
}

func TestAIUnknownNeverImplicitlyResends(t *testing.T) {
	t.Setenv("EXPLABS_API_KEY", "test-private-key")
	t.Setenv("EXPLABS_BASE_URL", "https://api.experientiallabs.ai/v1")
	t.Setenv("EXPLABS_MODEL", "test-model")
	old, oldExclude := proposeRepairAI, excludeRepairCache
	defer func() { proposeRepairAI = old; excludeRepairCache = oldExclude }()
	excludeRepairCache = func(context.Context, string) (deep.BackupStatus, error) { return deep.BackupStatus{}, nil }
	calls := 0
	proposeRepairAI = func(context.Context, repairai.Config, []byte) ([]byte, error) {
		calls++
		return nil, errors.Join(repairai.ErrUncertain, errors.New("transport uncertain"))
	}
	file := filepath.Join(t.TempDir(), "plan.md")
	if e := os.WriteFile(file, []byte("# Sample\n- [ ] T1.0 Meaning Owner: owner stage: implement acc: [observable]\n"), 0600); e != nil {
		t.Fatal(e)
	}
	data := filepath.Join(t.TempDir(), "repair")
	for i := 0; i < 2; i++ {
		var out, errs bytes.Buffer
		if c := runRepair([]string{"--ai", "--data", data, file}, &out, &errs); c == 0 {
			t.Fatal("unknown succeeded")
		}
	}
	if calls != 1 {
		t.Fatal("uncertain invocation automatically resent", calls)
	}
}

func TestAIMissingAuthoredMeaningRefusesBeforeRequest(t *testing.T) {
	old := proposeRepairAI
	defer func() { proposeRepairAI = old }()
	calls := 0
	proposeRepairAI = func(context.Context, repairai.Config, []byte) ([]byte, error) { calls++; return nil, nil }
	file := filepath.Join(t.TempDir(), "plan.md")
	if e := os.WriteFile(file, []byte("# Sample\n- [ ] T1.0 Meaning\n"), 0600); e != nil {
		t.Fatal(e)
	}
	data := filepath.Join(t.TempDir(), "absent")
	var out, errs bytes.Buffer
	if c := runRepair([]string{"--ai", "--data", data, file}, &out, &errs); c == 0 {
		t.Fatal("missing meaning accepted")
	}
	if calls != 0 || !strings.Contains(out.String(), "line 2") || !strings.Contains(errs.String(), "no request sent") {
		t.Fatal(calls, out.String(), errs.String())
	}
	if _, e := os.Stat(data); !os.IsNotExist(e) {
		t.Fatal("semantic rejection persisted state")
	}
}

func TestRepairConfigScope(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "plans"), 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "plans", "plan.md")
	config := filepath.Join(root, ".env")
	for _, p := range []string{source, config} {
		if err := os.WriteFile(p, []byte("sample"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{config, filepath.Join(alias, ".env")} {
		if err := checkRepairConfigScope(source, p); err == nil {
			t.Fatal("selected project config accepted", p)
		}
	}
	outside := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(outside, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkRepairConfigScope(source, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/kazi-org/wazi\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkRepairConfigScope(source, config); err == nil {
		t.Fatal("selected project module declaration alone authorized config")
	}
	t.Chdir(root)
	if err := checkRepairConfigScope(source, config); err != nil {
		t.Fatal("explicit caller Wazi configuration rejected", err)
	}
}

func TestAIRejectsSelectedProjectConfigBeforePersistence(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "plan.md")
	config := filepath.Join(root, ".env")
	if err := os.WriteFile(source, []byte("# Sample\n-[X] T1.0 Meaning Owner: owner stage: implement acc: [observable]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("EXPLABS_API_KEY=selected-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	old := proposeRepairAI
	defer func() { proposeRepairAI = old }()
	calls := 0
	proposeRepairAI = func(context.Context, repairai.Config, []byte) ([]byte, error) { calls++; return nil, nil }
	data := filepath.Join(t.TempDir(), "absent")
	var out, errs bytes.Buffer
	if code := runRepair([]string{"--ai", "--env-file", config, "--data", data, source}, &out, &errs); code == 0 {
		t.Fatal("selected config accepted")
	}
	if calls != 0 || !strings.Contains(errs.String(), "no request sent") || strings.Contains(out.String()+errs.String(), "selected-secret") {
		t.Fatal(calls, out.String(), errs.String())
	}
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Fatal("rejection persisted state")
	}
}

func TestAIAmbiguousCheckboxRefusesBeforeConfigAndPersistence(t *testing.T) {
	source := filepath.Join(t.TempDir(), "plan.md")
	if err := os.WriteFile(source, []byte("# Sample\n- [??] T1.0 Meaning Owner: owner stage: implement acc: [observable]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	old := proposeRepairAI
	defer func() { proposeRepairAI = old }()
	calls := 0
	proposeRepairAI = func(context.Context, repairai.Config, []byte) ([]byte, error) { calls++; return nil, nil }
	data := filepath.Join(t.TempDir(), "absent")
	var out, errs bytes.Buffer
	if code := runRepair([]string{"--ai", "--env-file", "/nonexistent-owner-config", "--data", data, source}, &out, &errs); code == 0 {
		t.Fatal("ambiguous syntax accepted")
	}
	if calls != 0 || !strings.Contains(errs.String(), "syntax preflight") || !strings.Contains(errs.String(), "no request sent") {
		t.Fatal(calls, out.String(), errs.String())
	}
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Fatal("rejection persisted state")
	}
}

func TestAIForeignWorkingDirectoryDoesNotDiscoverCredentials(t *testing.T) {
	for _, name := range []string{"EXPLABS_API_KEY", "EXPLABS_BASE_URL", "EXPLABS_MODEL"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	t.Chdir(root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/foreign\n// module github.com/kazi-org/wazi\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("EXPLABS_API_KEY=foreign-secret\nEXPLABS_BASE_URL=https://api.experientiallabs.ai/v1\nEXPLABS_MODEL=test-model\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "plan.md")
	if err := os.WriteFile(source, []byte("# Sample\n-[X] T1.0 Meaning Owner: owner stage: implement acc: [observable]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	old := proposeRepairAI
	defer func() { proposeRepairAI = old }()
	calls := 0
	proposeRepairAI = func(context.Context, repairai.Config, []byte) ([]byte, error) { calls++; return nil, nil }
	data := filepath.Join(t.TempDir(), "absent")
	var out, errs bytes.Buffer
	if code := runRepair([]string{"--ai", "--data", data, source}, &out, &errs); code == 0 {
		t.Fatal("foreign cwd discovered credentials")
	}
	if calls != 0 || strings.Contains(out.String()+errs.String(), "foreign-secret") {
		t.Fatal(calls, out.String(), errs.String())
	}
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Fatal("rejection persisted state")
	}
}
