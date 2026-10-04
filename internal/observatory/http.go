package observatory

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Host struct {
	Service *Service
	Assets  http.Handler
	token   string
	Extra   func(http.ResponseWriter, *http.Request) bool
}

func NewHost(s *Service, assets http.Handler, extra func(http.ResponseWriter, *http.Request) bool) (*Host, error) {
	t, e := RandomCapability()
	if e != nil {
		return nil, e
	}
	return &Host{Service: s, Assets: assets, token: t, Extra: extra}, nil
}
func (h *Host) Handler() http.Handler { return http.HandlerFunc(h.serve) }
func (h *Host) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !loopbackHost(r.Host) {
		http.Error(w, "loopback host required", http.StatusForbidden)
		return
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "cross-site request rejected", http.StatusForbidden)
		return
	}
	if o := r.Header.Get("Origin"); o != "" {
		u, e := url.Parse(o)
		if e != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
			http.Error(w, "origin rejected", http.StatusForbidden)
			return
		}
	}
	if r.URL.Path == "/api/session" && r.Method == http.MethodGet {
		writeJSON(w, map[string]string{"token": h.token})
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != "GET" && r.Method != "HEAD" {
		got := r.Header.Get("X-Wazi-Session")
		if len(got) != len(h.token) || subtle.ConstantTimeCompare([]byte(got), []byte(h.token)) != 1 {
			http.Error(w, "session capability required", http.StatusForbidden)
			return
		}
	}
	switch r.URL.Path {
	case "/api/plans":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", 405)
			return
		}
		scan, e := h.Service.Discover(r.Context())
		if e != nil {
			http.Error(w, "local plan discovery failed", 500)
			return
		}
		writeJSON(w, map[string]any{"projects": scan.RawProjects(), "warnings": scan.Warnings, "scannedAt": scan.ScannedAt})
		return
	case "/api/project":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			ProjectID string `json:"projectId"`
		}
		if !decode(w, r, &req) {
			return
		}
		snap, e := h.Service.Snapshot(r.Context(), req.ProjectID)
		if e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		writeJSON(w, snap)
		return
	case "/api/file":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			ProjectID      string `json:"projectId"`
			Target         string `json:"target"`
			SnapshotDigest string `json:"snapshotDigest"`
		}
		if !decode(w, r, &req) {
			return
		}
		snap, e := h.Service.Snapshot(r.Context(), req.ProjectID)
		if e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		if req.SnapshotDigest == "" || req.SnapshotDigest != snap.SnapshotDigest {
			http.Error(w, "selected source snapshot changed; refresh before opening this file", 409)
			return
		}
		detail, e := h.Service.File(snap, req.Target)
		if e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		writeJSON(w, detail)
		return
	case "/api/bindings":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			ProjectID     string `json:"projectId"`
			PlanPath      string `json:"planPath"`
			TaskID        string `json:"taskId"`
			Target        string `json:"target"`
			Kind          string `json:"kind"`
			BasisDigest   string `json:"basisDigest"`
			SidecarDigest string `json:"sidecarDigest"`
			Action        string `json:"action"`
		}
		if !decode(w, r, &req) {
			return
		}
		snap, e := h.Service.WriteBinding(req.ProjectID, req.PlanPath, req.TaskID, req.Target, req.Kind, req.BasisDigest, req.SidecarDigest, req.Action)
		if e != nil {
			status := 400
			if strings.Contains(e.Error(), "changed") {
				status = 409
			}
			http.Error(w, e.Error(), status)
			return
		}
		writeJSON(w, snap)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		if h.Extra != nil && h.Extra(w, r) {
			return
		}
		http.NotFound(w, r)
		return
	}
	if h.Assets == nil {
		http.NotFound(w, r)
		return
	}
	h.Assets.ServeHTTP(w, r)
}
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if e := dec.Decode(dst); e != nil {
		http.Error(w, "invalid request body", 400)
		return false
	}
	var extra any
	if e := dec.Decode(&extra); e != io.EOF {
		http.Error(w, "request must contain one JSON value", 400)
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
func loopbackHost(host string) bool {
	h, _, e := net.SplitHostPort(host)
	if e != nil {
		h = host
	}
	ip := net.ParseIP(strings.Trim(h, "[]"))
	return ip != nil && ip.IsLoopback() || strings.EqualFold(h, "localhost")
}
