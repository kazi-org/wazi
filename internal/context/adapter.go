package context

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Service struct {
	config    HostConfig
	reader    Reader
	qualified bool
}

func NewService(config HostConfig, reader Reader) (*Service, error) {
	if config.Lease <= 0 {
		config.Lease = 60 * time.Second
	}
	if config.MaxRecords <= 0 {
		config.MaxRecords = 100
	}
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = 64 * 1024
	}
	if len(config.ByRepositoryID) == 0 {
		return nil, fmt.Errorf("creating context service: no explicit repository mappings")
	}
	copyMap := make(map[string]Mapping, len(config.ByRepositoryID))
	for repo, m := range config.ByRepositoryID {
		if repo == "" || m.BrainID == "" || m.AudienceID == "" || m.ProjectID == "" || (len(m.EntityIDs) == 0 && len(m.ReferenceIDs) == 0) {
			return nil, fmt.Errorf("creating context service: incomplete explicit mapping for repository %q", repo)
		}
		m.EntityIDs = sortedUnique(m.EntityIDs)
		m.ReferenceIDs = sortedUnique(m.ReferenceIDs)
		if len(m.EntityIDs) == 0 && len(m.ReferenceIDs) == 0 {
			return nil, fmt.Errorf("creating context service: mapping for repository %q has no usable entity or reference IDs", repo)
		}
		copyMap[repo] = m
	}
	config.ByRepositoryID = copyMap
	qualified := reader != nil && reader.Qualification().valid()
	return &Service{config: config, reader: reader, qualified: qualified}, nil
}

func (s *Service) ScopeForRepository(repositoryID string) (Scope, bool) {
	m, ok := s.config.ByRepositoryID[repositoryID]
	if !ok {
		return Scope{}, false
	}
	return Scope{RepositoryID: repositoryID, BrainID: m.BrainID, AudienceID: m.AudienceID, ProjectID: m.ProjectID, EntityIDs: append([]string(nil), m.EntityIDs...), ReferenceIDs: append([]string(nil), m.ReferenceIDs...)}, true
}

func (s *Service) Read(ctx context.Context, repositoryID string) (Bundle, error) {
	scope, ok := s.ScopeForRepository(repositoryID)
	if !ok {
		return unavailableBundle(Scope{RepositoryID: repositoryID}), ErrUnavailable
	}
	if !s.qualified {
		return unavailableBundle(scope), ErrUnqualified
	}
	limits := Limits{Records: s.config.MaxRecords, BodyBytes: s.config.MaxBodyBytes}
	b, err := s.reader.ReadProjectContext(ctx, scope, limits)
	if err != nil {
		return unavailableBundle(scope), fmt.Errorf("reading scoped Serenity context: %w", err)
	}
	if err := validateBundle(scope, b, limits); err != nil {
		return unavailableBundle(scope), err
	}
	leaseEnd := time.Now().Add(s.config.Lease)
	if !b.ValidTo.IsZero() && b.ValidTo.Before(leaseEnd) {
		leaseEnd = b.ValidTo
	}
	b.ValidTo = leaseEnd
	return b, nil
}

func (s *Service) ValidateLineage(ctx context.Context, scope Scope, refs []LineageRef) (LineageValidation, error) {
	if !s.qualified {
		return LineageValidation{}, ErrUnqualified
	}
	expectedScope, mapped := s.ScopeForRepository(scope.RepositoryID)
	if !mapped || !sameScope(scope, expectedScope) || len(refs) == 0 {
		return LineageValidation{}, ErrInvalid
	}
	for _, r := range refs {
		if r.ReferenceID == "" || r.ContentDigest == "" || r.Version == "" || r.BrainID != scope.BrainID || r.AudienceID != scope.AudienceID || r.ProjectID != scope.ProjectID || !scopeAllows(scope, r.EntityID, r.ReferenceID) {
			return LineageValidation{}, ErrInvalid
		}
	}
	v, err := s.reader.ValidateLineage(ctx, scope, refs)
	if err != nil {
		return LineageValidation{}, fmt.Errorf("revalidating Serenity lineage: %w", err)
	}
	if len(v.Results) != len(refs) {
		return LineageValidation{}, fmt.Errorf("%w: incomplete lineage result set", ErrInvalid)
	}
	byID := make(map[string]LineageResult, len(v.Results))
	for _, got := range v.Results {
		if got.ReferenceID == "" {
			return LineageValidation{}, ErrInvalid
		}
		if _, exists := byID[got.ReferenceID]; exists {
			return LineageValidation{}, fmt.Errorf("%w: duplicate lineage result", ErrInvalid)
		}
		byID[got.ReferenceID] = got
	}
	for _, want := range refs {
		got, exists := byID[want.ReferenceID]
		if !exists || got.State != LineageEligible || got.BrainID != want.BrainID || got.AudienceID != want.AudienceID || got.ProjectID != want.ProjectID || got.EntityID != want.EntityID || got.ContentDigest != want.ContentDigest || got.Version != want.Version || got.ExpiresAt.IsZero() || !time.Now().Before(got.ExpiresAt) {
			return v, fmt.Errorf("%w: lineage is stale or ineligible", ErrUnavailable)
		}
	}
	return v, nil
}

