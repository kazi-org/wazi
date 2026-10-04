package context

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fixtureReader struct {
	qualification ContractQualification
	read          func(context.Context, Scope, Limits) (Bundle, error)
	validate      func(context.Context, Scope, []LineageRef) (LineageValidation, error)
	calls         int
}

func (f *fixtureReader) Qualification() ContractQualification { return f.qualification }
func (f *fixtureReader) ReadProjectContext(ctx context.Context, s Scope, l Limits) (Bundle, error) {
	f.calls++
	return f.read(ctx, s, l)
}
func (f *fixtureReader) ValidateLineage(ctx context.Context, scope Scope, refs []LineageRef) (LineageValidation, error) {
	if f.validate == nil {
		return LineageValidation{}, errors.New("fixture lineage not configured")
	}
	return f.validate(ctx, scope, refs)
}

func completeQualification() ContractQualification {
	return ContractQualification{OwnerRevision: "owner-rev", Protocol: "fixture", Authentication: "fixture-principal", ProviderFree: true, AudienceEnforced: true, ProjectScopeEnforced: true, VisibilityEnforced: true, TTLKnown: true, ForgetSemanticsKnown: true, CancellationSemanticsKnown: true, ResponseBoundsKnown: true, AllTypedSections: true, AllTypeLineage: true, FixtureContract: "fixture-v1"}
}

func fixtureConfig() HostConfig {
	return HostConfig{ByRepositoryID: map[string]Mapping{"repo-stable-1": {BrainID: "brain-1", AudienceID: "aud-1", ProjectID: "proj-1", EntityIDs: []string{"entity-1"}}}, Lease: time.Minute, MaxRecords: 10, MaxBodyBytes: 4096}
}

func TestUnqualifiedReaderReturnsHonestUnavailableWithoutCallingReader(t *testing.T) {
	r := &fixtureReader{qualification: ContractQualification{}}
	s, err := NewService(fixtureConfig(), r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Read(context.Background(), "repo-stable-1")
	if !errors.Is(err, ErrUnqualified) {
		t.Fatalf("Read error = %v, want ErrUnqualified", err)
	}
	if r.calls != 0 {
		t.Fatalf("unqualified reader called %d times", r.calls)
	}
	for _, name := range SectionNames() {
		if b.Sections[name].Status != Unavailable {
			t.Errorf("%s status = %s", name, b.Sections[name].Status)
		}
	}
}

func TestNoProjectMappingsStartInUnavailableState(t *testing.T) {
	s, err := NewService(HostConfig{}, nil)
	if err != nil {
		t.Fatalf("NewService with no mappings: %v", err)
	}
	b, err := s.Read(context.Background(), "repo-stable-1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Read error = %v, want unavailable", err)
	}
	for _, name := range SectionNames() {
		if b.Sections[name].Status != Unavailable {
			t.Errorf("%s status = %s", name, b.Sections[name].Status)
		}
	}
}

func TestLineageNegativeAndUnavailableStayDistinct(t *testing.T) {
	r := &fixtureReader{qualification: completeQualification()}
	s, err := NewService(fixtureConfig(), r)
	if err != nil {
		t.Fatal(err)
	}
	scope, ok := s.ScopeForRepository("repo-stable-1")
	if !ok {
		t.Fatal("mapped scope missing")
	}
	now := time.Now()
	ref := LineageRef{ReferenceID: "fact-1", Kind: Fact, BrainID: scope.BrainID, AudienceID: scope.AudienceID, ProjectID: scope.ProjectID, EntityID: "entity-1", ContentDigest: DigestContent("body"), Version: "v1", ExpiresAt: now.Add(time.Second)}
	base := LineageValidation{ValidatedAt: now, Results: []LineageResult{{ReferenceID: ref.ReferenceID, State: LineageIneligible, BrainID: ref.BrainID, AudienceID: ref.AudienceID, ProjectID: ref.ProjectID, EntityID: ref.EntityID, ContentDigest: ref.ContentDigest, Version: ref.Version, ExpiresAt: now.Add(time.Second)}}}
	r.validate = func(context.Context, Scope, []LineageRef) (LineageValidation, error) { return base, nil }
	if _, err := s.ValidateLineage(context.Background(), scope, []LineageRef{ref}); !errors.Is(err, ErrIneligible) {
		t.Fatalf("negative lineage error = %v", err)
	}
	base.Results[0].State = LineageUnavailable
	if _, err := s.ValidateLineage(context.Background(), scope, []LineageRef{ref}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unavailable lineage error = %v", err)
	}
}

