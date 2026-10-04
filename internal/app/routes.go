// Package app joins the observatory's filesystem, scoped context and private AI boundaries.
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	brain "github.com/kazi-org/wazi/internal/context"
	"github.com/kazi-org/wazi/internal/deep"
	"github.com/kazi-org/wazi/internal/observatory"
)

type Routes struct {
	Observatory *observatory.Service
	Context     *brain.Service
	Deep        *deep.Service
	mu          sync.Mutex
	leases      map[string]lease
}
type lease struct {
	projectID, snapshotDigest string
	bundle                    brain.Bundle
}
type selection struct {
	ProjectID      string `json:"projectId"`
	PlanPath       string `json:"planPath"`
	TaskID         string `json:"taskId"`
	SnapshotDigest string `json:"snapshotDigest"`
	ContextLease   string `json:"contextLease"`
	Question       string `json:"question"`
	ContextMode    string `json:"contextMode"`
	Regenerate     bool   `json:"regenerate"`
	Key            string `json:"key"`
}

// Handle is used only behind observatory.Host's loopback, origin and capability checks.
func (a *Routes) Handle(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/context") && !strings.HasPrefix(r.URL.Path, "/api/deep/") {
		return false
	}
	if r.Method != http.MethodPost {
		replyError(w, http.StatusMethodNotAllowed, "POST required")
		return true
	}
	var input selection
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		replyError(w, 400, "Invalid local request.")
		return true
	}
	var tail any
	if err := decoder.Decode(&tail); err != io.EOF {
		replyError(w, 400, "One JSON request is required.")
		return true
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	snapshot, err := a.Observatory.Snapshot(ctx, input.ProjectID)
	if err != nil {
		replyError(w, 409, "Selected project is unavailable; refresh local projects.")
		return true
	}
	switch r.URL.Path {
	case "/api/context":
		if a.Context == nil {
			replyJSON(w, unavailable("No qualified Serenity owner reader configured."))
			return true
		}
		bundle, readErr := a.Context.Read(ctx, snapshot.RepositoryID)
		if readErr != nil {
			replyJSON(w, unavailable("The scoped Serenity owner read API is unavailable or unqualified."))
			return true
		}
		token, err := observatory.RandomCapability()
		if err != nil {
			replyError(w, 500, "Could not establish context lease.")
			return true
		}
		a.mu.Lock()
		if a.leases == nil {
			a.leases = make(map[string]lease)
		}
		for key, value := range a.leases {
			if !time.Now().Before(value.bundle.ValidTo) {
				delete(a.leases, key)
			}
		}
		if len(a.leases) >= 128 {
			a.mu.Unlock()
			replyError(w, 503, "Context lease capacity reached.")
			return true
		}
		a.leases[token] = lease{input.ProjectID, snapshot.SnapshotDigest, bundle}
		a.mu.Unlock()
		sections := make(map[string]any)
		for name, section := range bundle.Sections {
			key := string(name)
			if name == brain.Questions {
				key = "openQuestions"
			}
			items := make([]map[string]any, 0, len(section.Records))
			for _, record := range section.Records {
				items = append(items, map[string]any{"id": record.ReferenceID, "text": record.Content, "kind": record.Kind, "version": record.Version, "contentDigest": record.ContentDigest, "expiresAt": record.ExpiresAt})
			}
			sections[key] = map[string]any{"status": section.Status, "items": items}
		}
		replyJSON(w, map[string]any{"leaseId": token, "expiresAt": bundle.ValidTo, "fetchedAt": bundle.FetchedAt, "scope": map[string]string{"projectId": bundle.Scope.ProjectID, "audience": bundle.Scope.AudienceID}, "sections": sections})
	case "/api/deep/answer":
		if a.Deep == nil {
			replyError(w, 503, "Private analysis storage is unavailable.")
			return true
		}
		if input.SnapshotDigest != snapshot.SnapshotDigest {
			replyError(w, 409, "Plan/code changed. Refresh before sending a new request.")
			return true
		}
		coherent, err := a.Observatory.ResolveTask(snapshot, input.PlanPath, input.TaskID)
		if err != nil {
			replyError(w, 409, "Task source is stale or ambiguous. Refresh before sending.")
			return true
		}
		manifest := deep.Manifest{RepositoryID: snapshot.RepositoryID, TaskRef: input.PlanPath + "#" + input.TaskID, Question: strings.TrimSpace(input.Question), PlanBody: coherent.RawPlan, PlanDigest: hash(coherent.RawPlan), CodeDigest: snapshot.SnapshotDigest, PromptVersion: "wazi.observatory.v1", AnalyzerVersion: "wazi.local.v1", Model: deep.Model, Settings: json.RawMessage(`{"max_tokens":2048,"temperature":0.2}`), ContextMode: deep.ContextPlanCode}
		for _, source := range coherent.Sources {
			manifest.Code = append(manifest.Code, deep.CodeInput{Path: source.Path, Body: source.Body, SHA256: strings.TrimPrefix(source.SHA256, "sha256:")})
		}
		manifest.CodeDigest = deep.ComputeCodeDigest(manifest.Code)
		if input.ContextMode == "with-context" {
			a.mu.Lock()
			shown, ok := a.leases[input.ContextLease]
			a.mu.Unlock()
			if !ok || shown.projectID != input.ProjectID || shown.snapshotDigest != snapshot.SnapshotDigest || !time.Now().Before(shown.bundle.ValidTo) {
				replyError(w, 409, "Shown context is unavailable or expired. Refresh it, or explicitly choose plan/code only.")
				return true
			}
			scope := shown.bundle.Scope
			manifest.ContextMode = deep.ContextMemory
			manifest.ContextScope = deep.ContextScope{RepositoryID: scope.RepositoryID, BrainID: scope.BrainID, AudienceID: scope.AudienceID, ProjectID: scope.ProjectID, EntityIDs: scope.EntityIDs, ReferenceIDs: scope.ReferenceIDs}
			for _, name := range brain.SectionNames() {
				section := shown.bundle.Sections[name]
				if section.Status == brain.Unavailable {
					replyError(w, 409, "Required context section is unavailable; choose plan/code only explicitly.")
					return true
				}
				for _, record := range section.Records {
					manifest.Context = append(manifest.Context, deep.ContextItem{OwnerRef: record.ReferenceID, Kind: string(record.Kind), EntityID: record.EntityID, BrainID: record.BrainID, AudienceID: record.AudienceID, ProjectID: record.ProjectID, ContentDigest: record.ContentDigest, Version: record.Version, ExpiresAt: record.ExpiresAt, Body: record.Content})
				}
			}
		} else if input.ContextMode != "plan-code-only" {
			replyError(w, 400, "Choose an explicit context input mode.")
			return true
		}
		var result deep.Result
		if input.Regenerate {
			result, err = a.Deep.Regenerate(ctx, manifest, deep.CostDisclosure{Summary: "User clicked Regenerate: a new request may incur a charge.", Acknowledged: true})
		} else {
			result, err = a.Deep.Analyze(ctx, manifest)
		}
		if err != nil {
			replyError(w, 409, safeDeepError(err))
			return true
		}
		replyJSON(w, result)
	case "/api/deep/delete":
		if a.Deep == nil {
			replyError(w, 503, "Private analysis storage unavailable.")
			return true
		}
		// Repository ownership is checked by the store; an opaque key is never scope authority.
		if err := a.Deep.DeleteForRepository(ctx, input.Key, snapshot.RepositoryID); err != nil {
			replyError(w, 409, "Saved answer cannot be deleted in this project scope.")
			return true
		}
		replyJSON(w, map[string]string{"status": "deleted"})
	case "/api/deep/list":
		if a.Deep == nil {
			replyError(w, 503, "Private analysis storage unavailable.")
			return true
		}
		results, err := a.Deep.List(ctx, snapshot.RepositoryID)
		if err != nil {
			replyError(w, 503, "Private receipts are unavailable.")
			return true
		}
		replyJSON(w, map[string]any{"results": results})
	case "/api/deep/inspect":
		if a.Deep == nil {
			replyError(w, 503, "Private analysis storage unavailable.")
			return true
		}
		result, err := a.Deep.InspectForRepository(ctx, input.Key, snapshot.RepositoryID)
		if err != nil {
			replyError(w, 409, safeDeepError(err))
			return true
		}
		replyJSON(w, result)
	default:
		replyError(w, 404, "Local route not found.")
	}
	return true
}
func unavailable(reason string) map[string]any {
	sections := map[string]any{}
	for _, name := range []string{"facts", "decisions", "constraints", "intents", "openQuestions"} {
		sections[name] = map[string]string{"status": "unavailable", "reason": reason}
	}
	return map[string]any{"sections": sections, "unavailable": reason}
}
func hash(body string) string { sum := sha256.Sum256([]byte(body)); return hex.EncodeToString(sum[:]) }
func replyJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(value)
}
func replyError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func safeDeepError(err error) string {
	switch {
	case errors.Is(err, deep.ErrUnknownOutcome):
		return "Previous provider outcome is uncertain. It will not be resent automatically. Inspect the receipt or explicitly regenerate."
	case errors.Is(err, deep.ErrInProgress):
		return "An identical request is already in progress."
	case errors.Is(err, deep.ErrCapacity):
		return "Private cache capacity is full. Delete saved answers before requesting more."
	case errors.Is(err, deep.ErrMemoryUnavailable):
		return "Memory-derived answers are hidden because current lineage cannot be validated."
	case errors.Is(err, deep.ErrLineageInvalid):
		return "Memory lineage changed or expired; derived answer was invalidated."
	default:
		return "Analysis is unavailable or its provider outcome is uncertain. No automatic retry will occur."
	}
}
