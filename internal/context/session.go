package context

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrDiscarded = errors.New("context response discarded after scope change or cancellation")

// Session binds reads to one selected repository and generation. Changing the
// selection cancels in-flight work and drops retained bodies immediately.
type Session struct {
	mu         sync.Mutex
	service    *Service
	generation uint64
	scope      Scope
	selected   bool
	cancel     context.CancelFunc
	bundle     *Bundle
}

func NewSession(service *Service) *Session { return &Session{service: service} }

func (s *Session) Select(repositoryID string) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	s.bundle = nil
	scope, ok := s.service.ScopeForRepository(repositoryID)
	s.selected = ok
	if ok {
		s.scope = scope
	} else {
		s.scope = Scope{}
	}
	if !ok {
		return s.generation, ErrUnavailable
	}
	return s.generation, nil
}

func (s *Session) Refresh(parent context.Context, generation uint64) (Bundle, error) {
	s.mu.Lock()
	if !s.selected || generation != s.generation {
		s.mu.Unlock()
		return Bundle{}, ErrDiscarded
	}
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	repositoryID := s.scope.RepositoryID
	s.mu.Unlock()

	b, err := s.service.Read(ctx, repositoryID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if generation != s.generation || ctx.Err() != nil {
		cancel()
		return Bundle{}, ErrDiscarded
	}
	s.cancel = nil
	cancel()
	if err != nil {
		s.bundle = nil
		return b, err
	}
	s.bundle = &b
	return b, nil
}

// Current returns a copy only while the bounded lease remains live. Once stale,
// the retained memory bodies are cleared before returning.
func (s *Session) Current() (Bundle, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bundle == nil {
		return Bundle{}, false
	}
	if !nowBefore(s.bundle.ValidTo) {
		s.bundle = nil
		return Bundle{}, false
	}
	return cloneBundle(*s.bundle), true
}

func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	s.selected = false
	s.scope = Scope{}
	s.bundle = nil
}

func nowBefore(t time.Time) bool { return time.Now().Before(t) }

func cloneBundle(b Bundle) Bundle {
	b.Scope.EntityIDs = append([]string(nil), b.Scope.EntityIDs...)
	b.Scope.ReferenceIDs = append([]string(nil), b.Scope.ReferenceIDs...)
	sections := make(map[SectionName]Section, len(b.Sections))
	for k, v := range b.Sections {
		v.Records = append([]Record(nil), v.Records...)
		for i := range v.Records {
			v.Records[i].SourceURI = cloneString(v.Records[i].SourceURI)
			v.Records[i].Attribution = cloneString(v.Records[i].Attribution)
		}
		sections[k] = v
	}
	b.Sections = sections
	return b
}

func cloneString(v *string) *string {
	if v == nil {
		return nil
	}
	copyValue := *v
	return &copyValue
}
