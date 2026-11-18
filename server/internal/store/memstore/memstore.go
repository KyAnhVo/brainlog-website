// Package memstore is an in-memory store.Store: the mock database used until MySQL lands.
// It is safe for concurrent use. Everything is lost when the process exits.
package memstore

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
	"github.com/KyAnhVo/brainlog-website/server/internal/textnorm"
)

type trackingRow struct {
	tracking model.Tracking
	statuses []model.Status // kept sorted by model.StatusLess
}

type Store struct {
	mu        sync.RWMutex
	accounts  map[string]model.Account // by id
	byEmail   map[string]string        // lowercased email -> id
	sessions  map[ids.SessionID]model.Session
	trackings map[string]*trackingRow // by trackingNumber
}

var _ store.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		accounts:  map[string]model.Account{},
		byEmail:   map[string]string{},
		sessions:  map[ids.SessionID]model.Session{},
		trackings: map[string]*trackingRow{},
	}
}

func emailKey(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// ---------------------------------------------------------------------------
// Accounts
// ---------------------------------------------------------------------------

func (s *Store) CreateAccount(_ context.Context, a model.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.byEmail[emailKey(a.Email)]; dup {
		return store.ErrConflict
	}
	if _, dup := s.accounts[a.ID]; dup {
		return store.ErrConflict
	}
	s.accounts[a.ID] = a
	s.byEmail[emailKey(a.Email)] = a.ID
	return nil
}

func (s *Store) AccountByID(_ context.Context, id string) (model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.accounts[id]
	if !ok {
		return model.Account{}, store.ErrNotFound
	}
	return a, nil
}

func (s *Store) AccountByEmail(_ context.Context, email string) (model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byEmail[emailKey(email)]
	if !ok {
		return model.Account{}, store.ErrNotFound
	}
	return s.accounts[id], nil
}

func (s *Store) DeleteAccountByID(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteAccountLocked(id)
}

func (s *Store) DeleteAccountByEmail(_ context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byEmail[emailKey(email)]
	if !ok {
		return store.ErrNotFound
	}
	return s.deleteAccountLocked(id)
}

func (s *Store) deleteAccountLocked(id string) error {
	a, ok := s.accounts[id]
	if !ok {
		return store.ErrNotFound
	}
	delete(s.accounts, id)
	delete(s.byEmail, emailKey(a.Email))
	for sid, sess := range s.sessions {
		if sess.UserID == id {
			delete(s.sessions, sid)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

func (s *Store) CreateSession(_ context.Context, sess model.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.sessions[sess.ID]; dup {
		return store.ErrConflict
	}
	s.sessions[sess.ID] = sess
	return nil
}

func (s *Store) SessionByID(_ context.Context, id ids.SessionID) (model.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	if !ok {
		return model.Session{}, store.ErrNotFound
	}
	return sess, nil
}

func (s *Store) InvalidateSession(_ context.Context, id ids.SessionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return store.ErrNotFound
	}
	sess.IsValid = false
	s.sessions[id] = sess
	return nil
}

// ---------------------------------------------------------------------------
// Trackings
// ---------------------------------------------------------------------------

func cloneStatus(st model.Status) model.Status {
	st.Warnings = slices.Clone(st.Warnings)
	if st.Type == model.StatusGood && st.Warnings == nil {
		st.Warnings = []string{}
	}
	return st
}

func cloneTracking(t model.Tracking) model.Tracking {
	if t.Physical != nil {
		p := *t.Physical
		if p.PcsCount != nil {
			n := *p.PcsCount
			p.PcsCount = &n
		}
		t.Physical = &p
	}
	return t
}

func (s *Store) CreateTracking(_ context.Context, t model.Tracking, first model.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.trackings[t.TrackingNumber]; dup {
		return store.ErrConflict
	}
	s.trackings[t.TrackingNumber] = &trackingRow{
		tracking: cloneTracking(t),
		statuses: []model.Status{cloneStatus(first)},
	}
	return nil
}

func (s *Store) Tracking(_ context.Context, trackingNumber string) (model.Tracking, []model.Status, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row, ok := s.trackings[trackingNumber]
	if !ok {
		return model.Tracking{}, nil, store.ErrNotFound
	}
	out := make([]model.Status, len(row.statuses))
	for i, st := range row.statuses {
		out[i] = cloneStatus(st)
	}
	return cloneTracking(row.tracking), out, nil
}

func (s *Store) AddStatus(_ context.Context, trackingNumber string, st model.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.trackings[trackingNumber]
	if !ok {
		return store.ErrNotFound
	}
	st = cloneStatus(st)
	i, _ := slices.BinarySearchFunc(row.statuses, st, func(a, b model.Status) int {
		if model.StatusLess(a, b) {
			return -1
		}
		if model.StatusLess(b, a) {
			return 1
		}
		return 0
	})
	row.statuses = slices.Insert(row.statuses, i, st)
	return nil
}

// listRow carries the folded text keys so each row is normalised once per query.
type listRow struct {
	model.TrackingWithLast
	number, pol, pod string
}

type sortKey int

const (
	keyTrackingNumber sortKey = iota
	keyPOL
	keyPOD
	keyLastOccurring
	keyLastCreation
	keyCreation
)

func compareKey(a, b *listRow, k sortKey) int {
	switch k {
	case keyTrackingNumber:
		return cmp.Compare(a.number, b.number)
	case keyPOL:
		return cmp.Compare(a.pol, b.pol)
	case keyPOD:
		return cmp.Compare(a.pod, b.pod)
	case keyLastOccurring:
		return a.LastStatus.OccurringTime.Compare(b.LastStatus.OccurringTime)
	case keyLastCreation:
		return a.LastStatus.CreationTime.Compare(b.LastStatus.CreationTime)
	default:
		return a.CreationTime.Compare(b.CreationTime)
	}
}

func sortKeyOf(f store.SortField) sortKey {
	switch f {
	case store.SortTrackingNumber:
		return keyTrackingNumber
	case store.SortPOL:
		return keyPOL
	case store.SortPOD:
		return keyPOD
	case store.SortCreationTime:
		return keyCreation
	default:
		return keyLastOccurring
	}
}

type orderKey struct {
	key  sortKey
	sign int // +1 ASC, -1 DESC
}

// ordering builds the "desc" key list from api-shape.md, skipping repeated keys.
func ordering(sortBy store.SortField) []orderKey {
	candidates := []orderKey{
		{sortKeyOf(sortBy), -1},
		{keyLastOccurring, -1},
		{keyLastCreation, -1},
		{keyCreation, -1},
		{keyTrackingNumber, +1},
	}
	var out []orderKey
	for _, c := range candidates {
		if !slices.ContainsFunc(out, func(o orderKey) bool { return o.key == c.key }) {
			out = append(out, c)
		}
	}
	return out
}

func (s *Store) ListTrackings(_ context.Context, f store.TrackingFilter) ([]model.TrackingWithLast, int, error) {
	wantNumber := textnorm.Fold(f.TrackingNumber)
	wantPOL := textnorm.Fold(f.POL)
	wantPOD := textnorm.Fold(f.POD)

	s.mu.RLock()
	matches := make([]*listRow, 0, len(s.trackings))
	for _, row := range s.trackings {
		t := row.tracking
		last := row.statuses[len(row.statuses)-1]
		r := &listRow{
			TrackingWithLast: model.TrackingWithLast{Tracking: t, LastStatus: last},
			number:           textnorm.Fold(t.TrackingNumber),
			pol:              textnorm.Fold(t.POL),
			pod:              textnorm.Fold(t.POD),
		}
		if !strings.Contains(r.number, wantNumber) ||
			!strings.HasPrefix(r.pol, wantPOL) ||
			!strings.HasPrefix(r.pod, wantPOD) ||
			t.CreationTime.Before(f.CreationTimeMin) || t.CreationTime.After(f.CreationTimeMax) ||
			last.OccurringTime.Before(f.LastStatusOccurringTimeMin) ||
			last.OccurringTime.After(f.LastStatusOccurringTimeMax) {
			continue
		}
		r.Tracking = cloneTracking(t)
		r.LastStatus = cloneStatus(last)
		matches = append(matches, r)
	}
	s.mu.RUnlock()

	keys := ordering(f.SortBy)
	flip := 1 // desc ordering as defined
	if !f.Desc {
		flip = -1 // asc is its exact reverse
	}
	slices.SortFunc(matches, func(a, b *listRow) int {
		for _, k := range keys {
			if c := compareKey(a, b, k.key); c != 0 {
				return c * k.sign * flip
			}
		}
		// Folded numbers can collide ("ABC" vs "abc"); keep the order total and deterministic.
		return cmp.Compare(a.TrackingNumber, b.TrackingNumber) * flip
	})

	total := len(matches)
	start := min(f.Offset, total)
	end := min(start+f.Limit, total)
	out := make([]model.TrackingWithLast, 0, end-start)
	for _, r := range matches[start:end] {
		out = append(out, r.TrackingWithLast)
	}
	return out, total, nil
}
