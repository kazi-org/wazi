// Package deep implements explicit, privately cached Wazi analysis requests.
package deep

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const Model = "openai/gpt-6-luna"

var (
	ErrUnknownOutcome      = errors.New("prior provider outcome is unknown; explicit regeneration required")
	ErrInProgress          = errors.New("an identical request is already in progress")
	ErrCapacity            = errors.New("private analysis cache capacity would be exceeded")
	ErrMemoryUnavailable   = errors.New("memory-derived result cannot be validated while its owner seam is unavailable")
	ErrLineageInvalid      = errors.New("memory-derived result lineage is no longer eligible or current")
	ErrNotFound            = errors.New("analysis receipt not found")
	ErrProviderUnavailable = errors.New("OpenRouter is unavailable or not configured")
)

type ContextMode string

const (
	ContextPlanCode ContextMode = "plan_code"
	ContextMemory   ContextMode = "memory"
)

// ContextItem captures exactly one item the host displayed and the user selected.
// Body is sent to the provider only when included by the host in this manifest.
type ContextItem struct {
	OwnerRef      string    `json:"owner_ref"`
	Kind          string    `json:"kind"`
	EntityID      string    `json:"entity_id"`
	BrainID       string    `json:"brain_id"`
	AudienceID    string    `json:"audience_id"`
	ProjectID     string    `json:"project_id"`
	ContentDigest string    `json:"content_digest"`
	Version       string    `json:"version"`
	ExpiresAt     time.Time `json:"expires_at"`
	Body          string    `json:"body"`
}

type ContextScope struct {
	RepositoryID string   `json:"repository_id"`
	BrainID      string   `json:"brain_id"`
	AudienceID   string   `json:"audience_id"`
	ProjectID    string   `json:"project_id"`
	EntityIDs    []string `json:"entity_ids"`
	ReferenceIDs []string `json:"reference_ids"`
}

type CodeInput struct {
	Path   string `json:"path"`
	Body   string `json:"body"`
	SHA256 string `json:"sha256"`
}

// Manifest is constructed by the Go host from qualified snapshots, never from
// browser-authored claims. Settings must be canonicalized by the host.
type Manifest struct {
	RepositoryID    string          `json:"repository_id"`
	TaskRef         string          `json:"task_ref"`
	Question        string          `json:"question"`
	PlanDigest      string          `json:"plan_digest"`
	CodeDigest      string          `json:"code_digest"`
	PlanBody        string          `json:"plan_body"`
	Code            []CodeInput     `json:"code"`
	PromptVersion   string          `json:"prompt_version"`
	AnalyzerVersion string          `json:"analyzer_version"`
	Model           string          `json:"model"`
	Settings        json.RawMessage `json:"settings"`
	ContextMode     ContextMode     `json:"context_mode"`
	ContextScope    ContextScope    `json:"context_scope,omitempty"`
	Context         []ContextItem   `json:"context,omitempty"`
}

type Result struct {
	Key           string    `json:"key"`
	TaskRef       string    `json:"taskRef,omitempty"`
	Answer        string    `json:"answer,omitempty"`
	Model         string    `json:"model"`
	CreatedAt     time.Time `json:"createdAt,omitempty"`
	Freshness     string    `json:"freshness"`
	Status        string    `json:"status"`
	Persistence   bool      `json:"persistence"`
	Cached        bool      `json:"cached"`
	MemoryDerived bool      `json:"memoryDerived"`
	VisibleUntil  time.Time `json:"visibleUntil,omitempty"`
}

type Engine interface {
	Complete(context.Context, Manifest) (string, error)
}
type LineageValidator interface {
	ValidateLineage(context.Context, ContextScope, []ContextItem) (LineageValidation, error)
}
type LineageValidation struct {
	Available  bool
	Valid      bool
	ValidUntil time.Time
}

type CostDisclosure struct {
	Summary      string
	Acknowledged bool
}

type CancellationError struct {
	Err            error
	OutcomeUnknown bool
}

func (e *CancellationError) Error() string {
	return "analysis request canceled; provider outcome may be unknown"
}
func (e *CancellationError) Unwrap() error { return e.Err }

func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// ComputeCodeDigest returns the manifest digest expected by Analyze.
func ComputeCodeDigest(inputs []CodeInput) string {
	var b strings.Builder
	for _, c := range inputs {
		b.WriteString(c.Path)
		b.WriteByte(0)
		b.WriteString(c.SHA256)
		b.WriteByte('\n')
	}
	return digest(b.String())
}

type Config struct {
	Dir                        string
	MaxBytes                   int64
	Engine                     Engine
	Validator                  LineageValidator
	MemoryPersistenceQualified bool
}
