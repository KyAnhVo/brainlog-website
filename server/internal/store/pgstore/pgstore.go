// Package pgstore is the PostgreSQL implementation of store.Store, built on pgx.
package pgstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
	"github.com/KyAnhVo/brainlog-website/server/internal/textnorm"
)

type Store struct {
	pool *pgxpool.Pool
}

var _ store.Store = (*Store)(nil)

// Open connects using a libpq-style URL or DSN (e.g. postgresql://user:pass@host/db?sslmode=require),
// verifies the connection, and applies pending migrations.
func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// pgx error messages can echo the DSN, which holds the password. Don't wrap err.
		return nil, errors.New("DB_ADDR is not a valid PostgreSQL connection string")
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}
	if err := migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func utc(t time.Time) time.Time { return t.UTC() }

// ---------------------------------------------------------------------------
// Accounts
// ---------------------------------------------------------------------------

const accountCols = `id, email, password_hashed, name, account_create_time, role`

func scanAccount(row pgx.Row) (model.Account, error) {
	var a model.Account
	var role string
	err := row.Scan(&a.ID, &a.Email, &a.PasswordHashed, &a.Name, &a.CreateTime, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Account{}, store.ErrNotFound
	}
	if err != nil {
		return model.Account{}, err
	}
	a.CreateTime = utc(a.CreateTime)
	a.Role = model.Role(role)
	return a, nil
}

func (s *Store) CreateAccount(ctx context.Context, a model.Account) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO accounts (`+accountCols+`) VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.Email, a.PasswordHashed, a.Name, a.CreateTime, string(a.Role))
	if isUniqueViolation(err) {
		return store.ErrConflict
	}
	return err
}

func (s *Store) AccountByID(ctx context.Context, id string) (model.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx, `SELECT `+accountCols+` FROM accounts WHERE id = $1`, id))
}

func (s *Store) AccountByEmail(ctx context.Context, email string) (model.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx,
		`SELECT `+accountCols+` FROM accounts WHERE lower(email) = lower($1)`, strings.TrimSpace(email)))
}

// Sessions are removed by ON DELETE CASCADE.
func (s *Store) DeleteAccountByID(ctx context.Context, id string) error {
	return expectOne(s.pool.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, id))
}

func (s *Store) DeleteAccountByEmail(ctx context.Context, email string) error {
	return expectOne(s.pool.Exec(ctx, `DELETE FROM accounts WHERE lower(email) = lower($1)`, strings.TrimSpace(email)))
}

func expectOne(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

func (s *Store) CreateSession(ctx context.Context, sess model.Session) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (session_id, user_id, start_time, ends_time, is_valid) VALUES ($1, $2, $3, $4, $5)`,
		sess.ID[:], sess.UserID, sess.StartTime, sess.EndsTime, sess.IsValid)
	if isUniqueViolation(err) {
		return store.ErrConflict
	}
	return err
}

func (s *Store) SessionByID(ctx context.Context, id ids.SessionID) (model.Session, error) {
	sess := model.Session{ID: id}
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, start_time, ends_time, is_valid FROM sessions WHERE session_id = $1`, id[:],
	).Scan(&sess.UserID, &sess.StartTime, &sess.EndsTime, &sess.IsValid)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Session{}, store.ErrNotFound
	}
	if err != nil {
		return model.Session{}, err
	}
	sess.StartTime, sess.EndsTime = utc(sess.StartTime), utc(sess.EndsTime)
	return sess, nil
}

func (s *Store) InvalidateSession(ctx context.Context, id ids.SessionID) error {
	return expectOne(s.pool.Exec(ctx, `UPDATE sessions SET is_valid = false WHERE session_id = $1`, id[:]))
}

// ---------------------------------------------------------------------------
// Trackings
// ---------------------------------------------------------------------------

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func warningsParam(st model.Status) []string {
	if st.Type != model.StatusGood {
		return nil // NULL
	}
	if st.Warnings == nil {
		return []string{}
	}
	return st.Warnings
}

func insertStatus(ctx context.Context, tx pgx.Tx, trackingNumber string, st model.Status) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO statuses (id, tracking_number, status_type, status, warnings, occurring_time, location, creator_id, creation_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		st.ID, trackingNumber, string(st.Type), st.Status, warningsParam(st),
		st.OccurringTime, st.Location, st.CreatorID, st.CreationTime)
	return err
}