func TestReferenceOnlyMappingAllowsUnmappedEntityAndRejectsUnknownReference(t *testing.T) {
	now := time.Now()
	r := &fixtureReader{qualification: completeQualification()}
	r.read = func(_ context.Context, scope Scope, _ Limits) (Bundle, error) {
		sections := map[SectionName]Section{}
		for _, name := range SectionNames() {
			sections[name] = Section{Status: Empty}
		}
		record := Record{ReferenceID: "ref-allowed", Kind: Fact, BrainID: scope.BrainID, AudienceID: scope.AudienceID, ProjectID: scope.ProjectID, EntityID: "entity-outside-mapping", Content: "fact body", ContentDigest: DigestContent("fact body"), Version: "v1", FetchedAt: now, ExpiresAt: now.Add(time.Minute)}
		sections[Facts] = Section{Status: Available, Records: []Record{record}}
		return Bundle{Scope: scope, FetchedAt: now, ValidTo: now.Add(time.Minute), Sections: sections}, nil
	}
	config := HostConfig{ByRepositoryID: map[string]Mapping{"repo-stable-1": {BrainID: "brain-1", AudienceID: "aud-1", ProjectID: "proj-1", ReferenceIDs: []string{"ref-allowed"}}}}
	s, err := NewService(config, r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(context.Background(), "repo-stable-1"); err != nil {
		t.Fatalf("mapped reference with unrelated entity was rejected: %v", err)
	}

	r.read = func(_ context.Context, scope Scope, _ Limits) (Bundle, error) {
		sections := map[SectionName]Section{}
		for _, name := range SectionNames() {
			sections[name] = Section{Status: Empty}
		}
		record := Record{ReferenceID: "ref-unknown", Kind: Fact, BrainID: scope.BrainID, AudienceID: scope.AudienceID, ProjectID: scope.ProjectID, EntityID: "entity-outside-mapping", Content: "fact body", ContentDigest: DigestContent("fact body"), Version: "v1", FetchedAt: now, ExpiresAt: now.Add(time.Minute)}
		sections[Facts] = Section{Status: Available, Records: []Record{record}}
		return Bundle{Scope: scope, FetchedAt: now, ValidTo: now.Add(time.Minute), Sections: sections}, nil
	}
	if _, err := s.Read(context.Background(), "repo-stable-1"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unmapped reference error = %v, want ErrInvalid", err)
	}
}

func TestLineageRejectsDuplicateRequestsVersionChangesAndOversize(t *testing.T) {
	config := fixtureConfig()
	scope := Scope{RepositoryID: "repo-stable-1", BrainID: "brain-1", AudienceID: "aud-1", ProjectID: "proj-1", EntityIDs: []string{"entity-1"}}
	now := time.Now()
	ref := LineageRef{ReferenceID: "fact-1", Kind: Fact, BrainID: scope.BrainID, AudienceID: scope.AudienceID, ProjectID: scope.ProjectID, EntityID: "entity-1", ContentDigest: DigestContent("body"), Version: "v1", ExpiresAt: now.Add(time.Hour)}
	calls := 0
	r := &fixtureReader{qualification: completeQualification()}
	r.validate = func(_ context.Context, _ Scope, refs []LineageRef) (LineageValidation, error) {
		calls++
		return LineageValidation{ValidatedAt: time.Now(), Results: []LineageResult{{ReferenceID: refs[0].ReferenceID, State: LineageEligible, BrainID: ref.BrainID, AudienceID: ref.AudienceID, ProjectID: ref.ProjectID, EntityID: ref.EntityID, ContentDigest: ref.ContentDigest, Version: "v2", ExpiresAt: time.Now().Add(5 * time.Minute)}}}, nil
	}
	s, err := NewService(config, r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ValidateLineage(context.Background(), scope, []LineageRef{ref, ref}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate request error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("duplicate request reached reader %d times", calls)
	}
	if _, err := s.ValidateLineage(context.Background(), scope, []LineageRef{ref}); !errors.Is(err, ErrIneligible) {
		t.Fatalf("changed version error = %v, want ErrIneligible", err)
	}

	r.validate = func(_ context.Context, _ Scope, refs []LineageRef) (LineageValidation, error) {
		return LineageValidation{ValidatedAt: time.Now(), Results: []LineageResult{{ReferenceID: refs[0].ReferenceID, State: LineageEligible, BrainID: ref.BrainID, AudienceID: ref.AudienceID, ProjectID: ref.ProjectID, EntityID: ref.EntityID, ContentDigest: ref.ContentDigest, Version: ref.Version, ExpiresAt: time.Now().Add(30 * time.Second)}}}, nil
	}
	v, err := s.ValidateLineage(context.Background(), scope, []LineageRef{ref})
	if err != nil {
		t.Fatal(err)
	}
	if !v.Results[0].ExpiresAt.Before(ref.ExpiresAt) {
		t.Fatalf("shortened owner expiry %s was not preserved in result", v.Results[0].ExpiresAt)
	}

	config.MaxResponseBytes = 10
	smallService, err := NewService(config, r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := smallService.ValidateLineage(context.Background(), scope, []LineageRef{ref}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized lineage response error = %v, want ErrInvalid", err)
	}
}

func TestReadAcceptsMappedTypedFixtureAndRejectsScopeEscape(t *testing.T) {
	r := &fixtureReader{qualification: completeQualification()}
	r.read = func(_ context.Context, scope Scope, _ Limits) (Bundle, error) {
		now := time.Now()
		sections := map[SectionName]Section{}
		for _, name := range SectionNames() {
			sections[name] = Section{Status: Empty}
		}
		sections[Facts] = Section{Status: Available, Records: []Record{{ReferenceID: "fact-1", Kind: Fact, BrainID: scope.BrainID, AudienceID: scope.AudienceID, ProjectID: scope.ProjectID, EntityID: "entity-1", Content: "typed fact", ContentDigest: DigestContent("typed fact"), Version: "v2", FetchedAt: now, ExpiresAt: now.Add(30 * time.Second)}}}
		return Bundle{Scope: scope, FetchedAt: now, ValidTo: now.Add(time.Minute), Sections: sections}, nil
	}
	s, err := NewService(fixtureConfig(), r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Read(context.Background(), "repo-stable-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Sections[Facts].Records[0].Kind; got != Fact {
		t.Fatalf("record kind = %q", got)
	}
	if !b.ValidTo.Equal(b.Sections[Facts].Records[0].ExpiresAt) {
		t.Fatalf("bundle lease %s exceeds earliest record expiry %s", b.ValidTo, b.Sections[Facts].Records[0].ExpiresAt)
	}
	r.read = func(_ context.Context, scope Scope, _ Limits) (Bundle, error) {
		now := time.Now()
		sections := map[SectionName]Section{}
		for _, name := range SectionNames() {
			sections[name] = Section{Status: Empty}
		}
		sections[Decisions] = Section{Status: Available, Records: []Record{{ReferenceID: "private-other-project", Kind: Decision, BrainID: scope.BrainID, AudienceID: scope.AudienceID, ProjectID: "other-project", EntityID: "entity-1", ContentDigest: "d", Version: "v", FetchedAt: now, ExpiresAt: now.Add(time.Second)}}}
		return Bundle{Scope: scope, FetchedAt: now, ValidTo: now.Add(time.Minute), Sections: sections}, nil
	}
	if _, err := s.Read(context.Background(), "repo-stable-1"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("scope escape error = %v, want ErrInvalid", err)
	}
}

func TestSessionCancelsAndDiscardsLateResponseAfterProjectSwitch(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	r := &fixtureReader{qualification: completeQualification()}
	r.read = func(ctx context.Context, scope Scope, _ Limits) (Bundle, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return Bundle{}, ctx.Err()
		}
		now := time.Now()
		sections := map[SectionName]Section{}
		for _, name := range SectionNames() {
			sections[name] = Section{Status: Empty}
		}
		return Bundle{Scope: scope, FetchedAt: now, ValidTo: now.Add(time.Minute), Sections: sections}, nil
	}
	svc, err := NewService(fixtureConfig(), r)
	if err != nil {
		t.Fatal(err)
	}
	session := NewSession(svc)
	gen, err := session.Select("repo-stable-1")
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := session.Refresh(context.Background(), gen); result <- err }()
	<-started
	if _, err := session.Select("unmapped-repo"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("select error = %v", err)
	}
	close(release)
	if err := <-result; !errors.Is(err, ErrDiscarded) {
		t.Fatalf("late refresh error = %v, want ErrDiscarded", err)
	}
	if _, ok := session.Current(); ok {
		t.Fatal("late response retained after selection change")
	}
}

func TestLeaseExpiryClearsRetainedBundle(t *testing.T) {
	s := &Session{bundle: &Bundle{ValidTo: time.Now().Add(-time.Second)}}
	if _, ok := s.Current(); ok {
		t.Fatal("expired bundle was visible")
	}
	if s.bundle != nil {
		t.Fatal("expired bundle body was not cleared")
	}
}
