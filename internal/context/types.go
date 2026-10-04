// Package context defines Wazi's bounded, read-only Serenity context contract.
// It contains no transport implementation and cannot qualify an owner API by
// itself. A concrete reader must provide independently established evidence.
package context

import (
	"context"
	"errors"
	"time"
)

type SectionName string

const (
	Facts       SectionName = "facts"
	Decisions   SectionName = "decisions"
	Constraints SectionName = "constraints"
	Intents     SectionName = "intents"
	Questions   SectionName = "open_questions"
)

var requiredSections = [...]SectionName{Facts, Decisions, Constraints, Intents, Questions}

// SectionNames returns the five typed panel sections in stable display order.
func SectionNames() []SectionName { return append([]SectionName(nil), requiredSections[:]...) }

type SectionStatus string

const (
	Available   SectionStatus = "available"
	Empty       SectionStatus = "empty"
	Unavailable SectionStatus = "unavailable"
)

type RecordKind string

const (
	Fact       RecordKind = "fact"
	Decision   RecordKind = "decision"
	Constraint RecordKind = "constraint"
	Intent     RecordKind = "intent"
	Question   RecordKind = "open_question"
)

// Scope is always constructed from explicit trusted host configuration. Names
// and display labels are deliberately absent; they must never imply scope.
type Scope struct {
	RepositoryID string   `json:"repositoryId"`
	BrainID      string   `json:"brainId"`
	AudienceID   string   `json:"audienceId"`
	ProjectID    string   `json:"projectId"`
	EntityIDs    []string `json:"entityIds"`
	ReferenceIDs []string `json:"referenceIds"`
}

func (s Scope) valid() bool {
	return s.RepositoryID != "" && s.BrainID != "" && s.AudienceID != "" && s.ProjectID != "" && (len(s.EntityIDs) > 0 || len(s.ReferenceIDs) > 0)
}

type Mapping struct {
	BrainID      string
	AudienceID   string
	ProjectID    string
	EntityIDs    []string
	ReferenceIDs []string
}

// HostConfig maps stable repository IDs to owner identifiers. It intentionally
// has no display-name lookup or global/default scope.
type HostConfig struct {
	ByRepositoryID   map[string]Mapping
	Lease            time.Duration
	MaxRecords       int
	MaxBodyBytes     int
	MaxResponseBytes int
}

type Record struct {
	ReferenceID   string     `json:"referenceId"`
	Kind          RecordKind `json:"kind"`
	BrainID       string     `json:"brainId"`
	AudienceID    string     `json:"audienceId"`
	ProjectID     string     `json:"projectId"`
	EntityID      string     `json:"entityId"`
	Content       string     `json:"content"`
	ContentDigest string     `json:"contentDigest"`
	Version       string     `json:"version"`
	SourceURI     *string    `json:"sourceUri,omitempty"`
	Attribution   *string    `json:"attribution,omitempty"`
	FetchedAt     time.Time  `json:"fetchedAt"`
	ExpiresAt     time.Time  `json:"expiresAt"`
}

type Section struct {
	Status  SectionStatus `json:"status"`
	Records []Record      `json:"records"`
}

type Bundle struct {
	Scope     Scope                   `json:"scope"`
	FetchedAt time.Time               `json:"fetchedAt"`
	ValidTo   time.Time               `json:"validTo"`
	Sections  map[SectionName]Section `json:"sections"`
}

// ContractQualification is an owner-reader's evidence declaration. All gates
// are mandatory; a Wazi fixture or arbitrary implementation cannot establish
// owner qualification without the cited revision and fixture contract.
type ContractQualification struct {
	OwnerRevision              string
	Protocol                   string
	Authentication             string
	ProviderFree               bool
	AudienceEnforced           bool
	ProjectScopeEnforced       bool
	VisibilityEnforced         bool
	TTLKnown                   bool
	ForgetSemanticsKnown       bool
	CancellationSemanticsKnown bool
	ResponseBoundsKnown        bool
	AllTypedSections           bool
	AllTypeLineage             bool
	FixtureContract            string
}

func (q ContractQualification) valid() bool {
	return q.OwnerRevision != "" && q.Protocol != "" && q.Authentication != "" && q.ProviderFree && q.AudienceEnforced && q.ProjectScopeEnforced && q.VisibilityEnforced && q.TTLKnown && q.ForgetSemanticsKnown && q.CancellationSemanticsKnown && q.ResponseBoundsKnown && q.AllTypedSections && q.AllTypeLineage && q.FixtureContract != ""
}

type Reader interface {
	Qualification() ContractQualification
	ReadProjectContext(context.Context, Scope, Limits) (Bundle, error)
	ValidateLineage(context.Context, Scope, []LineageRef) (LineageValidation, error)
}

type Limits struct {
	Records       int
	BodyBytes     int
	ResponseBytes int
}

var (
	ErrUnavailable = errors.New("Serenity context is unavailable")
	ErrUnqualified = errors.New("Serenity reader contract is not qualified")
	ErrInvalid     = errors.New("invalid Serenity context response")
	ErrIneligible  = errors.New("Serenity memory lineage is no longer eligible")
)

type LineageRef struct {
	ReferenceID   string     `json:"referenceId"`
	Kind          RecordKind `json:"kind"`
	BrainID       string     `json:"brainId"`
	AudienceID    string     `json:"audienceId"`
	ProjectID     string     `json:"projectId"`
	EntityID      string     `json:"entityId"`
	ContentDigest string     `json:"contentDigest"`
	Version       string     `json:"version"`
	ExpiresAt     time.Time  `json:"expiresAt"`
}

type LineageState string

const (
	LineageEligible    LineageState = "eligible"
	LineageIneligible  LineageState = "ineligible"
	LineageUnavailable LineageState = "unavailable"
)

type LineageResult struct {
	ReferenceID   string       `json:"referenceId"`
	State         LineageState `json:"state"`
	BrainID       string       `json:"brainId"`
	AudienceID    string       `json:"audienceId"`
	ProjectID     string       `json:"projectId"`
	EntityID      string       `json:"entityId"`
	ContentDigest string       `json:"contentDigest"`
	Version       string       `json:"version"`
	ExpiresAt     time.Time    `json:"expiresAt"`
}

type LineageValidation struct {
	ValidatedAt time.Time       `json:"validatedAt"`
	Results     []LineageResult `json:"results"`
}
