package app

import (
	"context"
	"errors"
	brain "github.com/kazi-org/wazi/internal/context"
	"github.com/kazi-org/wazi/internal/deep"
)

// LineageAdapter never upgrades a fixture contract to Serenity owner authority.
type LineageAdapter struct{ Service *brain.Service }

func (a LineageAdapter) ValidateLineage(ctx context.Context, scope deep.ContextScope, items []deep.ContextItem) (deep.LineageValidation, error) {
	if a.Service == nil {
		return deep.LineageValidation{Available: false}, deep.ErrMemoryUnavailable
	}
	ownerScope := brain.Scope{RepositoryID: scope.RepositoryID, BrainID: scope.BrainID, AudienceID: scope.AudienceID, ProjectID: scope.ProjectID, EntityIDs: scope.EntityIDs, ReferenceIDs: scope.ReferenceIDs}
	refs := make([]brain.LineageRef, 0, len(items))
	for _, item := range items {
		refs = append(refs, brain.LineageRef{ReferenceID: item.OwnerRef, Kind: brain.RecordKind(item.Kind), EntityID: item.EntityID, BrainID: item.BrainID, AudienceID: item.AudienceID, ProjectID: item.ProjectID, ContentDigest: item.ContentDigest, Version: item.Version, ExpiresAt: item.ExpiresAt})
	}
	_, err := a.Service.ValidateLineage(ctx, ownerScope, refs)
	if errors.Is(err, brain.ErrIneligible) || errors.Is(err, brain.ErrInvalid) {
		return deep.LineageValidation{Available: true, Valid: false}, deep.ErrLineageInvalid
	}
	if err != nil {
		return deep.LineageValidation{Available: false}, deep.ErrMemoryUnavailable
	}
	return deep.LineageValidation{Available: true, Valid: true}, nil
}
