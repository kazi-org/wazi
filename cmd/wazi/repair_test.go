package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairHelpAndDisabledModesDoNotWrite(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--ai"}, {"--env-file", "missing"}, {"--interactive"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			dir := t.TempDir()
			data := filepath.Join(dir, "not-created")
			var out, errs bytes.Buffer
			code := runRepair(append([]string{"--data", data}, args...), &out, &errs)
			if args[0] == "--help" && code != 0 {
				t.Fatal(code)
			}
			if args[0] != "--help" && code == 0 {
				t.Fatal("disabled mode succeeded")
			}
			if _, e := os.Stat(data); !os.IsNotExist(e) {
				t.Fatal("disabled/help wrote data")
			}
		})
	}
}
func TestRepairPreviewIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.md")
	source := []byte("# Sample\n\n* [X] T1.0 Meaning  Owner: owner  kind: agent stage: implement  acc: [visible result]\n")
	if e := os.WriteFile(path, source, 0600); e != nil {
		t.Fatal(e)
	}
	data := filepath.Join(t.TempDir(), "absent")
	var out, errs bytes.Buffer
	if code := runRepair([]string{"--data", data, path}, &out, &errs); code != 0 {
		t.Fatalf("code%d: %s", code, errs.String())
	}
	got, e := os.ReadFile(path)
	if e != nil || !bytes.Equal(got, source) {
		t.Fatal("preview changed source")
	}
	if _, e = os.Stat(data); !os.IsNotExist(e) {
		t.Fatal("preview created candidate store")
	}
	if !strings.Contains(out.String(), "+- [x]") {
		t.Fatal("repair diff missing", out.String())
	}
}
func TestDiffPreservesCRLFAndNoNewline(t *testing.T) {
	var out bytes.Buffer
	printRepairDiff(&out, []byte("* [X] T1"), []byte("- [x] T1"))
	if !strings.Contains(out.String(), "\\ No newline at end of file") {
		t.Fatal(out.String())
	}
}

func TestRepairSubprocessSaveApplyAndStaleRefusal(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "plan.md")
	original := []byte("# Sample\n* [X] T1.0 Do work  Owner: owner  kind: agent stage: implement  acc: [observable result]\n")
	if e := os.WriteFile(sourcePath, original, 0600); e != nil {
		t.Fatal(e)
	}
	data := filepath.Join(t.TempDir(), "private")
	run := func(args ...string) ([]byte, error) {
		t.Helper()
		cmd := exec.Command(testCLI, append([]string{"repair", "--data", data}, args...)...)
		cmd.Env = append(os.Environ(), "EXPLABS_API_KEY=PRIVATE_SENTINEL_NO_CALL", "EXPLABS_BASE_URL=http://127.0.0.1:1", "EXPLABS_MODEL=PRIVATE_MODEL_SENTINEL")
		return cmd.CombinedOutput()
	}
	output, e := run("--save-candidate", sourcePath)
	if e != nil {
		t.Fatalf("save: %v %s", e, output)
	}
	var manifest struct {
		ID string `json:"id"`
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "{") {
			if e = json.Unmarshal([]byte(line), &manifest); e != nil {
				t.Fatal(e)
			}
		}
	}
	if manifest.ID == "" {
		t.Fatal("missing candidate", string(output))
	}
	if strings.Contains(string(output), "PRIVATE_") {
		t.Fatal("configuration leaked")
	}
	if e = os.WriteFile(sourcePath, []byte("owner concurrent change\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if out, e := run("--apply-candidate", manifest.ID); e == nil {
		t.Fatal("stale apply succeeded", string(out))
	}
	if e = os.WriteFile(sourcePath, original, 0600); e != nil {
		t.Fatal(e)
	}
	output, e = run("--apply-candidate", manifest.ID)
	if e != nil {
		t.Fatalf("apply: %v %s", e, output)
	}
	got, e := os.ReadFile(sourcePath)
	if e != nil || !bytes.Contains(got, []byte("- [x] T1.0")) {
		t.Fatal("candidate not applied", string(got), e)
	}
	backup := strings.TrimSpace(strings.TrimPrefix(string(output), "Applied exact candidate. Original backup:"))
	old, e := os.ReadFile(backup)
	if e != nil || !bytes.Equal(old, original) {
		t.Fatal("backup mismatch", e)
	}
	if out, e := run("--ai", sourcePath); e == nil || strings.Contains(string(out), "PRIVATE_") {
		t.Fatal("AI should fail disabled privately", string(out))
	}
}
