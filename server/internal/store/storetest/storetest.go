// Package storetest is a contract test suite that every store.Store implementation must pass.
package storetest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
)

var base = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

func at(h int) time.Time { return base.Add(time.Duration(h) * time.Hour) }

func good(name string, occurring, created time.Time) model.Status {
	return model.Status{
		ID: ids.NewUUIDv7(created), Type: model.StatusGood, Status: name, Warnings: []string{},
		OccurringTime: occurring, Location: "x", CreatorID: "c", CreationTime: created,
	}
}

func everything(sortBy store.SortField, desc bool) store.TrackingFilter {
	return store.TrackingFilter{
		CreationTimeMax: at(10_000), LastStatusOccurringTimeMax: at(10_000),
		SortBy: sortBy, Desc: desc, Limit: 100,
	}
}

func numbers(rows []model.TrackingWithLast) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.TrackingNumber
	}
	return out
}

func mustCreate(t *testing.T, s store.Store, tr model.Tracking, statuses ...model.Status) {
	t.Helper()
	ctx := context.Background()
	if tr.POD == "" {
		tr.POD = "P"
	}
	if tr.CreatorID == "" {
		tr.CreatorID = "c"
	}
	if err := s.CreateTracking(ctx, tr, statuses[0]); err != nil {
		t.Fatal(err)
	}
	for _, st := range statuses[1:] {
		if err := s.AddStatus(ctx, tr.TrackingNumber, st); err != nil {
			t.Fatal(err)
		}
	}
}

// Run runs every contract test. newStore must return an empty store each call.
func Run(t *testing.T, newStore func(t *testing.T) store.Store) {
	t.Run("LastStatusAndTieBreaks", func(t *testing.T) { testLastStatus(t, newStore(t)) })
	t.Run("Pagination", func(t *testing.T) { testPagination(t, newStore(t)) })
	t.Run("RoundTrip", func(t *testing.T) { testRoundTrip(t, newStore(t)) })
	t.Run("Accounts", func(t *testing.T) { testAccounts(t, newStore(t)) })
	t.Run("Sessions", func(t *testing.T) { testSessions(t, newStore(t)) })
}

func testLastStatus(t *testing.T, s store.Store) {
	ctx := context.Background()
	// A: last status occurs at h10, entered at h11.
	mustCreate(t, s, model.Tracking{TrackingNumber: "A", POL: "Đà Nẵng", CreationTime: at(0)},
		good("Tracking created", at(0), at(0)), good("Left POL", at(10), at(11)))
	// B: same last occurring time as A but entered later, so B sorts before A in desc.
	mustCreate(t, s, model.Tracking{TrackingNumber: "B", POL: "Hải Phòng", CreationTime: at(1)},
		good("Tracking created", at(1), at(1)), good("Left POL", at(10), at(12)))
	// C: statuses added out of order; its last status is the h20 one.
	mustCreate(t, s, model.Tracking{TrackingNumber: "C", POL: "da lat", CreationTime: at(2)},
		good("Arrived at POD", at(20), at(21)), good("Tracking created", at(2), at(22)))

	_, st, err := s.Tracking(ctx, "C")
	if err != nil {
		t.Fatal(err)
	}
	if st[0].Status != "Tracking created" || st[len(st)-1].Status != "Arrived at POD" {
		t.Fatalf("C statuses out of order: %v", st)
	}

	desc, total, err := s.ListTrackings(ctx, everything(store.SortLastStatusOccurringTime, true))
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || !slices.Equal(numbers(desc), []string{"C", "B", "A"}) {
		t.Fatalf("desc = %v (total %d)", numbers(desc), total)
	}
	if desc[0].LastStatus.Status != "Arrived at POD" {
		t.Fatalf("C last status = %q", desc[0].LastStatus.Status)
	}
	asc, _, _ := s.ListTrackings(ctx, everything(store.SortLastStatusOccurringTime, false))
	if !slices.Equal(numbers(asc), []string{"A", "B", "C"}) {
		t.Fatalf("asc should be the exact reverse, got %v", numbers(asc))
	}

	f := everything(store.SortTrackingNumber, false)
	f.POL = "DA"
	rows, _, _ := s.ListTrackings(ctx, f)
	if !slices.Equal(numbers(rows), []string{"A", "C"}) {
		t.Fatalf("pol=DA -> %v", numbers(rows))
	}

	// Time bounds are inclusive.
	f = everything(store.SortTrackingNumber, false)
	f.LastStatusOccurringTimeMin, f.LastStatusOccurringTimeMax = at(10), at(10)
	rows, _, _ = s.ListTrackings(ctx, f)
	if !slices.Equal(numbers(rows), []string{"A", "B"}) {
		t.Fatalf("lastStatus in [h10,h10] -> %v", numbers(rows))
	}
}

