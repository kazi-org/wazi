package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testCLI string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "wazi-cli-test-")
	if err != nil {
		os.Exit(2)
	}
	testCLI = filepath.Join(dir, "wazi")
	build := exec.Command("go", "build", "-o", testCLI, ".")
	if output, err := build.CombinedOutput(); err != nil {
		_, _ = os.Stderr.Write(output)
		os.Exit(2)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

type launched struct {
	cmd                              *exec.Cmd
	stdin                            io.WriteCloser
	stdout                           *bufio.Reader
	stderr                           *strings.Builder
	root, assets, data, node, bridge string
}

func fixture(t *testing.T) *launched {
	t.Helper()
	d := t.TempDir()
	root := filepath.Join(d, "root")
	assets := filepath.Join(d, "web")
	scripts := filepath.Join(d, "parser", "scripts")
	src := filepath.Join(d, "parser", "src")
	for _, p := range []string{root, assets, scripts, src} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{filepath.Join(scripts, "host-bridge.mjs"), filepath.Join(scripts, "plans.mjs"), filepath.Join(src, "plan-parser.mjs")} {
		if err := os.WriteFile(p, []byte("// fixture\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	node := filepath.Join(d, "node")
	if err := os.WriteFile(node, []byte("#!/bin/sh\nprintf '%s\\n' '{\"scan\":{\"projects\":[],\"warnings\":[],\"scannedAt\":\"test\"},\"roots\":{}}'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return &launched{root: root, assets: assets, data: filepath.Join(d, "data"), node: node, bridge: filepath.Join(scripts, "host-bridge.mjs")}
}
func (f *launched) start(t *testing.T, watch bool) {
	f.startArgs(t, "-desktop-nonce", "0123456789abcdef0123456789abcdef")
}
func (f *launched) startArgs(t *testing.T, desktopArgs ...string) {
	t.Helper()
	args := []string{"-root", f.root, "-assets", f.assets, "-data", f.data, "-node", f.node, "-bridge", f.bridge, "-desktop-ready"}
	args = append(args, desktopArgs...)
	args = append(args, "-parent-watch")
	f.cmd = exec.Command(testCLI, args...)
	f.cmd.Dir = t.TempDir()
	f.cmd.Env = []string{"PATH=", "HOME=" + filepath.Dir(f.root)}
	stdin, err := f.cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	f.stdin = stdin
	out, err := f.cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	f.stdout = bufio.NewReader(io.LimitReader(out, 4097))
	f.stderr = &strings.Builder{}
	f.cmd.Stderr = f.stderr
	if err = f.cmd.Start(); err != nil {
		t.Fatal(err)
	}
}
func (f *launched) ready(t *testing.T) {
	t.Helper()
	line, err := f.stdout.ReadString('\n')
	if err != nil {
		t.Fatalf("readiness: %v; stderr=%s", err, f.stderr.String())
	}
	if len(line) > 4096 {
		t.Fatal("readiness exceeded contract bound")
	}
	var got struct {
		Protocol, Version, Origin string
		PID                       int
		Nonce                     string
	}
	if err = json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("invalid readiness %q: %v", line, err)
	}
	if got.Protocol != "wazi-desktop/1" || got.Version != "0.1.0" || got.PID != f.cmd.Process.Pid || got.Nonce != "0123456789abcdef0123456789abcdef" || !strings.HasPrefix(got.Origin, "http://127.0.0.1:") {
		t.Fatalf("unexpected readiness %+v", got)
	}
	resp, err := http.Get(got.Origin + "/api/session")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("session status %d", resp.StatusCode)
	}
	resp, err = http.Post(got.Origin+"/api/project", "application/json", strings.NewReader(`{"projectId":"missing"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("unauthenticated API status %d", resp.StatusCode)
	}
}
func waitExit(t *testing.T, c *exec.Cmd) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("host exit: %v", err)
		}
	case <-time.After(5 * time.Second):
		_ = c.Process.Kill()
		t.Fatal("host did not stop within bound")
	}
}
func TestDesktopHostReadinessEOFAndSignal(t *testing.T) {
	for _, mode := range []string{"eof", "signal"} {
		t.Run(mode, func(t *testing.T) {
			f := fixture(t)
			f.start(t, true)
			f.ready(t)
			if mode == "eof" {
				if err := f.stdin.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := f.cmd.Process.Signal(os.Interrupt); err != nil {
					t.Fatal(err)
				}
			}
			waitExit(t, f.cmd)
		})
	}
}
func TestDesktopRejectsMissingResourcesAndNonceBeforeReadiness(t *testing.T) {
	f := fixture(t)
	f.startArgs(t)
	err := f.cmd.Wait()
	if err == nil {
		t.Fatal("missing nonce accepted")
	}
	if f.stdout.Buffered() != 0 {
		t.Fatal("readiness emitted for invalid nonce")
	}
	f = fixture(t)
	_ = os.Remove(filepath.Join(filepath.Dir(f.bridge), "plans.mjs"))
	f.start(t, true)
	err = f.cmd.Wait()
	if err == nil {
		t.Fatal("missing parser resource accepted")
	}
	if f.stdout.Buffered() != 0 {
		t.Fatal("readiness emitted with missing resource")
	}
}