// refreshLastStatus recomputes the denormalised last status: greatest occurringTime,
// ties by greatest creationTime, then id (same as model.StatusLess).
func refreshLastStatus(ctx context.Context, tx pgx.Tx, trackingNumber string) error {
	_, err := tx.Exec(ctx, `
		UPDATE trackings t SET
			last_status_id = l.id,
			last_status_occurring_time = l.occurring_time,
			last_status_creation_time = l.creation_time
		FROM (
			SELECT id, occurring_time, creation_time FROM statuses
			WHERE tracking_number = $1
			ORDER BY occurring_time DESC, creation_time DESC, id COLLATE "C" DESC
			LIMIT 1
		) l
		WHERE t.tracking_number = $1`, trackingNumber)
	return err
}

func (s *Store) CreateTracking(ctx context.Context, t model.Tracking, first model.Status) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var pcs *int
		var weight, length, width, height *float64
		if p := t.Physical; p != nil {
			pcs = p.PcsCount
			weight, length, width, height = &p.WeightKg, &p.Dimensions.LengthCm, &p.Dimensions.WidthCm, &p.Dimensions.HeightCm
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO trackings (tracking_number, cont, seal, pol, pod,
				pcs_count, weight_kg, length_cm, width_cm, height_cm,
				creator_id, creation_time, customer_id,
				tracking_number_fold, pol_fold, pod_fold)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
			t.TrackingNumber, nullIfEmpty(t.Cont), nullIfEmpty(t.Seal), t.POL, t.POD,
			pcs, weight, length, width, height,
			t.CreatorID, t.CreationTime, nullIfEmpty(t.CustomerID),
			textnorm.Fold(t.TrackingNumber), textnorm.Fold(t.POL), textnorm.Fold(t.POD))
		if isUniqueViolation(err) {
			return store.ErrConflict
		}
		if err != nil {
			return err
		}
		if err := insertStatus(ctx, tx, t.TrackingNumber, first); err != nil {
			return err
		}
		return refreshLastStatus(ctx, tx, t.TrackingNumber)
	})
}

func (s *Store) AddStatus(ctx context.Context, trackingNumber string, st model.Status) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Lock the tracking so concurrent inserts can't race on the last-status refresh.
		var one int
		err := tx.QueryRow(ctx, `SELECT 1 FROM trackings WHERE tracking_number = $1 FOR UPDATE`, trackingNumber).Scan(&one)
		if errors.Is(err, pgx.ErrNoRows) {
			return store.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := insertStatus(ctx, tx, trackingNumber, st); err != nil {
			return err
		}
		return refreshLastStatus(ctx, tx, trackingNumber)
	})
}

// trackingCols selects a tracking with table alias t.
const trackingCols = `t.tracking_number, t.cont, t.seal, t.pol, t.pod,
	t.pcs_count, t.weight_kg, t.length_cm, t.width_cm, t.height_cm,
	t.creator_id, t.creation_time, t.customer_id`

// statusCols selects a status with table alias s.
const statusCols = `s.id, s.status_type, s.status, s.warnings, s.occurring_time, s.location, s.creator_id, s.creation_time`

type trackingScan struct {
	cont, seal, customer          *string
	pcs                           *int32
	weight, length, width, height *float64
}

func (ts *trackingScan) dest(t *model.Tracking) []any {
	return []any{&t.TrackingNumber, &ts.cont, &ts.seal, &t.POL, &t.POD,
		&ts.pcs, &ts.weight, &ts.length, &ts.width, &ts.height,
		&t.CreatorID, &t.CreationTime, &ts.customer}
}

func (ts *trackingScan) finish(t *model.Tracking) {
	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	t.Cont, t.Seal, t.CustomerID = deref(ts.cont), deref(ts.seal), deref(ts.customer)
	t.CreationTime = utc(t.CreationTime)
	if ts.weight != nil {
		p := &model.PhysicalDetails{
			WeightKg:   *ts.weight,
			Dimensions: model.Dimensions{LengthCm: *ts.length, WidthCm: *ts.width, HeightCm: *ts.height},
		}
		if ts.pcs != nil {
			n := int(*ts.pcs)
			p.PcsCount = &n
		}
		t.Physical = p
	}
}

type statusScan struct {
	statusType string
	warnings   []string
}

func (ss *statusScan) dest(st *model.Status) []any {
	return []any{&st.ID, &ss.statusType, &st.Status, &ss.warnings, &st.OccurringTime, &st.Location, &st.CreatorID, &st.CreationTime}
}

func (ss *statusScan) finish(st *model.Status) {
	st.Type = model.StatusType(ss.statusType)
	st.OccurringTime, st.CreationTime = utc(st.OccurringTime), utc(st.CreationTime)
	if st.Type == model.StatusGood {
		st.Warnings = ss.warnings
		if st.Warnings == nil {
			st.Warnings = []string{}
		}
	}
}