func testPagination(t *testing.T, s store.Store) {
	for i, n := range []string{"T1", "T2", "T3", "T4", "T5"} {
		mustCreate(t, s, model.Tracking{TrackingNumber: n, POL: "X", CreationTime: at(i)}, good("Tracking created", at(i), at(i)))
	}
	f := everything(store.SortTrackingNumber, false)
	f.Offset, f.Limit = 2, 2
	rows, total, err := s.ListTrackings(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 || !slices.Equal(numbers(rows), []string{"T3", "T4"}) {
		t.Fatalf("page = %v total %d", numbers(rows), total)
	}
	f.Offset = 10
	rows, total, _ = s.ListTrackings(context.Background(), f)
	if total != 5 || len(rows) != 0 {
		t.Fatalf("past end = %v total %d", numbers(rows), total)
	}
}

func testRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	pcs := 12
	tr := model.Tracking{
		TrackingNumber: "RT-1", Cont: "MSKU1", Seal: "SL1", POL: "Hồ Chí Minh", POD: "Singapore",
		Physical:  &model.PhysicalDetails{PcsCount: &pcs, WeightKg: 840.5, Dimensions: model.Dimensions{LengthCm: 240, WidthCm: 120.25, HeightCm: 150}},
		CreatorID: "creator", CreationTime: at(0).Add(123 * time.Millisecond), CustomerID: "cust",
	}
	bad := model.Status{ID: "s-bad", Type: model.StatusBad, Status: "Lost", OccurringTime: at(3), Location: "Sea", CreatorID: "c2", CreationTime: at(4)}
	warned := good("Left POL", at(2), at(2))
	warned.Warnings = []string{"w1", "w2"}
	mustCreate(t, s, tr, good("Tracking created", at(0), at(0)), warned, bad)
	mustCreate(t, s, model.Tracking{TrackingNumber: "RT-2", POL: "X", CreationTime: at(1)}, good("Tracking created", at(1), at(1)))

	if err := s.CreateTracking(ctx, model.Tracking{TrackingNumber: "RT-1", POL: "X", POD: "Y", CreatorID: "c", CreationTime: at(0)},
		good("Tracking created", at(0), at(0))); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("duplicate tracking: %v", err)
	}
	if err := s.AddStatus(ctx, "nope", good("Finished", at(0), at(0))); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("status on missing tracking: %v", err)
	}
	if _, _, err := s.Tracking(ctx, "nope"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing tracking: %v", err)
	}

	got, st, err := s.Tracking(ctx, "RT-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Cont != "MSKU1" || got.Seal != "SL1" || got.CustomerID != "cust" || got.Physical == nil ||
		*got.Physical.PcsCount != 12 || got.Physical.WeightKg != 840.5 || got.Physical.Dimensions.WidthCm != 120.25 ||
		!got.CreationTime.Equal(tr.CreationTime) || got.CreationTime.Location() != time.UTC {
		t.Fatalf("tracking round trip: %+v", got)
	}
	if len(st) != 3 || st[0].Warnings == nil || len(st[0].Warnings) != 0 ||
		!slices.Equal(st[1].Warnings, []string{"w1", "w2"}) ||
		st[2].Type != model.StatusBad || st[2].Warnings != nil || st[2].CreatorID != "c2" {
		t.Fatalf("statuses round trip: %+v", st)
	}

	plain, _, _ := s.Tracking(ctx, "RT-2")
	if plain.Cont != "" || plain.Physical != nil || plain.CustomerID != "" {
		t.Fatalf("optional fields should be empty: %+v", plain)
	}

	rows, _, _ := s.ListTrackings(ctx, everything(store.SortTrackingNumber, false))
	if rows[0].LastStatus.ID != "s-bad" || rows[0].LastStatus.Type != model.StatusBad {
		t.Fatalf("list last status: %+v", rows[0].LastStatus)
	}
}

func testAccounts(t *testing.T, s store.Store) {
	ctx := context.Background()
	acc := model.Account{ID: "u1", Email: "A@x.test", PasswordHashed: "h", Name: "A", CreateTime: at(0), Role: model.RoleAdmin}
	if err := s.CreateAccount(ctx, acc); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccount(ctx, model.Account{ID: "u2", Email: "a@X.test", PasswordHashed: "h", Name: "B", CreateTime: at(0), Role: model.RoleUser}); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("case-insensitive duplicate email: %v", err)
	}
	got, err := s.AccountByEmail(ctx, " a@x.TEST ")
	if err != nil || got != acc {
		t.Fatalf("by email: %+v %v", got, err)
	}
	if got, err := s.AccountByID(ctx, "u1"); err != nil || got != acc {
		t.Fatalf("by id: %+v %v", got, err)
	}
	if _, err := s.AccountByID(ctx, "not-a-uuid"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing id: %v", err)
	}
	if err := s.DeleteAccountByEmail(ctx, "a@x.TEST"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteAccountByID(ctx, "u1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}

func testSessions(t *testing.T, s store.Store) {
	ctx := context.Background()
	acc := model.Account{ID: "u1", Email: "a@x.test", PasswordHashed: "h", Name: "A", CreateTime: at(0), Role: model.RoleUser}
	if err := s.CreateAccount(ctx, acc); err != nil {
		t.Fatal(err)
	}
	sess := model.Session{ID: ids.NewSessionID(), UserID: "u1", StartTime: at(0), EndsTime: at(720), IsValid: true}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatal(err)
	}
	got, err := s.SessionByID(ctx, sess.ID)
	if err != nil || got != sess {
		t.Fatalf("session round trip: %+v %v", got, err)
	}
	if err := s.InvalidateSession(ctx, sess.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.SessionByID(ctx, sess.ID); got.IsValid {
		t.Fatal("still valid after invalidate")
	}
	if err := s.InvalidateSession(ctx, ids.NewSessionID()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("invalidate missing: %v", err)
	}
	// Deleting the account deletes its sessions.
	if err := s.DeleteAccountByID(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionByID(ctx, sess.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("session survived account deletion: %v", err)
	}
}
