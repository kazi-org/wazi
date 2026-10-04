package observatory

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID          string `json:"id"`
	SourceID    string `json:"sourceId"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	CanonicalID string `json:"canonicalId"`
	Line        int    `json:"line"`
	SourceBlock string `json:"sourceBlock"`
}
type Plan struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	SourceDigest string `json:"sourceDigest"`
	Tasks        []Task `json:"tasks"`
}
type planVersion struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

func planManifestDigest(plans []Plan, root string) (string, error) {
	manifest := make([]planVersion, 0, len(plans))
	for _, plan := range plans {
		b, err := safeRead(root, plan.Path, 1<<20)
		if err != nil {
			return "", err
		}
		actual := digest(b)
		if plan.SourceDigest == "" || actual != plan.SourceDigest {
			return "", fmt.Errorf("plan %s changed after its task graph was parsed", plan.Path)
		}
		manifest = append(manifest, planVersion{Path: filepath.ToSlash(plan.Path), Digest: actual})
	}
	b, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return digest(b), nil
}

type Project struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hosted bool   `json:"hosted"`
	Plans  []Plan `json:"plans"`
}
type Scan struct {
	Projects    []Project `json:"projects"`
	Warnings    []string  `json:"warnings"`
	ScannedAt   string    `json:"scannedAt"`
	rawProjects json.RawMessage
}

func (s Scan) RawProjects() any {
	var rows []map[string]any
	if json.Unmarshal(s.rawProjects, &rows) != nil {
		return s.Projects
	}
	byID := map[string]bool{}
	for _, p := range s.Projects {
		byID[p.ID] = p.Hosted
	}
	for _, row := range rows {
		id, _ := row["id"].(string)
		row["hosted"] = byID[id]
	}
	return rows
}

type Binding struct {
	PlanPath        string `json:"planPath"`
	TaskID          string `json:"taskId"`
	Target          string `json:"target"`
	Kind            string `json:"kind"`
	BasisDigest     string `json:"basisDigest"`
	PlanDigest      string `json:"planDigest"`
	TargetDigest    string `json:"targetDigest"`
	AnalyzerVersion string `json:"analyzerVersion"`
	Freshness       string `json:"freshness"`
	ConfirmedAt     string `json:"confirmedAt,omitempty"`
}

const AnalyzerVersion = "local-path-suggestions-v1"

func stableTaskID(task Task) bool {
	return task.CanonicalID != "" || (task.SourceID != "" && !strings.HasPrefix(task.SourceID, "line-"))
}

type Sidecar struct {
	Version      int       `json:"version"`
	RepositoryID string    `json:"repositoryId"`
	Bindings     []Binding `json:"bindings"`
	Dismissed    []Binding `json:"dismissed"`
}
type Suggestion struct {
	PlanPath    string `json:"planPath"`
	TaskID      string `json:"taskId"`
	Target      string `json:"target"`
	Kind        string `json:"kind"`
	Reason      string `json:"reason"`
	BasisDigest string `json:"basisDigest"`
}
type File struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type Snapshot struct {
	ProjectID      string       `json:"projectId"`
	RepositoryID   string       `json:"repositoryId"`
	PlanDigest     string       `json:"planDigest"`
	SnapshotDigest string       `json:"snapshotDigest"`
	Files          []File       `json:"files"`
	Suggestions    []Suggestion `json:"suggestions"`
	Bindings       []Binding    `json:"bindings"`
	SidecarDigest  string       `json:"sidecarDigest"`
	Warnings       []string     `json:"warnings"`
}
type Source struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Body   string `json:"body"`
	SHA256 string `json:"sha256"`
}
type CoherentSnapshot struct {
	Snapshot Snapshot `json:"snapshot"`
	Plan     Plan     `json:"plan"`
	Sources  []Source `json:"sources"`
	RawPlan  string   `json:"rawPlan"`
}
type FileDetail struct {
	Path      string    `json:"path"`
	Kind      string    `json:"kind"`
	SHA256    string    `json:"sha256"`
	Body      string    `json:"body"`
	Truncated bool      `json:"truncated"`
	Bindings  []Binding `json:"bindings"`
}
type Service struct {
	Root, DataDir, Node, Bridge string
	MaxFiles                    int
	MaxBytes                    int64
	mu                          sync.Mutex
	projects                    map[string]string
	repositories                map[string]string
	scan                        Scan
}

func New(root, dataDir, node, bridge string) *Service {
	return &Service{Root: root, DataDir: dataDir, Node: node, Bridge: bridge, MaxFiles: 2500, MaxBytes: 32 << 20, projects: map[string]string{}, repositories: map[string]string{}}
}
func digest(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }
func validRepositoryID(id string) bool {
	if len(id) != len("repo-")+24 || !strings.HasPrefix(id, "repo-") {
		return false
	}
	for _, c := range id[len("repo-"):] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.Len()
	if remaining > 0 {
		if remaining > n {
			remaining = n
		}
		_, _ = b.Buffer.Write(p[:remaining])
	}
	if remaining < n {
		b.exceeded = true
	}
	return n, nil
}
func (s *Service) Discover(ctx context.Context) (Scan, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Node, s.Bridge, s.Root)
	var out, stderr boundedBuffer
	out.limit, stderr.limit = 16<<20, 4096
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Scan{}, fmt.Errorf("plan scanner: %w: %s", err, stderr.String())
	}
	if out.exceeded {
		return Scan{}, errors.New("plan scanner output exceeded the 16 MiB bound")
	}
	var result struct {
		Scan struct {
			Projects  json.RawMessage `json:"projects"`
			Warnings  []string        `json:"warnings"`
			ScannedAt string          `json:"scannedAt"`
		} `json:"scan"`
		Roots map[string]string `json:"roots"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return Scan{}, err
	}
	got := Scan{Warnings: result.Scan.Warnings, ScannedAt: result.Scan.ScannedAt, rawProjects: result.Scan.Projects}
	if err := json.Unmarshal(got.rawProjects, &got.Projects); err != nil {
		return Scan{}, err
	}
	for i := range got.Projects {
		got.Projects[i].Hosted = false
		for j := range got.Projects[i].Plans {
			for k := range got.Projects[i].Plans[j].Tasks {
				task := &got.Projects[i].Plans[j].Tasks[k]
				task.ID = task.SourceID
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	registryPath := filepath.Join(s.DataDir, "repositories.json")
	if err := os.MkdirAll(s.DataDir, 0700); err != nil {
		return Scan{}, err
	}
	if fi, err := os.Lstat(s.DataDir); err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return Scan{}, errors.New("private data path must be a real directory")
	}
	if err := os.Chmod(s.DataDir, 0700); err != nil {
		return Scan{}, fmt.Errorf("secure private data directory: %w", err)
	}
	unlock, err := lockFile(filepath.Join(s.DataDir, ".repositories.lock"))
	if err != nil {
		return Scan{}, err
	}
	defer unlock()
	registry := map[string]string{}
	if b, e := os.ReadFile(registryPath); e == nil {
		if e = json.Unmarshal(b, &registry); e != nil || registry == nil {
			return Scan{}, errors.New("private repository identity registry is corrupt; refusing to replace identities")
		}
		for locator, id := range registry {
			if locator == "" || !validRepositoryID(id) {
				return Scan{}, errors.New("private repository identity registry has an invalid entry")
			}
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return Scan{}, e
	}
	s.scan = got
	s.projects = map[string]string{}
	s.repositories = map[string]string{}
	changed := false
	for _, p := range got.Projects {
		candidate := result.Roots[p.ID]
		abs, e := filepath.Abs(candidate)
		if e != nil || candidate == "" {
			got.Warnings = append(got.Warnings, "A discovered project root could not be resolved.")
			continue
		}
		rel, e := filepath.Rel(s.Root, abs)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		s.projects[p.ID] = abs
		for i := range got.Projects {
			if got.Projects[i].ID == p.ID {
				got.Projects[i].Hosted = true
			}
		}
		locator, identityErr := identityLocator(abs)
		if identityErr != nil {
			return Scan{}, identityErr
		}
		rid := registry[locator]
		if rid == "" {
			id, e := RandomCapability()
			if e != nil {
				return Scan{}, e
			}
			rid = "repo-" + id[:24]
			registry[locator] = rid
			changed = true
		}
		s.repositories[p.ID] = rid
	}
	if changed {
		if e := os.MkdirAll(s.DataDir, 0700); e != nil {
			return Scan{}, e
		}
		b, _ := json.MarshalIndent(registry, "", "  ")
		tmp, e := os.CreateTemp(s.DataDir, ".repositories-*.tmp")
		if e != nil {
			return Scan{}, e
		}
		name := tmp.Name()
		if _, e = tmp.Write(b); e == nil {
			e = tmp.Sync()
		}
		ce := tmp.Close()
		if e == nil {
			e = ce
		}
		if e == nil {
			e = os.Chmod(name, 0600)
		}
		if e == nil {
			e = os.Rename(name, registryPath)
		}
		_ = os.Remove(name)
		if e != nil {
			return Scan{}, e
		}
	}
	return got, nil
}
func (s *Service) project(id string) (string, *Project, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	root := s.projects[id]
	if root == "" {
		return "", nil, "", errors.New("unknown project; refresh discovery")
	}
	abs, e := filepath.Abs(root)
	if e != nil {
		return "", nil, "", e
	}
	rel, e := filepath.Rel(s.Root, abs)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", nil, "", errors.New("project escaped discovery root")
	}
	for i := range s.scan.Projects {
		if s.scan.Projects[i].ID == id {
			p := s.scan.Projects[i]
			return abs, &p, s.repositories[id], nil
		}
	}
	return "", nil, "", errors.New("project unavailable")
}
func identityLocator(root string) (string, error) {
	marker := filepath.Join(root, ".git")
	st, err := os.Lstat(marker)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			canonical, e := filepath.EvalSymlinks(root)
			if e != nil {
				return "", e
			}
			return "root:" + canonical, nil
		}
		return "", fmt.Errorf("inspect project Git metadata: %w", err)
	}
	gitDir := marker
	if st.Mode().IsRegular() {
		b, e := os.ReadFile(marker)
		if e != nil {
			return "", fmt.Errorf("read project Git metadata: %w", e)
		}
		line := strings.TrimSpace(string(b))
		if !strings.HasPrefix(line, "gitdir: ") {
			return "", errors.New("project .git pointer is malformed")
		}
		gitDir = strings.TrimSpace(strings.TrimPrefix(line, "gitdir: "))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(root, gitDir)
		}
	}
	if st.IsDir() {
		gitDir = marker
	}
	if common, e := os.ReadFile(filepath.Join(gitDir, "commondir")); e == nil {
		v := strings.TrimSpace(string(common))
		if !filepath.IsAbs(v) {
			v = filepath.Join(gitDir, v)
		}
		gitDir = v
	} else if !errors.Is(e, os.ErrNotExist) {
		return "", fmt.Errorf("read Git common directory: %w", e)
	}
	common, e := filepath.EvalSymlinks(gitDir)
	if e != nil {
		return "", fmt.Errorf("resolve Git common directory: %w", e)
	}
	return "git:" + common, nil
}
func (s *Service) Snapshot(ctx context.Context, id string) (Snapshot, error) {
	if _, err := s.Discover(ctx); err != nil {
		return Snapshot{}, err
	}
	root, p, repoID, err := s.project(id)
	if err != nil {
		return Snapshot{}, err
	}
	snap := Snapshot{ProjectID: id, RepositoryID: repoID, Warnings: []string{}}
	planDigest, err := planManifestDigest(p.Plans, root)
	if err != nil {
		return Snapshot{}, err
	}
	snap.PlanDigest = planDigest
	files, warns := s.scanFiles(root)
	snap.Files = files
	snap.Warnings = append(snap.Warnings, warns...)
	snap.Suggestions = s.suggest(*p, files, snap.PlanDigest)
	side, sd, e := s.loadSidecar(root, repoID)
	if e != nil {
		return Snapshot{}, e
	}
	if len(side.Dismissed) > 0 {
		remaining := snap.Suggestions[:0]
		for _, suggestion := range snap.Suggestions {
			dismissed := false
			for _, item := range side.Dismissed {
				if item.PlanPath == suggestion.PlanPath && item.TaskID == suggestion.TaskID && item.Target == suggestion.Target && item.Kind == suggestion.Kind && item.BasisDigest == suggestion.BasisDigest {
					dismissed = true
					break
				}
			}
			if !dismissed {
				remaining = append(remaining, suggestion)
			}
		}
		snap.Suggestions = remaining
	}
	snap.Bindings = side.Bindings
	snap.SidecarDigest = sd
	for i := range snap.Bindings {
		b := &snap.Bindings[i]
		b.Freshness = "missing"
		for _, f := range files {
			if f.Path == b.Target {
				b.Freshness = "stale"
				if b.PlanDigest == snap.PlanDigest && b.TargetDigest == f.SHA256 && (b.AnalyzerVersion == AnalyzerVersion || b.AnalyzerVersion == "manual-link-v1") {
					b.Freshness = "current"
				}
				break
			}
		}
	}
	b, _ := json.Marshal(struct {
		R string
		P string
		F []File
		B []Binding
	}{snap.RepositoryID, snap.PlanDigest, files, side.Bindings})
	snap.SnapshotDigest = digest(b)
	return snap, nil
}
func safeRead(root, rel string, limit int64) ([]byte, error) {
	canonicalRoot, e := filepath.EvalSymlinks(root)
	if e != nil {
		return nil, e
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, errors.New("unsafe relative path")
	}
	full := filepath.Join(root, clean)
	resolved, e := filepath.EvalSymlinks(full)
	if e != nil {
		return nil, e
	}
	inside, e := filepath.Rel(canonicalRoot, resolved)
	if e != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return nil, errors.New("symlink escaped project")
	}
	fi, e := os.Stat(resolved)
	if e != nil || !fi.Mode().IsRegular() || fi.Size() > limit {
		return nil, errors.New("file unavailable or exceeds bound")
	}
	return os.ReadFile(resolved)
}
func (s *Service) scanFiles(root string) ([]File, []string) {
	files := []File{}
	warnings := []string{}
	var totalBytes int64
	capped := false
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			warnings = append(warnings, "Some source paths could not be inspected.")
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() && rel != "." && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" || d.Name() == ".wazi" || d.Name() == "dist" || d.Name() == "build" || d.Name() == "coverage" || d.Name() == "secrets" || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || sensitivePath(rel) {
			return nil
		}
		if len(files) >= s.MaxFiles {
			capped = true
			return filepath.SkipAll
		}
		ext := strings.ToLower(filepath.Ext(path))
		kind := ""
		if strings.Contains(strings.ToLower(filepath.Base(path)), "test") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".spec.ts") || strings.HasSuffix(path, ".test.ts") {
			kind = "test"
		} else if ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".py" || ext == ".rs" || ext == ".swift" || ext == ".java" || ext == ".c" || ext == ".h" {
			kind = "code"
		}
		if kind == "" {
			return nil
		}
		info, e := d.Info()
		if e != nil || info.Size() > 128<<10 {
			return nil
		}
		if totalBytes+info.Size() > s.MaxBytes {
			capped = true
			return filepath.SkipAll
		}
		b, e := os.ReadFile(path)
		if e != nil || bytes.IndexByte(b, 0) >= 0 {
			return nil
		}
		files = append(files, File{Path: filepath.ToSlash(rel), Kind: kind, Size: info.Size(), SHA256: digest(b)})
		totalBytes += info.Size()
		return nil
	})
	if err != nil || capped {
		warnings = append(warnings, "Source scan reached its configured file bound.")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, warnings
}
func sensitivePath(rel string) bool {
	name := strings.ToLower(filepath.Base(rel))
	if name == ".env" || strings.HasPrefix(name, ".env.") || strings.Contains(name, "secret") || strings.Contains(name, "credential") || strings.Contains(name, "token") || strings.Contains(name, "private") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".pem" || ext == ".key" || ext == ".p12" || ext == ".pfx"
}
func (s *Service) suggest(p Project, files []File, pd string) []Suggestion {
	out := []Suggestion{}
	for _, pl := range p.Plans {
		for _, t := range pl.Tasks {
			if !stableTaskID(t) {
				continue
			}
			words := strings.Fields(strings.ToLower(t.Title))
			for _, f := range files {
				score := 0
				base := strings.ToLower(filepath.Base(f.Path))
				for _, w := range words {
					w = strings.Trim(w, "`*_()/.,:;-")
					if len(w) > 3 && strings.Contains(base, w) {
						score++
					}
				}
				explicit := strings.Contains(strings.ToLower(t.SourceBlock), strings.ToLower(f.Path)) || strings.Contains(strings.ToLower(t.SourceBlock), strings.ToLower(filepath.Base(f.Path)))
				if score == 0 && !explicit {
					continue
				}
				reason := fmt.Sprintf("Path name matches %d task-title term(s); review before confirming.", score)
				if explicit {
					reason = "Task text explicitly mentions this project-relative path; review before confirming."
				}
				basis := digest([]byte(pd + "\x00" + f.Path + "\x00" + f.SHA256 + "\x00" + t.ID))
				out = append(out, Suggestion{pl.Path, t.ID, f.Path, f.Kind, reason, basis})
				if len(out) >= 300 {
					return out
				}
			}
		}
	}
	return out
}
func (s *Service) loadSidecar(root, id string) (Sidecar, string, error) {
	path := filepath.Join(root, ".wazi", "links.json")
	if fi, e := os.Lstat(path); e == nil && (fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular()) {
		return Sidecar{}, "", errors.New("binding sidecar must be a regular local file")
	}
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return Sidecar{Version: 1, RepositoryID: id, Bindings: []Binding{}, Dismissed: []Binding{}}, digest(nil), nil
	}
	if e != nil {
		return Sidecar{}, "", e
	}
	var x Sidecar
	if e = json.Unmarshal(b, &x); e != nil {
		return Sidecar{}, "", e
	}
	if x.Version != 1 || x.RepositoryID != id {
		return Sidecar{}, "", errors.New("unsupported sidecar version or repository identity")
	}
	return x, digest(b), nil
}
func (s *Service) WriteBinding(id, planPath, taskID, target, kind, basis, expected, action string) (Snapshot, error) {
	root, p, repoID, e := s.project(id)
	if e != nil {
		return Snapshot{}, e
	}
	current, e := s.Snapshot(context.Background(), id)
	if e != nil {
		return Snapshot{}, e
	}
	if action == "manual" {
		if basis != current.SnapshotDigest {
			return Snapshot{}, errors.New("manual link basis is stale")
		}
	} else {
		matched := false
		for _, candidate := range current.Suggestions {
			if candidate.PlanPath == planPath && candidate.TaskID == taskID && candidate.Target == target && candidate.Kind == kind && candidate.BasisDigest == basis {
				matched = true
				break
			}
		}
		if !matched {
			return Snapshot{}, errors.New("suggestion basis is stale or does not match this target")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(root, ".wazi")
	if e = os.MkdirAll(dir, 0700); e != nil {
		return Snapshot{}, e
	}
	if fi, err := os.Lstat(dir); err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return Snapshot{}, errors.New(".wazi must be a real project-local directory")
	}
	lockDir := filepath.Join(s.DataDir, "locks")
	if e = os.MkdirAll(lockDir, 0700); e != nil {
		return Snapshot{}, e
	}
	if fi, err := os.Lstat(lockDir); err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return Snapshot{}, errors.New("private lock path must be a real directory")
	}
	if e = os.Chmod(lockDir, 0700); e != nil {
		return Snapshot{}, e
	}
	unlock, e := lockFile(filepath.Join(lockDir, repoID+".bindings.lock"))
	if e != nil {
		return Snapshot{}, e
	}
	defer unlock()
	side, actual, e := s.loadSidecar(root, repoID)
	if e != nil {
		return Snapshot{}, e
	}
	if expected != actual {
		return Snapshot{}, errors.New("sidecar changed; refresh and review again")
	}
	cleanTarget := filepath.Clean(filepath.FromSlash(target))
	if target == "" || filepath.ToSlash(cleanTarget) != target {
		return Snapshot{}, errors.New("target must be a normalized project-relative path")
	}
	rel, e := filepath.Rel(root, filepath.Join(root, cleanTarget))
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(target) {
		return Snapshot{}, errors.New("target must stay inside selected project")
	}
	full := filepath.Join(root, rel)
	fi, e := os.Lstat(full)
	if e != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return Snapshot{}, errors.New("target must be an existing regular project file")
	}
	targetDigest := ""
	indexed := false
	for _, f := range current.Files {
		if f.Path == filepath.ToSlash(rel) && f.Kind == kind {
			targetDigest = f.SHA256
			indexed = true
			break
		}
	}
	if !indexed {
		return Snapshot{}, errors.New("target must be in the selected code/test snapshot with its analyzed kind")
	}
	currentTarget, e := safeRead(root, filepath.ToSlash(rel), 128<<10)
	if e != nil || digest(currentTarget) != targetDigest {
		return Snapshot{}, errors.New("target changed since analysis; refresh and review again")
	}
	planDigest, re := planManifestDigest(p.Plans, root)
	if re != nil || planDigest != current.PlanDigest {
		return Snapshot{}, errors.New("plan changed since analysis; refresh and review again")
	}
	stable := false
	for _, pl := range p.Plans {
		if pl.Path == planPath {
			count := 0
			for _, t := range pl.Tasks {
				if t.ID == taskID {
					count++
					if stableTaskID(t) {
						stable = true
					}
				}
			}
			if count != 1 {
				stable = false
			}
		}
	}
	if !stable {
		return Snapshot{}, errors.New("task lacks a stable authored ID or is ambiguous")
	}
	if action != "confirm" && action != "manual" && action != "dismiss" {
		return Snapshot{}, errors.New("invalid action")
	}
	planDigest = current.PlanDigest
	analyzer := AnalyzerVersion
	if action == "manual" {
		analyzer = "manual-link-v1"
	}
	binding := Binding{PlanPath: planPath, TaskID: taskID, Target: filepath.ToSlash(rel), Kind: kind, BasisDigest: basis, PlanDigest: planDigest, TargetDigest: targetDigest, AnalyzerVersion: analyzer}
	if action == "dismiss" {
		side.Dismissed = append(side.Dismissed, binding)
	} else {
		binding.ConfirmedAt = time.Now().UTC().Format(time.RFC3339Nano)
		side.Bindings = append(side.Bindings, binding)
	}
	data, _ := json.MarshalIndent(side, "", "  ")
	tmp, e := os.CreateTemp(dir, ".links-*.tmp")
	if e != nil {
		return Snapshot{}, e
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, e = tmp.Write(data); e == nil {
		e = tmp.Sync()
	}
	ce := tmp.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return Snapshot{}, e
	}
	if e = os.Chmod(name, 0600); e != nil {
		return Snapshot{}, e
	}
	latest, checkErr := os.ReadFile(filepath.Join(dir, "links.json"))
	latestDigest := digest(latest)
	if errors.Is(checkErr, os.ErrNotExist) {
		latestDigest = digest(nil)
	} else if checkErr != nil {
		return Snapshot{}, checkErr
	}
	if latestDigest != actual {
		return Snapshot{}, errors.New("sidecar changed during update; refresh and review again")
	}
	if e = os.Rename(name, filepath.Join(dir, "links.json")); e != nil {
		return Snapshot{}, e
	}
	s.mu.Unlock()
	result, e := s.Snapshot(context.Background(), id)
	s.mu.Lock()
	return result, e
}
func (s *Service) ResolveTask(snapshot Snapshot, planPath, taskID string) (CoherentSnapshot, error) {
	current, err := s.Snapshot(context.Background(), snapshot.ProjectID)
	if err != nil {
		return CoherentSnapshot{}, err
	}
	if current.SnapshotDigest != snapshot.SnapshotDigest {
		return CoherentSnapshot{}, errors.New("selected snapshot is stale; refresh before resolving task")
	}
	snapshot = current
	root, p, _, e := s.project(snapshot.ProjectID)
	if e != nil {
		return CoherentSnapshot{}, e
	}
	var plan *Plan
	var task *Task
	for i := range p.Plans {
		if p.Plans[i].Path == planPath {
			plan = &p.Plans[i]
			for j := range plan.Tasks {
				if plan.Tasks[j].ID == taskID {
					task = &plan.Tasks[j]
				}
			}
		}
	}
	if plan == nil || task == nil {
		return CoherentSnapshot{}, errors.New("task not found")
	}
	raw, e := safeRead(root, planPath, 1<<20)
	if e != nil {
		return CoherentSnapshot{}, e
	}
	if digest(raw) != plan.SourceDigest {
		return CoherentSnapshot{}, errors.New("plan changed after task resolution; refresh before using this task")
	}
	out := CoherentSnapshot{Snapshot: snapshot, Plan: *plan, RawPlan: string(raw), Sources: []Source{}}
	for _, b := range snapshot.Bindings {
		if b.PlanPath != planPath || b.TaskID != taskID {
			continue
		}
		detail, e := s.File(snapshot, b.Target)
		if e != nil {
			continue
		}
		out.Sources = append(out.Sources, Source{b.Target, b.Kind, detail.Body, detail.SHA256})
	}
	return out, nil
}
func (s *Service) File(snapshot Snapshot, target string) (FileDetail, error) {
	root, _, _, e := s.project(snapshot.ProjectID)
	if e != nil {
		return FileDetail{}, e
	}
	clean := filepath.Clean(filepath.FromSlash(target))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return FileDetail{}, errors.New("file path must be project relative")
	}
	rel := filepath.ToSlash(clean)
	var indexed *File
	for i := range snapshot.Files {
		if snapshot.Files[i].Path == rel {
			indexed = &snapshot.Files[i]
			break
		}
	}
	if indexed == nil {
		return FileDetail{}, errors.New("file is outside selected analysis snapshot")
	}
	b, e := safeRead(root, rel, 256<<10)
	if e != nil {
		return FileDetail{}, e
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return FileDetail{}, errors.New("binary files are not displayed")
	}
	if digest(b) != indexed.SHA256 {
		return FileDetail{}, errors.New("file changed since selected snapshot")
	}
	links := []Binding{}
	for _, x := range snapshot.Bindings {
		if x.Target == rel {
			links = append(links, x)
		}
	}
	return FileDetail{Path: rel, Kind: indexed.Kind, SHA256: digest(b), Body: string(b), Bindings: links}, nil
}
func RandomCapability() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
