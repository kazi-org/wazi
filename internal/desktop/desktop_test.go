package desktop

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReadyContract(t *testing.T) {
	var out bytes.Buffer
	if err := WriteReady(&out, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 49152}, "0123456789abcdef", 42); err != nil {
		t.Fatal(err)
	}
	if out.Len() > ReadyLimit || strings.Count(out.String(), "\n") != 1 || !strings.Contains(out.String(), `"protocol":"wazi-desktop/1"`) || !strings.Contains(out.String(), `"version":"0.1.0"`) {
		t.Fatalf("unexpected readiness record: %q", out.String())
	}
	if err := ValidateOrigin("http://127.0.0.1:49152"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"http://localhost:80", "https://127.0.0.1:80", "http://user@127.0.0.1:80", "http://127.0.0.1:80/path", "http://127.0.0.1:0", "http://127.0.0.1:65536"} {
		if ValidateOrigin(bad) == nil {
			t.Errorf("accepted origin %q", bad)
		}
	}
}

func TestResourcesRequirePackagedParserAndAbsolutePaths(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "root")
	assets := filepath.Join(d, "web")
	parser := filepath.Join(d, "parser")
	scripts := filepath.Join(parser, "scripts")
	src := filepath.Join(parser, "src")
	for _, p := range []string{root, assets, scripts, src} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{filepath.Join(scripts, "host-bridge.mjs"), filepath.Join(scripts, "plans.mjs"), filepath.Join(src, "plan-parser.mjs")} {
		if err := os.WriteFile(p, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}
	node := filepath.Join(d, "node")
	if err := os.WriteFile(node, []byte(""), 0700); err != nil {
		t.Fatal(err)
	}
	r := Resources{Root: root, Assets: assets, Data: filepath.Join(d, "data"), Node: node, Bridge: filepath.Join(scripts, "host-bridge.mjs")}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(src, "plan-parser.mjs")); err != nil {
		t.Fatal(err)
	}
	if r.Validate() == nil {
		t.Fatal("accepted incomplete parser closure")
	}
	r.Root = "relative"
	if r.Validate() == nil {
		t.Fatal("accepted relative root")
	}
}

func TestWriteReadyRejectsBadNonceAndNonLoopback(t *testing.T) {
	var out bytes.Buffer
	if WriteReady(&out, &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 80}, "0123456789abcdef", 1) == nil {
		t.Fatal("accepted wildcard listener")
	}
	if WriteReady(&out, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 80}, "bad", 1) == nil {
		t.Fatal("accepted weak nonce")
	}
}
