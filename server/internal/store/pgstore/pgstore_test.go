package pgstore

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/seed"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
	"github.com/KyAnhVo/brainlog-website/server/internal/store/memstore"
	"github.com/KyAnhVo/brainlog-website/server/internal/store/storetest"
)

// These tests need a disposable PostgreSQL database:
//
//	TEST_DATABASE_URL=postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable go test ./...
//
// Each test runs in its own freshly created schema, which is dropped afterwards.
// Never point this at a database that holds real data.
func testDSN(t *testing.T) string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL tests")
	}
	return dsn
}

// newSchemaStore opens a Store whose tables live in a new, empty schema.
func newSchemaStore(t *testing.T) *Store {
	t.Helper()
	dsn := testDSN(t)
	ctx := context.Background()
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		_ = admin.Close(context.Background())
	})

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	s, err := Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestContract(t *testing.T) {
	testDSN(t)
	storetest.Run(t, func(t *testing.T) store.Store { return newSchemaStore(t) })
}

func TestMigrateIsIdempotent(t *testing.T) {
	s := newSchemaStore(t)
	if err := migrate(context.Background(), s.pool); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

// TestMatchesMemstore loads the full seed into both stores and requires identical results
// for every sort field, direction, and a spread of filters.
func TestMatchesMemstore(t *testing.T) {
	pg := newSchemaStore(t)
	mem := memstore.New()
	ctx := context.Background()
	for _, s := range []store.Store{pg, mem} {
		if _, err := seed.Load(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	if loaded, err := seed.Load(ctx, pg); err != nil || loaded {
		t.Fatalf("re-seeding should be a no-op: loaded=%v err=%v", loaded, err)
	}

	epoch := time.Unix(0, 0).UTC()
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	filters := []store.TrackingFilter{
		{},
		{POL: "da"},
		{POL: "HO CHI"},
		{POD: "hai"},
		{TrackingNumber: "2026-001"},
		{TrackingNumber: "FAIL"},
		{TrackingNumber: "zzz"},
		{CreationTimeMin: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)},
		{LastStatusOccurringTimeMax: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)},
	}
	sorts := []store.SortField{
		store.SortTrackingNumber, store.SortPOL, store.SortPOD, store.SortLastStatusOccurringTime, store.SortCreationTime,
	}
	for _, base := range filters {
		for _, sb := range sorts {
			for _, desc := range []bool{true, false} {
				for _, page := range []struct{ off, lim int }{{0, 100}, {0, 10}, {10, 10}} {
					f := base
					if f.CreationTimeMax.IsZero() {
						f.CreationTimeMax = now
					}
					if f.LastStatusOccurringTimeMax.IsZero() {
						f.LastStatusOccurringTimeMax = now
					}
					if f.CreationTimeMin.IsZero() {
						f.CreationTimeMin = epoch
					}
					if f.LastStatusOccurringTimeMin.IsZero() {
						f.LastStatusOccurringTimeMin = epoch
					}
					f.SortBy, f.Desc, f.Offset, f.Limit = sb, desc, page.off, page.lim

					want, wantTotal, err := mem.ListTrackings(ctx, f)
					if err != nil {
						t.Fatal(err)
					}
					got, gotTotal, err := pg.ListTrackings(ctx, f)
					if err != nil {
						t.Fatal(err)
					}
					name := fmt.Sprintf("%+v", f)
					if gotTotal != wantTotal {
						t.Fatalf("%s: total pg=%d mem=%d", name, gotTotal, wantTotal)
					}
					if g, w := describe(got), describe(want); !slices.Equal(g, w) {
						t.Fatalf("%s:\n pg  %v\n mem %v", name, g, w)
					}
				}
			}
		}
	}

	// Full histories match too (statuses have random ids, so compare everything but the id).
	all, _, _ := mem.ListTrackings(ctx, store.TrackingFilter{CreationTimeMax: now, LastStatusOccurringTimeMax: now, SortBy: store.SortTrackingNumber, Limit: 100})
	for _, row := range all {
		_, ws, _ := mem.Tracking(ctx, row.TrackingNumber)
		_, gs, err := pg.Tracking(ctx, row.TrackingNumber)
		if err != nil || len(gs) != len(ws) {
			t.Fatalf("%s: %v (%d vs %d statuses)", row.TrackingNumber, err, len(gs), len(ws))
		}
		for i := range ws {
			if ws[i].Status != gs[i].Status || !ws[i].OccurringTime.Equal(gs[i].OccurringTime) ||
				!ws[i].CreationTime.Equal(gs[i].CreationTime) || !slices.Equal(ws[i].Warnings, gs[i].Warnings) {
				t.Fatalf("%s status %d: pg %+v mem %+v", row.TrackingNumber, i, gs[i], ws[i])
			}
		}
	}
}

// describe summarises rows without the status ids, which are random per seed run.
func describe(rows []model.TrackingWithLast) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		ls := r.LastStatus
		out[i] = strings.Join([]string{
			r.TrackingNumber, r.POL, r.POD, r.CreationTime.Format(time.RFC3339Nano),
			string(ls.Type), ls.Status, ls.OccurringTime.Format(time.RFC3339Nano), ls.CreationTime.Format(time.RFC3339Nano),
			strings.Join(ls.Warnings, ";"),
		}, "|")
	}
	return out
}