func (s *Store) Tracking(ctx context.Context, trackingNumber string) (model.Tracking, []model.Status, error) {
	var t model.Tracking
	var ts trackingScan
	err := s.pool.QueryRow(ctx, `SELECT `+trackingCols+` FROM trackings t WHERE t.tracking_number = $1`, trackingNumber).
		Scan(ts.dest(&t)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Tracking{}, nil, store.ErrNotFound
	}
	if err != nil {
		return model.Tracking{}, nil, err
	}
	ts.finish(&t)

	rows, err := s.pool.Query(ctx, `SELECT `+statusCols+` FROM statuses s WHERE s.tracking_number = $1
		ORDER BY s.occurring_time, s.creation_time, s.id COLLATE "C"`, trackingNumber)
	if err != nil {
		return model.Tracking{}, nil, err
	}
	defer rows.Close()
	var statuses []model.Status
	for rows.Next() {
		var st model.Status
		var ss statusScan
		if err := rows.Scan(ss.dest(&st)...); err != nil {
			return model.Tracking{}, nil, err
		}
		ss.finish(&st)
		statuses = append(statuses, st)
	}
	return t, statuses, rows.Err()
}

// Sort columns, all compared byte-wise (COLLATE "C") to match memstore's code-point order.
var sortColumn = map[store.SortField]string{
	store.SortTrackingNumber:          `t.tracking_number_fold COLLATE "C"`,
	store.SortPOL:                     `t.pol_fold COLLATE "C"`,
	store.SortPOD:                     `t.pod_fold COLLATE "C"`,
	store.SortLastStatusOccurringTime: `t.last_status_occurring_time`,
	store.SortCreationTime:            `t.creation_time`,
}

// orderBy builds the "desc" ordering from api-shape.md (repeated keys skipped), or its exact reverse.
func orderBy(sortBy store.SortField, desc bool) string {
	type key struct {
		col  string
		desc bool
	}
	candidates := []key{
		{sortColumn[sortBy], true},
		{sortColumn[store.SortLastStatusOccurringTime], true},
		{`t.last_status_creation_time`, true},
		{sortColumn[store.SortCreationTime], true},
		{sortColumn[store.SortTrackingNumber], false},
	}
	var parts []string
	seen := map[string]bool{}
	for _, k := range candidates {
		if seen[k.col] {
			continue
		}
		seen[k.col] = true
		parts = append(parts, k.col+direction(k.desc == desc)) // asc flips every key
	}
	// Folded numbers can collide; the raw number makes the order total (as memstore does).
	parts = append(parts, `t.tracking_number COLLATE "C"`+direction(!desc))
	return strings.Join(parts, ", ")
}

func direction(desc bool) string {
	if desc {
		return " DESC"
	}
	return " ASC"
}

const listWhere = `
	strpos(t.tracking_number_fold, $1) > 0
	AND starts_with(t.pol_fold, $2)
	AND starts_with(t.pod_fold, $3)
	AND t.creation_time BETWEEN $4 AND $5
	AND t.last_status_occurring_time BETWEEN $6 AND $7`

func (s *Store) ListTrackings(ctx context.Context, f store.TrackingFilter) ([]model.TrackingWithLast, int, error) {
	if _, ok := sortColumn[f.SortBy]; !ok {
		return nil, 0, fmt.Errorf("unknown sort field %q", f.SortBy)
	}
	args := []any{
		textnorm.Fold(f.TrackingNumber), textnorm.Fold(f.POL), textnorm.Fold(f.POD),
		f.CreationTimeMin, f.CreationTimeMax, f.LastStatusOccurringTimeMin, f.LastStatusOccurringTimeMax,
	}

	var out []model.TrackingWithLast
	var total int
	// One read-only snapshot so the page and the count agree.
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM trackings t WHERE`+listWhere, args...).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT `+trackingCols+`, `+statusCols+`
			FROM trackings t JOIN statuses s ON s.id = t.last_status_id
			WHERE`+listWhere+`
			ORDER BY `+orderBy(f.SortBy, f.Desc)+`
			LIMIT $8 OFFSET $9`,
			append(args, f.Limit, f.Offset)...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r model.TrackingWithLast
			var ts trackingScan
			var ss statusScan
			if err := rows.Scan(append(ts.dest(&r.Tracking), ss.dest(&r.LastStatus)...)...); err != nil {
				return err
			}
			ts.finish(&r.Tracking)
			ss.finish(&r.LastStatus)
			out = append(out, r)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
