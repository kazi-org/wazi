package deep

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type receipt struct {
	Key       string    `json:"key"`
	Status    string    `json:"status"` // dispatching, completed, unknown, deleted
	Manifest  Manifest  `json:"manifest"`
	Answer    string    `json:"answer,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Service struct {
	dir               string
	max               int64
	engine            Engine
	validator         LineageValidator
	memoryPersistence bool
}

func New(c Config) (*Service, error) {
	if c.Dir == "" {
		return nil, errors.New("private cache directory is required")
	}
	if c.MaxBytes <= 0 {
		return nil, errors.New("positive cache capacity is required")
	}
	if err := os.MkdirAll(c.Dir, 0700); err != nil {
		return nil, fmt.Errorf("create private cache directory: %w", err)
	}
	if err := os.Chmod(c.Dir, 0700); err != nil {
		return nil, fmt.Errorf("secure private cache directory: %w", err)
	}
	info, err := os.Lstat(c.Dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("private cache path must be a real directory")
	}
	s := &Service{dir: c.Dir, max: c.MaxBytes, engine: c.Engine, validator: c.Validator, memoryPersistence: c.MemoryPersistenceQualified && c.Validator != nil}
	if err := s.sweepExpired(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func Key(m Manifest) (string, error) {
	if err := validate(m); err != nil {
		return "", err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func validate(m Manifest) error {
	if m.RepositoryID == "" || m.TaskRef == "" || m.Question == "" || m.PlanDigest == "" || m.CodeDigest == "" || m.PlanBody == "" || m.PromptVersion == "" || m.AnalyzerVersion == "" {
		return errors.New("manifest is missing repository, task, question, source bodies or versions")
	}
	if digest(m.PlanBody) != m.PlanDigest {
		return errors.New("plan body digest does not match manifest")
	}
	if len(m.Code) == 0 {
		return errors.New("selected code input is required")
	}
	var codeDigestInput strings.Builder
	inputBytes := len(m.PlanBody) + len(m.Question)
	for _, c := range m.Code {
		if c.Path == "" || path.IsAbs(c.Path) || strings.Contains(c.Path, "\\") || path.Clean(c.Path) != c.Path || c.Path == ".." || strings.HasPrefix(c.Path, "../") || c.Body == "" || c.SHA256 == "" || digest(c.Body) != c.SHA256 {
			return errors.New("code input is missing fields or has an invalid digest")
		}
		codeDigestInput.WriteString(c.Path)
		codeDigestInput.WriteByte(0)
		codeDigestInput.WriteString(c.SHA256)
		codeDigestInput.WriteByte('\n')
		inputBytes += len(c.Path) + len(c.Body)
	}
	if digest(codeDigestInput.String()) != m.CodeDigest {
		return errors.New("code manifest digest does not match inputs")
	}
	if m.Model != Model {
		return fmt.Errorf("unsupported model %q", m.Model)
	}
	if len(m.Settings) == 0 || !json.Valid(m.Settings) {
		return errors.New("settings must be valid JSON")
	}
	if _, err := parseSettings(m.Settings); err != nil {
		return err
	}
	switch m.ContextMode {
	case ContextPlanCode:
		if len(m.Context) > 0 {
			return errors.New("plan/code-only request cannot contain memory context")
		}
	case ContextMemory:
		if len(m.Context) == 0 {
			return errors.New("memory request must contain the exact displayed context")
		}
	default:
		return errors.New("context mode must be plan_code or memory")
	}
	if m.ContextMode == ContextMemory && (m.ContextScope.RepositoryID != m.RepositoryID || m.ContextScope.BrainID == "" || m.ContextScope.AudienceID == "" || m.ContextScope.ProjectID == "") {
		return errors.New("memory scope must bind the repository, brain, audience and project")
	}
	for _, c := range m.Context {
		refOK, entityOK := false, false
		for _, id := range m.ContextScope.ReferenceIDs {
			if id == c.OwnerRef {
				refOK = true
				break
			}
		}
		for _, id := range m.ContextScope.EntityIDs {
			if c.EntityID != "" && id == c.EntityID {
				entityOK = true
				break
			}
		}
		if c.OwnerRef == "" || c.Kind == "" || c.ExpiresAt.IsZero() || (!refOK && !entityOK) || c.BrainID != m.ContextScope.BrainID || c.AudienceID != m.ContextScope.AudienceID || c.ProjectID != m.ContextScope.ProjectID || c.ContentDigest == "" || c.Version == "" || c.Body == "" || digest(c.Body) != c.ContentDigest {
			return errors.New("context item is missing displayed content or exact scoped lineage")
		}
		inputBytes += len(c.Body)
	}
	if inputBytes > maxRequestBytes/2 {
		return errors.New("selected input exceeds the local prompt-size limit")
	}
	if m.ContextMode == ContextMemory {
		if len(m.ContextScope.ReferenceIDs) == 0 && len(m.ContextScope.EntityIDs) == 0 {
			return errors.New("memory scope must identify at least one mapped entity or reference")
		}
		refs, entities := map[string]bool{}, map[string]bool{}
		for _, ref := range m.ContextScope.ReferenceIDs {
			if ref == "" || refs[ref] {
				return errors.New("memory scope contains duplicate or empty references")
			}
			refs[ref] = true
		}
		for _, id := range m.ContextScope.EntityIDs {
			if id == "" || entities[id] {
				return errors.New("memory scope contains duplicate or empty entities")
			}
			entities[id] = true
		}
		for _, c := range m.Context {
			refOK, entityOK := refs[c.OwnerRef], entities[c.EntityID]
			if !refOK && !entityOK {
				return errors.New("displayed memory items do not match qualified scope")
			}
		}
	}
	return nil
}

func (s *Service) Analyze(ctx context.Context, m Manifest) (Result, error) {
	key, err := Key(m)
	if err != nil {
		return Result{}, err
	}
	ephemeral := m.ContextMode == ContextMemory && !s.memoryPersistence
	unlock, err := s.acquire(ctx, "key-"+key)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	old, err := s.read(key)
	if err == nil {
		switch old.Status {
		case "completed":
			result, err := s.visible(ctx, old)
			if err == nil {
				result.Cached = true
			}
			return result, err
		case "dispatching", "unknown", "ephemeral_completed":
			return Result{Key: key, Model: m.Model, Status: "unknown", Freshness: "unknown"}, ErrUnknownOutcome
		case "deleted":
			return Result{Key: key, Model: m.Model, Status: "deleted", Freshness: "deleted"}, ErrUnknownOutcome
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, err
	}
	if s.engine == nil {
		return Result{}, ErrProviderUnavailable
	}
	if m.ContextMode == ContextMemory && s.validator != nil {
		if _, err := s.checkLineage(ctx, m); err != nil {
			return Result{}, fmt.Errorf("validate memory lineage before dispatch: %w", err)
		}
	}
	releaseCapacity, err := s.reserveCapacity(ctx, key, m)
	if err != nil {
		return Result{}, err
	}
	defer releaseCapacity()
	stored := cloneManifest(m)
	if ephemeral {
		stored.PlanBody = ""
		stored.Code = nil
		for i := range stored.Context {
			stored.Context[i].Body = ""
		}
	}
	r := receipt{Key: key, Status: "dispatching", Manifest: stored, UpdatedAt: time.Now().UTC()}
	if err := s.write(r); err != nil {
		return Result{}, err
	}
	answer, callErr := s.engine.Complete(ctx, cloneManifest(m))
	if callErr != nil {
		r.Status = "unknown"
		r.Answer = ""
		r.UpdatedAt = time.Now().UTC()
		_ = s.write(r)
		if errors.Is(callErr, context.Canceled) || errors.Is(callErr, context.DeadlineExceeded) {
			callErr = &CancellationError{Err: callErr, OutcomeUnknown: true}
		}
		return Result{Key: key, Model: m.Model, Status: "unknown", Freshness: "unknown"}, fmt.Errorf("provider outcome uncertain; receipt retained: %w", callErr)
	}
	r.Status = "completed"
	r.Answer = answer
	r.CreatedAt = time.Now().UTC()
	r.UpdatedAt = r.CreatedAt
	if ephemeral {
		r.Status = "ephemeral_completed"
		r.Answer = ""
		if err := s.write(r); err != nil {
			return Result{}, err
		}
		return Result{Key: key, TaskRef: m.TaskRef, Answer: answer, Model: m.Model, CreatedAt: r.CreatedAt, Status: "ephemeral", Freshness: "session_only", Persistence: false}, nil
	}
	if err := s.write(r); err != nil {
		return Result{Key: key, Model: m.Model, Status: "unknown", Freshness: "unknown"}, fmt.Errorf("persist completed answer; outcome retained as dispatched: %w", err)
	}
	return s.visible(ctx, r)
}

func cloneManifest(m Manifest) Manifest {
	m.Settings = append(json.RawMessage(nil), m.Settings...)
	m.Code = append([]CodeInput(nil), m.Code...)
	m.Context = append([]ContextItem(nil), m.Context...)
	m.ContextScope.EntityIDs = append([]string(nil), m.ContextScope.EntityIDs...)
	m.ContextScope.ReferenceIDs = append([]string(nil), m.ContextScope.ReferenceIDs...)
	return m
}

func (s *Service) visible(ctx context.Context, r receipt) (Result, error) {
	if r.Status != "completed" {
		return Result{Key: r.Key, Model: r.Manifest.Model, Status: r.Status, Freshness: r.Status}, nil
	}
	if r.Manifest.ContextMode == ContextMemory {
		if s.validator == nil || !s.memoryPersistence {
			return Result{Key: r.Key, Model: r.Manifest.Model, Status: "hidden", Freshness: "owner_unavailable"}, ErrMemoryUnavailable
		}
		if _, err := s.checkLineage(ctx, r.Manifest); err != nil {
			if errors.Is(err, ErrMemoryUnavailable) {
				return Result{Key: r.Key, Model: r.Manifest.Model, Status: "hidden", Freshness: "owner_unavailable"}, err
			}
			if !errors.Is(err, ErrLineageInvalid) {
				return Result{Key: r.Key, Model: r.Manifest.Model, Status: "hidden", Freshness: "owner_unavailable"}, fmt.Errorf("memory lineage could not be revalidated: %w", err)
			}
			r.Answer = ""
			r.Manifest.PlanBody = ""
			r.Manifest.Code = nil
			r.Status = "invalidated"
			for i := range r.Manifest.Context {
				r.Manifest.Context[i].Body = ""
			}
			r.UpdatedAt = time.Now().UTC()
			_ = s.write(r)
			return Result{Key: r.Key, Model: r.Manifest.Model, Status: "invalidated", Freshness: "invalidated"}, fmt.Errorf("memory-derived answer invalidated: %w", err)
		}
	}
	return Result{Key: r.Key, TaskRef: r.Manifest.TaskRef, Answer: r.Answer, Model: r.Manifest.Model, CreatedAt: r.CreatedAt, Status: "completed", Freshness: "current", Persistence: true}, nil
}

func (s *Service) checkLineage(ctx context.Context, m Manifest) (LineageValidation, error) {
	if s.validator == nil {
		return LineageValidation{Available: false}, ErrMemoryUnavailable
	}
	v, e := s.validator.ValidateLineage(ctx, m.ContextScope, m.Context)
	if e != nil {
		if errors.Is(e, ErrLineageInvalid) {
			return v, e
		}
		return v, fmt.Errorf("%w: %v", ErrMemoryUnavailable, e)
	}
	if !v.Available {
		return v, ErrMemoryUnavailable
	}
	if !v.Valid {
		return v, ErrLineageInvalid
	}
	return v, nil
}

func (s *Service) inspect(ctx context.Context, key string) (Result, error) {
	if !validKey(key) {
		return Result{}, errors.New("invalid cache key")
	}
	r, e := s.read(key)
	if e != nil {
		return Result{}, e
	}
	return s.visible(ctx, r)
}
func (s *Service) delete(ctx context.Context, key string) error {
	if !validKey(key) {
		return errors.New("invalid cache key")
	}
	unlock, e := s.acquire(ctx, "key-"+key)
	if e != nil {
		return e
	}
	defer unlock()
	r, e := s.read(key)
	if e != nil {
		return e
	}
	r.Answer = ""
	r.Manifest.PlanBody = ""
	r.Manifest.Code = nil
	for i := range r.Manifest.Context {
		r.Manifest.Context[i].Body = ""
	}
	r.Status = "deleted"
	r.UpdatedAt = time.Now().UTC()
	return s.write(r)
}

// InspectForRepository displays a result only when the receipt belongs to the selected repository.
func (s *Service) InspectForRepository(ctx context.Context, key, repositoryID string) (Result, error) {
	if repositoryID == "" || !validKey(key) {
		return Result{}, errors.New("repository and valid cache key are required")
	}
	r, e := s.read(key)
	if e != nil {
		return Result{}, e
	}
	if r.Manifest.RepositoryID != repositoryID {
		return Result{}, ErrNotFound
	}
	return Result{Key: key, TaskRef: r.Manifest.TaskRef, Model: r.Manifest.Model, CreatedAt: r.CreatedAt, Status: r.Status, Freshness: "requires_current_manifest", Persistence: r.Status == "completed", Cached: true}, nil
}

func (s *Service) InspectForManifest(ctx context.Context, m Manifest) (Result, error) {
	key, e := Key(m)
	if e != nil {
		return Result{}, e
	}
	result, e := s.InspectForRepository(ctx, key, m.RepositoryID)
	if e != nil {
		return result, e
	}
	r, e := s.read(key)
	if e != nil {
		return Result{}, e
	}
	result, e = s.visible(ctx, r)
	if e == nil {
		result.Cached = true
	}
	return result, e
}

// DeleteForRepository retains a body-free receipt tombstone to prevent accidental resend.
func (s *Service) DeleteForRepository(ctx context.Context, key, repositoryID string) error {
	if repositoryID == "" || !validKey(key) {
		return errors.New("repository and valid cache key are required")
	}
	r, e := s.read(key)
	if e != nil {
		return e
	}
	if r.Manifest.RepositoryID != repositoryID {
		return ErrNotFound
	}
	return s.delete(ctx, key)
}

// List returns metadata only for one selected repository; answer bodies always use InspectForRepository.
func (s *Service) List(ctx context.Context, repositoryID string) ([]Result, error) {
	if repositoryID == "" {
		return nil, errors.New("repository ID is required")
	}
	entries, e := os.ReadDir(s.dir)
	if e != nil {
		return nil, e
	}
	out := make([]Result, 0)
	for _, ent := range entries {
		if !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		key := strings.TrimSuffix(ent.Name(), ".json")
		if !validKey(key) {
			continue
		}
		r, e := s.read(key)
		if e != nil {
			continue
		}
		if r.Manifest.RepositoryID != repositoryID {
			continue
		}
		fresh := r.Status
		if r.Manifest.ContextMode == ContextMemory {
			fresh = "revalidate_before_display"
		}
		out = append(out, Result{Key: key, TaskRef: r.Manifest.TaskRef, Model: r.Manifest.Model, CreatedAt: r.CreatedAt, Status: r.Status, Freshness: fresh, Persistence: r.Status == "completed"})
	}
	return out, nil
}

// Regenerate authorizes a new paid dispatch after showing the caller the prior
// status and estimated model/cost disclosure. It never retries automatically.
func (s *Service) Regenerate(ctx context.Context, m Manifest, costDisclosure CostDisclosure) (Result, error) {
	if strings.TrimSpace(costDisclosure.Summary) == "" || !costDisclosure.Acknowledged {
		return Result{}, errors.New("explicit cost disclosure acknowledgement is required")
	}
	if s.engine == nil {
		return Result{}, ErrProviderUnavailable
	}
	key, e := Key(m)
	if e != nil {
		return Result{}, e
	}
	unlock, e := s.acquire(ctx, "key-"+key)
	if e != nil {
		return Result{}, e
	}
	defer unlock()
	if m.ContextMode == ContextMemory {
		if !s.memoryPersistence {
			return Result{}, ErrMemoryUnavailable
		}
		if _, e := s.checkLineage(ctx, m); e != nil {
			return Result{}, e
		}
	}
	releaseCapacity, e := s.reserveCapacity(ctx, key, m)
	if e != nil {
		return Result{}, e
	}
	defer releaseCapacity()
	r := receipt{Key: key, Status: "dispatching", Manifest: m, UpdatedAt: time.Now().UTC()}
	if e = s.write(r); e != nil {
		return Result{}, e
	}
	ans, e := s.engine.Complete(ctx, m)
	if e != nil {
		r.Status = "unknown"
		r.UpdatedAt = time.Now().UTC()
		_ = s.write(r)
		return Result{Key: key, Model: m.Model, Status: "unknown"}, fmt.Errorf("regeneration outcome uncertain: %w", e)
	}
	r.Status = "completed"
	r.Answer = ans
	r.CreatedAt = time.Now().UTC()
	r.UpdatedAt = r.CreatedAt
	if e = s.write(r); e != nil {
		return Result{Key: key, Model: m.Model, Status: "unknown"}, e
	}
	return s.visible(ctx, r)
}

func (s *Service) sweepExpired(ctx context.Context) error {
	entries, e := os.ReadDir(s.dir)
	if e != nil {
		return e
	}
	for _, ent := range entries {
		if !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		k := strings.TrimSuffix(ent.Name(), ".json")
		if !validKey(k) {
			continue
		}
		r, e := s.read(k)
		if e != nil {
			continue
		}
		if r.Status != "completed" || r.Manifest.ContextMode != ContextMemory {
			continue
		}
		expired := false
		for _, c := range r.Manifest.Context {
			if !c.ExpiresAt.IsZero() && !time.Now().Before(c.ExpiresAt) {
				expired = true
			}
		}
		if expired {
			r.Answer = ""
			r.Status = "deleted"
			r.Manifest.PlanBody = ""
			r.Manifest.Code = nil
			for i := range r.Manifest.Context {
				r.Manifest.Context[i].Body = ""
			}
			r.UpdatedAt = time.Now().UTC()
			if e = s.write(r); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Service) filename(k string) string { return filepath.Join(s.dir, k+".json") }
func (s *Service) read(k string) (receipt, error) {
	f, e := os.Open(s.filename(k))
	if e != nil {
		return receipt{}, e
	}
	defer f.Close()
	var r receipt
	e = json.NewDecoder(io.LimitReader(f, 32<<20)).Decode(&r)
	if e != nil {
		return r, e
	}
	return r, nil
}
func (s *Service) write(r receipt) error {
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(s.dir, ".write-")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e != nil {
		f.Close()
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Rename(name, s.filename(r.Key)); e != nil {
		return e
	}
	d, e := os.Open(s.dir)
	if e == nil {
		e = d.Sync()
		_ = d.Close()
	}
	return e
}
func (s *Service) reserveCapacity(ctx context.Context, key string, m Manifest) (func(), error) {
	unlock, e := s.acquire(ctx, "capacity")
	if e != nil {
		return nil, e
	}
	entries, e := os.ReadDir(s.dir)
	if e != nil {
		unlock()
		return nil, e
	}
	var used int64
	for _, x := range entries {
		if strings.HasSuffix(x.Name(), ".json") && x.Name() != key+".json" {
			info, e := x.Info()
			if e == nil {
				used += info.Size()
			}
		}
	}
	b, _ := json.Marshal(receipt{Key: key, Status: "dispatching", Manifest: m, UpdatedAt: time.Now()})
	if used+int64(len(b))+maxResponseBytes+4096 > s.max {
		unlock()
		return nil, ErrCapacity
	}
	return unlock, nil
}
func (s *Service) acquire(ctx context.Context, n string) (func(), error) {
	p := filepath.Join(s.dir, "."+n+".lock")
	for {
		f, e := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
		if e != nil {
			return nil, e
		}
		if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e == nil {
			return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
		}
		_ = f.Close()
		if e != syscall.EWOULDBLOCK && e != syscall.EAGAIN {
			return nil, e
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}
func validKey(k string) bool {
	if len(k) != 64 {
		return false
	}
	_, e := hex.DecodeString(k)
	return e == nil
}
