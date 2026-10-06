package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultCLIHelpWithoutArguments(t *testing.T) {
	workDir := t.TempDir()
	homeDir := filepath.Join(workDir, "home")
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "default help",
			want: []string{"wazi repair [--ai] [--env-file FILE] [--save-candidate] [--data DIR] FILE.md", "--save-candidate", "--apply-candidate", "wazi [host flags]", "--assets DIR", "developer host", "No browser, desktop app, or provider starts"},
		},
		{
			name: "repair help",
			args: []string{"repair", "--help"},
			want: []string{"Usage: wazi repair", "--save-candidate", "--apply-candidate"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(testCLI, tc.args...)
			cmd.Dir = workDir
			cmd.Env = []string{"HOME=" + homeDir, "PATH="}
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("CLI exit: %v; output: %s", err, output)
			}
			for _, want := range tc.want {
				if !strings.Contains(string(output), want) {
					t.Errorf("output missing %q: %s", want, output)
				}
			}
			if strings.Contains(string(output), "frontend assets missing") {
				t.Errorf("help invocation fell through to host startup: %s", output)
			}
			entries, err := os.ReadDir(workDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Errorf("help invocation created files in unrelated cwd: %v", entries)
			}
		})
	}
}
