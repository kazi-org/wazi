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

var RequiredSections = [...]SectionName{Facts, Decisions, Constraints, Intents, Questions}

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
	RepositoryID string
	BrainID      string
	AudienceID   string
	ProjectID    string
	EntityIDs    []string
	ReferenceIDs []string
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
	ByRepositoryID map[string]Mapping
	Lease          time.Duration
	MaxRecords     int
	MaxBodyBytes   int
}

type Record struct {
	ReferenceID   string
	Kind          RecordKind
	BrainID       string
	AudienceID    string
	ProjectID     string
	EntityID      string
	Content       string
	ContentDigest string
	Version       string
	SourceURI     *string
	Attribution   *string
	FetchedAt     time.Time
	ExpiresAt     time.Time
}

type Section struct {
	Status  SectionStatus
	Records []Record
}

type Bundle struct {
	Scope     Scope
	FetchedAt time.Time
	ValidTo   time.Time
	Sections  map[SectionName]Section
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
	Records   int
	BodyBytes int
}

var (
	ErrUnavailable = errors.New("Serenity context is unavailable")
	ErrUnqualified = errors.New("Serenity reader contract is not qualified")
	ErrInvalid     = errors.New("invalid Serenity context response")
)

type LineageRef struct {
	ReferenceID   string
	Kind          RecordKind
	BrainID       string
	AudienceID    string
	ProjectID     string
	EntityID      string
	ContentDigest string
	Version       string
	ExpiresAt     time.Time
}

type LineageState string

const (
	LineageEligible    LineageState = "eligible"
	LineageIneligible  LineageState = "ineligible"
	LineageUnavailable LineageState = "unavailable"
)

type LineageResult struct {
	ReferenceID   string
	State         LineageState
	BrainID       string
	AudienceID    string
	ProjectID     string
	EntityID      string
	ContentDigest string
	Version       string
	ExpiresAt     time.Time
}

type LineageValidation struct {
	ValidatedAt time.Time
	Results     []LineageResult
}