func validateBundle(scope Scope, b Bundle, lim Limits) error {
	if !sameScope(b.Scope, scope) || b.FetchedAt.IsZero() || b.ValidTo.IsZero() || !time.Now().Before(b.ValidTo) {
		return fmt.Errorf("%w: invalid scope or validity", ErrInvalid)
	}
	allowedEntities, allowedRefs := setOf(scope.EntityIDs), setOf(scope.ReferenceIDs)
	count, bytes := 0, 0
	for _, sectionName := range RequiredSections {
		section, ok := b.Sections[sectionName]
		if !ok || (section.Status != Available && section.Status != Empty && section.Status != Unavailable) {
			return fmt.Errorf("%w: missing or invalid section %q", ErrInvalid, sectionName)
		}
		if section.Status == Empty && len(section.Records) != 0 || section.Status == Unavailable && len(section.Records) != 0 || section.Status == Available && len(section.Records) == 0 {
			return fmt.Errorf("%w: section status disagrees with records", ErrInvalid)
		}
		for _, r := range section.Records {
			if r.ReferenceID == "" || r.BrainID != scope.BrainID || r.AudienceID != scope.AudienceID || r.ProjectID != scope.ProjectID || !scopeAllows(scope, r.EntityID, r.ReferenceID) || r.ContentDigest == "" || r.Version == "" || r.FetchedAt.IsZero() || r.ExpiresAt.IsZero() || !time.Now().Before(r.ExpiresAt) || r.ExpiresAt.After(b.ValidTo) || r.Kind != kindFor(sectionName) {
				return fmt.Errorf("%w: record outside selected scope or without current provenance", ErrInvalid)
			}
			if len(allowedEntities) > 0 && r.EntityID != "" && !allowedEntities[r.EntityID] || len(allowedRefs) > 0 && !allowedRefs[r.ReferenceID] {
				return fmt.Errorf("%w: unmapped owner reference", ErrInvalid)
			}
			count++
			bytes += len(r.Content)
		}
	}
	if count > lim.Records || bytes > lim.BodyBytes {
		return fmt.Errorf("%w: response exceeds configured bound", ErrInvalid)
	}
	return nil
}

func sameScope(a, b Scope) bool {
	if a.RepositoryID != b.RepositoryID || a.BrainID != b.BrainID || a.AudienceID != b.AudienceID || a.ProjectID != b.ProjectID {
		return false
	}
	return sameStrings(a.EntityIDs, b.EntityIDs) && sameStrings(a.ReferenceIDs, b.ReferenceIDs)
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	left, right := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func kindFor(s SectionName) RecordKind {
	switch s {
	case Facts:
		return Fact
	case Decisions:
		return Decision
	case Constraints:
		return Constraint
	case Intents:
		return Intent
	case Questions:
		return Question
	default:
		return ""
	}
}

func scopeAllows(s Scope, entity, ref string) bool {
	for _, id := range s.EntityIDs {
		if entity != "" && id == entity {
			return true
		}
	}
	for _, id := range s.ReferenceIDs {
		if id == ref {
			return true
		}
	}
	return false
}

func unavailableBundle(scope Scope) Bundle {
	sections := make(map[SectionName]Section, len(RequiredSections))
	for _, n := range RequiredSections {
		sections[n] = Section{Status: Unavailable}
	}
	return Bundle{Scope: scope, Sections: sections}
}

func sortedUnique(v []string) []string {
	set := map[string]bool{}
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			set[s] = true
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func setOf(v []string) map[string]bool {
	m := make(map[string]bool, len(v))
	for _, s := range v {
		m[s] = true
	}
	return m
}
