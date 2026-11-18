// Package store defines the persistence boundary. memstore implements it in memory
// today; a MySQL implementation will implement the same interface later.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

type SortField string

const (
	SortTrackingNumber          SortField = "trackingNumber"
	SortPOL                     SortField = "pol"
	SortPOD                     SortField = "pod"
	SortLastStatusOccurringTime SortField = "lastStatusOccurringTime"
	SortCreationTime            SortField = "creationTime"
)

// TrackingFilter is an already-validated GET /api/admin/tracking query. Every field is set
// (defaults applied by the caller). Text comparisons are case- and accent-insensitive.
type TrackingFilter struct {
	TrackingNumber string // contains
	POL            string // starts with
	POD            string // starts with

	CreationTimeMin, CreationTimeMax                       time.Time // inclusive
	LastStatusOccurringTimeMin, LastStatusOccurringTimeMax time.Time // inclusive

	SortBy SortField
	// Desc selects the "desc" ordering: SortBy DESC, lastStatusOccurringTime DESC,
	// lastStatus.creationTime DESC, creationTime DESC, trackingNumber ASC (repeated keys skipped).
	// Asc is its exact reverse.
	Desc bool

	Offset, Limit int
}

type Store interface {
	// Accounts. Emails are unique case-insensitively.
	CreateAccount(ctx context.Context, a model.Account) error // ErrConflict on duplicate email or id
	AccountByID(ctx context.Context, id string) (model.Account, error)
	AccountByEmail(ctx context.Context, email string) (model.Account, error)
	// DeleteAccount* also delete every session of the account.
	DeleteAccountByID(ctx context.Context, id string) error
	DeleteAccountByEmail(ctx context.Context, email string) error

	// Sessions.
	CreateSession(ctx context.Context, s model.Session) error
	SessionByID(ctx context.Context, id ids.SessionID) (model.Session, error)
	InvalidateSession(ctx context.Context, id ids.SessionID) error

	// Trackings. Statuses are always returned in model.StatusLess order.
	CreateTracking(ctx context.Context, t model.Tracking, first model.Status) error // ErrConflict on duplicate number
	Tracking(ctx context.Context, trackingNumber string) (model.Tracking, []model.Status, error)
	// ListTrackings returns one page of matches and the total number of matches.
	ListTrackings(ctx context.Context, f TrackingFilter) (rows []model.TrackingWithLast, total int, err error)
	AddStatus(ctx context.Context, trackingNumber string, s model.Status) error // ErrNotFound if no tracking
}
