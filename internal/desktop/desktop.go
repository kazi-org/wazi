// Package desktop implements the private host-to-launcher startup contract.
package desktop

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	Protocol   = "wazi-desktop/1"
	Version    = "0.1.0"
	ReadyLimit = 4096
)

type Ready struct {
	Protocol string `json:"protocol"`
	Version  string `json:"version"`
	Origin   string `json:"origin"`
	PID      int    `json:"pid"`
	Nonce    string `json:"nonce"`
}

// ValidateNonce rejects empty, oversized and control-bearing launcher tokens.
func ValidateNonce(n string) error {
	if len(n) < 16 || len(n) > 256 {
		return fmt.Errorf("desktop nonce must be 16 to 256 bytes")
	}
	for _, r := range n {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_-", r)) {
			return fmt.Errorf("desktop nonce contains invalid characters")
		}
	}
	return nil
}

func WriteReady(w io.Writer, addr net.Addr, nonce string, pid int) error {
	if err := ValidateNonce(nonce); err != nil {
		return err
	}
	tcp, ok := addr.(*net.TCPAddr)
	if !ok || !tcp.IP.IsLoopback() || tcp.Port < 1 || tcp.Port > 65535 {
		return fmt.Errorf("desktop listener is not a valid loopback TCP endpoint")
	}
	line, err := json.Marshal(Ready{Protocol: Protocol, Version: Version, Origin: fmt.Sprintf("http://127.0.0.1:%d", tcp.Port), PID: pid, Nonce: nonce})
	if err != nil {
		return err
	}
	if len(line)+1 > ReadyLimit {
		return fmt.Errorf("desktop readiness exceeds limit")
	}
	line = append(line, '\n')
	n, err := w.Write(line)
	if err != nil {
		return err
	}
	if n != len(line) {
		return io.ErrShortWrite
	}
	return nil
}

// ValidateResources checks all files required to serve the desktop before bind.
type Resources struct{ Root, Assets, Data, Node, Bridge string }

func (r Resources) Validate() error {
	for name, p := range map[string]string{"root": r.Root, "assets": r.Assets, "data": r.Data, "node": r.Node, "bridge": r.Bridge} {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("desktop %s path must be absolute", name)
		}
	}
	for name, p := range map[string]string{"root": r.Root, "assets": r.Assets} {
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			return fmt.Errorf("desktop %s directory unavailable", name)
		}
	}
	if err := executable(r.Node); err != nil {
		return fmt.Errorf("desktop node runtime unavailable")
	}
	if err := regular(r.Bridge); err != nil {
		return fmt.Errorf("desktop parser bridge unavailable")
	}
	bridgeDir := filepath.Dir(r.Bridge)
	if err := regular(filepath.Join(bridgeDir, "plans.mjs")); err != nil {
		return fmt.Errorf("desktop parser scanner unavailable")
	}
	if err := regular(filepath.Join(filepath.Dir(bridgeDir), "src", "plan-parser.mjs")); err != nil {
		return fmt.Errorf("desktop parser unavailable")
	}
	return nil
}
func regular(p string) error {
	st, err := os.Stat(p)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("not regular")
	}
	return nil
}
func executable(p string) error {
	st, err := os.Stat(p)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("not executable")
	}
	return nil
}

// ValidateOrigin mirrors the launcher's strict loopback origin requirements.
func ValidateOrigin(s string) error {
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid desktop origin")
	}
	if u.Port() == "" {
		return fmt.Errorf("desktop origin has no port")
	}
	var p int
	if _, err := fmt.Sscanf(u.Port(), "%d", &p); err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid desktop origin port")
	}
	return nil
}
