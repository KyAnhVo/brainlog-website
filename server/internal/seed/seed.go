// Package seed loads the mock dataset into a store. It mirrors
// admin-panel-client/src/mocks/data.ts and is documented in devnote/server-mock.md.
package seed

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/password"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
)

const (
	Alice   = "0199a3f0-1c2e-7a41-9b3e-5d2f8c1a0e01"
	Binh    = "0199a3f0-4d7b-7c12-8f60-2b9e4a7d1c02"
	Chloe   = "0199a3f1-0a93-7e55-a1c4-8e3f6b2d9a03"
	Dung    = "0199a3f1-77c2-7f08-b2d1-4c5e9f1a6b04"
	Deleted = "01989e2c-5b10-7d3a-9e77-0f1a2b3c4d99" // created old data, then deleted: no account row
)

// t parses "2026-10-15 09:11" as UTC.
func t(s string) time.Time {
	v, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		panic(err)
	}
	return v
}

type account struct {
	id, email, password, name, created string
	role                               model.Role
}

var accounts = []account{
	{Alice, "admin@brainlog.test", "Admin123!", "Alice Nguyen", "2026-01-05 08:00", model.RoleAdmin},
	{Binh, "staff@brainlog.test", "Staff123!", "Bình Trần", "2026-02-11 02:30", model.RoleUser},
	{Chloe, "ops@brainlog.test", "Ops12345!", "Chloé Martin", "2026-03-20 14:15", model.RoleUser},
	{Dung, "dung@brainlog.test", "Dung1234!", "Dũng Phạm", "2026-06-01 09:45", model.RoleAdmin},
}

// step is one status in a history. status is a good status name, or "!text" for a bad status.
// creator "" = the tracking's creator. delay is how long after occurring it was entered (default 15m).
type step struct {
	status   string
	at       string
	location string
	warnings []string
	creator  string
	delay    *time.Duration
}

func d(x time.Duration) *time.Duration { return &x }
func pcs(n int) *int                   { return &n }

func phys(pcsCount *int, kg, l, w, h float64) *model.PhysicalDetails {
	return &model.PhysicalDetails{PcsCount: pcsCount, WeightKg: kg, Dimensions: model.Dimensions{LengthCm: l, WidthCm: w, HeightCm: h}}
}

const (
	vnHCM = "Hồ Chí Minh, VN"
	vnHP  = "Hải Phòng, VN"
	vnDN  = "Đà Nẵng, VN"
	sg    = "Singapore, SG"
	kr    = "Busan, KR"
	cn    = "Shanghai, CN"
	jp    = "Tokyo, JP"
	us    = "Los Angeles, US"
	nl    = "Rotterdam, NL"
	de    = "Hamburg, DE"
	my    = "Port Klang, MY"
	th    = "Laem Chabang, TH"
)

type tracking struct {
	model.Tracking
	created string
	steps   []step
}

var zero = d(0)

var trackings = []tracking{
	{model.Tracking{TrackingNumber: "BLJ-good-code", Cont: "MSKU1234567", Seal: "SL998877", POL: vnHP, POD: sg,
		Physical: phys(pcs(12), 840.5, 240, 120, 150), CreatorID: Alice, CustomerID: "customer-1"}, "2026-10-15 09:11", []step{
		{"Tracking created", "2026-10-15 09:11", vnHP, nil, Alice, zero},
		{"Customs clearance at POL", "2026-10-15 09:14", vnHP, nil, "", nil},
		{"Arrived at POL", "2026-10-20 09:15", vnHP, nil, Binh, nil},
		{"Left POL", "2026-10-21 14:30", vnHP, nil, Binh, nil},
		{"Arrived at POD", "2026-10-23 08:00", sg, nil, Chloe, nil},
		{"Customs clearance at POD", "2026-10-24 11:45", sg, []string{"Delayed by customs inspection"}, Chloe, d(10 * time.Minute)},
		// Same occurringTime as the previous status, entered later: it is the last status.
		{"Inland trucking to Place of Delivery", "2026-10-24 11:45", sg, nil, Chloe, d(20 * time.Minute)},
	}},
	{model.Tracking{TrackingNumber: "BLJ-good-code-fail-shipment", POL: vnDN, POD: kr, CreatorID: Alice}, "2026-10-15 09:11", []step{
		{"Tracking created", "2026-10-15 09:11", vnDN, nil, Alice, zero},
		{"Arrived at POL", "2026-10-20 09:15", vnDN, nil, Binh, nil},
		{"Customs clearance at POL", "2026-10-21 10:00", vnDN, []string{"Additional documents requested by customs"}, Binh, nil},
		{"!Rejected by customs: prohibited item", "2026-10-22 08:00", vnDN, nil, Alice, nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0001", Cont: "TGHU8821340", Seal: "HL220031", POL: vnHCM, POD: us,
		Physical: phys(pcs(40), 18250, 1203, 235, 239), CreatorID: Binh, CustomerID: "customer-2"}, "2026-09-02 01:05", []step{
		{"Tracking created", "2026-09-02 01:05", vnHCM, nil, Binh, zero},
		{"Ex-work", "2026-09-03 07:00", "Bình Dương, VN", nil, "", nil},
		{"Customs clearance at POL", "2026-09-05 03:20", vnHCM, nil, "", nil},
		{"Arrived at POL", "2026-09-05 10:00", vnHCM, nil, "", nil},
		{"Left POL", "2026-09-06 22:40", vnHCM, nil, "", nil},
		{"Arrived at POT", "2026-09-09 05:10", "Kaohsiung, TW", nil, Chloe, nil},
		{"Left POT", "2026-09-10 18:00", "Kaohsiung, TW", nil, Chloe, nil},
		{"Arrived at POD", "2026-09-28 16:30", us, nil, Chloe, nil},
		{"Customs clearance at POD", "2026-09-30 19:00", us, nil, Chloe, nil},
		{"Inland trucking to Place of Delivery", "2026-10-01 15:00", "Ontario, CA, US", nil, Chloe, nil},
		{"Arrived at Place of Delivery", "2026-10-02 21:15", "Ontario, CA, US", nil, Chloe, nil},
		{"Finished", "2026-10-03 00:00", "Ontario, CA, US", nil, Chloe, nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0002", Cont: "CMAU5530912", POL: vnHCM, POD: nl,
		Physical: phys(nil, 9600, 590, 235, 239), CreatorID: Binh}, "2026-09-14 03:00", []step{
		{"Tracking created", "2026-09-14 03:00", vnHCM, nil, Binh, zero},
		{"Arrived at POL", "2026-09-16 08:00", vnHCM, nil, "", nil},
		{"Left POL", "2026-09-18 12:00", vnHCM, nil, "", nil},
		{"Arrived at POT", "2026-09-22 06:00", sg, []string{"Transshipment window extended by 48h"}, "", nil},
		{"Left POT", "2026-09-25 09:30", sg, nil, "", nil},
		{"Arrived at POD", "2026-10-19 04:45", nl, nil, Chloe, nil},
		{"Customs clearance at POD", "2026-10-21 11:00", nl, nil, Chloe, nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0003", POL: vnDN, POD: jp,
		Physical: phys(pcs(3), 120.25, 80, 60, 45), CreatorID: Chloe, CustomerID: "customer-3"}, "2026-10-01 06:30", []step{
		{"Tracking created", "2026-10-01 06:30", vnDN, nil, Chloe, zero},
		{"Ex-work", "2026-10-02 02:00", "Hội An, VN", nil, "", nil},
		{"Arrived at POL", "2026-10-03 09:00", vnDN, nil, "", nil},
		{"Left POL", "2026-10-04 13:00", vnDN, nil, "", nil},
		{"Arrived at POD", "2026-10-08 01:00", jp, nil, "", nil},
		{"Customs clearance at POD", "2026-10-08 07:30", jp, nil, "", nil},
		{"Arrived at Place of Delivery", "2026-10-09 05:00", "Yokohama, JP", nil, "", nil},
		{"Finished", "2026-10-09 05:30", "Yokohama, JP", nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0004", Cont: "OOLU7781203", Seal: "OO551200", POL: cn, POD: vnHP, CreatorID: Alice}, "2026-10-05 10:00", []step{
		{"Tracking created", "2026-10-05 10:00", cn, nil, Alice, zero},
		{"Arrived at POL", "2026-10-06 04:00", cn, nil, "", nil},
		{"Left POL", "2026-10-07 20:00", cn, nil, "", nil},
		{"!Vessel diverted due to typhoon", "2026-10-09 03:00", "South China Sea", nil, Dung, nil},
		{"Arrived at POD", "2026-10-14 12:00", vnHP, []string{"Arrived 4 days late"}, Dung, nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0005", POL: sg, POD: vnHCM,
		Physical: phys(pcs(1), 2.5, 30, 20, 10), CreatorID: Dung}, "2026-10-10 02:15", []step{
		{"Tracking created", "2026-10-10 02:15", sg, nil, Dung, zero},
		{"Arrived at POL", "2026-10-11 03:00", sg, nil, "", nil},
		{"Left POL", "2026-10-11 15:00", sg, nil, "", nil},
		{"Arrived at POD", "2026-10-13 06:00", vnHCM, nil, "", nil},
		{"Customs clearance at POD", "2026-10-14 02:00", vnHCM, []string{"Duty unpaid, awaiting consignee"}, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0006", Cont: "HLXU2209981", POL: vnHP, POD: de,
		Physical: phys(pcs(220), 21400, 1203, 235, 269), CreatorID: Deleted, CustomerID: "customer-4"}, "2026-08-20 07:00", []step{
		{"Tracking created", "2026-08-20 07:00", vnHP, nil, Deleted, zero},
		{"Arrived at POL", "2026-08-22 05:00", vnHP, nil, Deleted, nil},
		{"Left POL", "2026-08-23 10:00", vnHP, nil, Deleted, nil},
		{"Arrived at POT", "2026-08-27 08:00", sg, nil, Deleted, nil},
		{"Left POT", "2026-08-28 23:00", sg, nil, Deleted, nil},
		{"Arrived at POD", "2026-09-24 06:00", de, nil, Alice, nil},
		{"Customs clearance at POD", "2026-09-25 09:00", de, nil, Alice, nil},
		{"Inland trucking to Place of Delivery", "2026-09-26 05:00", "Berlin, DE", nil, Alice, nil},
		{"Arrived at Place of Delivery", "2026-09-27 13:00", "Berlin, DE", nil, Alice, nil},
		{"Finished", "2026-09-27 14:00", "Berlin, DE", nil, Alice, nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0007", POL: my, POD: vnDN, CreatorID: Binh}, "2026-10-25 04:40", []step{
		{"Tracking created", "2026-10-25 04:40", my, nil, Binh, zero},
		{"Arrived at POL", "2026-10-27 02:00", my, nil, "", nil},
		{"Left POL", "2026-10-28 06:00", my, nil, "", nil},
		{"Arrived at POD", "2026-10-31 10:00", vnDN, nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0008", Cont: "EGLV1100293", Seal: "EG093311", POL: th, POD: vnHCM,
		Physical: phys(pcs(18), 4410, 590, 235, 239), CreatorID: Chloe}, "2026-10-28 08:20", []step{
		{"Tracking created", "2026-10-28 08:20", th, nil, Chloe, zero},
		{"Customs clearance at POL", "2026-10-29 03:00", th, nil, "", nil},
		{"!Container damaged during loading", "2026-10-30 07:10", th, nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0009", POL: vnHCM, POD: sg,
		Physical: phys(pcs(6), 310, 120, 100, 90), CreatorID: Alice, CustomerID: "customer-1"}, "2026-11-02 01:00", []step{
		{"Tracking created", "2026-11-02 01:00", vnHCM, nil, Alice, zero},
		{"Ex-work", "2026-11-03 02:00", "Long An, VN", nil, "", nil},
		{"Arrived at POL", "2026-11-04 04:30", vnHCM, nil, Binh, nil},
		{"Left POL", "2026-11-05 11:00", vnHCM, nil, Binh, nil},
		{"Arrived at POD", "2026-11-07 09:00", sg, nil, Binh, nil},
		// Entered 2 days late (backfilled).
		{"Customs clearance at POD", "2026-11-08 03:00", sg, nil, Chloe, d(48 * time.Hour)},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0010", Cont: "MSCU4432019", POL: kr, POD: vnHP, CreatorID: Dung}, "2026-11-03 05:00", []step{
		{"Tracking created", "2026-11-03 05:00", kr, nil, Dung, zero},
		{"Arrived at POL", "2026-11-04 01:00", kr, nil, "", nil},
		{"Left POL", "2026-11-05 08:00", kr, nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0011", POL: vnHP, POD: cn,
		Physical: phys(nil, 1500, 300, 200, 180), CreatorID: Binh}, "2026-11-05 03:30", []step{
		{"Tracking created", "2026-11-05 03:30", vnHP, nil, Binh, zero},
		{"Customs clearance at POL", "2026-11-06 02:00", vnHP, []string{"Inspection selected (red channel)", "Missing packing list"}, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0012", Cont: "ZIMU3319028", Seal: "ZM112233", POL: vnHCM, POD: us,
		Physical: phys(pcs(960), 19800, 1203, 235, 269), CreatorID: Alice, CustomerID: "customer-2"}, "2026-11-06 06:00", []step{
		{"Tracking created", "2026-11-06 06:00", vnHCM, nil, Alice, zero},
		{"Ex-work", "2026-11-07 01:00", "Đồng Nai, VN", nil, "", nil},
		{"Arrived at POL", "2026-11-08 08:00", vnHCM, nil, "", nil},
		{"Left POL", "2026-11-10 17:00", vnHCM, nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0013", POL: jp, POD: vnDN, CreatorID: Chloe}, "2026-11-08 00:30", []step{
		{"Tracking created", "2026-11-08 00:30", jp, nil, Chloe, zero},
		{"Arrived at POL", "2026-11-09 03:00", jp, nil, "", nil},
		{"!Shipment held: consignee unreachable", "2026-11-11 08:00", jp, nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0014", POL: vnDN, POD: sg,
		Physical: phys(pcs(2), 48, 60, 40, 40), CreatorID: Dung}, "2026-11-10 09:00", []step{
		{"Tracking created", "2026-11-10 09:00", vnDN, nil, Dung, zero},
		{"Arrived at POL", "2026-11-11 02:00", vnDN, nil, "", nil},
		{"Left POL", "2026-11-12 05:00", vnDN, nil, "", nil},
		{"Arrived at POD", "2026-11-14 07:00", sg, nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0015", Cont: "TCLU6650021", POL: nl, POD: vnHCM, CreatorID: Alice}, "2026-10-12 12:00", []step{
		{"Tracking created", "2026-10-12 12:00", nl, nil, Alice, zero},
		{"Arrived at POL", "2026-10-14 08:00", nl, nil, "", nil},
		{"Left POL", "2026-10-15 22:00", nl, nil, "", nil},
		{"Arrived at POT", "2026-11-01 04:00", "Colombo, LK", nil, "", nil},
		{"Left POT", "2026-11-03 10:00", "Colombo, LK", nil, "", nil},
		{"Arrived at POD", "2026-11-12 06:00", vnHCM, nil, Binh, nil},
		{"Customs clearance at POD", "2026-11-13 03:00", vnHCM, nil, Binh, nil},
		{"Inland trucking to Place of Delivery", "2026-11-14 01:00", "Bình Dương, VN", nil, Binh, nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0016", POL: vnHP, POD: kr, CreatorID: Binh}, "2026-11-12 02:00", []step{
		{"Tracking created", "2026-11-12 02:00", vnHP, nil, Binh, zero},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0017", Cont: "SEGU9001234", Seal: "SE776655", POL: sg, POD: de,
		Physical: phys(pcs(75), 12040.75, 1203, 235, 239), CreatorID: Chloe}, "2026-11-13 04:00", []step{
		{"Tracking created", "2026-11-13 04:00", sg, nil, Chloe, zero},
		{"Arrived at POL", "2026-11-14 06:00", sg, nil, "", nil},
		{"Left POL", "2026-11-15 13:00", sg, nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0018", POL: vnHCM, POD: th, CreatorID: Dung, CustomerID: "customer-3"}, "2026-11-14 08:00", []step{
		{"Tracking created", "2026-11-14 08:00", vnHCM, nil, Dung, zero},
		{"Ex-work", "2026-11-15 02:00", "Thủ Đức, VN", []string{"Pickup rescheduled once"}, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0019", POL: cn, POD: vnHCM,
		Physical: phys(pcs(10), 560, 200, 120, 100), CreatorID: Alice}, "2026-11-15 03:00", []step{
		{"Tracking created", "2026-11-15 03:00", cn, nil, Alice, zero},
		{"Arrived at POL", "2026-11-16 01:00", cn, nil, "", nil},
	}},
	{model.Tracking{TrackingNumber: "BLJ-2026-0020", POL: vnDN, POD: my, CreatorID: Binh}, "2026-11-16 07:30", []step{
		{"Tracking created", "2026-11-16 07:30", vnDN, nil, Binh, zero},
	}},
	{model.Tracking{TrackingNumber: "BLX-SAMPLE-0021", POL: us, POD: vnHP, CreatorID: Chloe}, "2026-09-20 18:00", []step{
		{"Tracking created", "2026-09-20 18:00", us, nil, Chloe, zero},
		{"Arrived at POL", "2026-09-22 16:00", us, nil, "", nil},
		{"Left POL", "2026-09-24 02:00", us, nil, "", nil},
		{"Arrived at POT", "2026-10-08 03:00", "Kaohsiung, TW", nil, "", nil},
		{"!Lost in transit: investigation opened", "2026-10-12 05:00", "Kaohsiung, TW", nil, Dung, nil},
	}},
	{model.Tracking{TrackingNumber: "BLX-SAMPLE-0022", POL: kr, POD: sg, CreatorID: Dung}, "2026-10-18 01:00", []step{
		{"Tracking created", "2026-10-18 01:00", kr, nil, Dung, zero},
		{"Arrived at POL", "2026-10-19 05:00", kr, nil, "", nil},
		{"Left POL", "2026-10-20 09:00", kr, nil, "", nil},
		{"Arrived at POD", "2026-10-24 05:00", sg, nil, "", nil},
		{"Customs clearance at POD", "2026-10-24 12:00", sg, nil, "", nil},
		{"Arrived at Place of Delivery", "2026-10-25 08:00", "Jurong, SG", nil, "", nil},
		{"Finished", "2026-10-25 09:00", "Jurong, SG", nil, "", nil},
	}},
}

func (st step) build(tr model.Tracking) model.Status {
	occurring := t(st.at)
	delay := 15 * time.Minute
	if st.delay != nil {
		delay = *st.delay
	}
	creator := st.creator
	if creator == "" {
		creator = tr.CreatorID
	}
	created := occurring.Add(delay)
	s := model.Status{
		ID:            ids.NewUUIDv7(created),
		OccurringTime: occurring,
		Location:      st.location,
		CreatorID:     creator,
		CreationTime:  created,
	}
	if text, bad := strings.CutPrefix(st.status, "!"); bad {
		s.Type, s.Status = model.StatusBad, text
	} else {
		if !model.IsGoodStatusName(st.status) {
			panic(fmt.Sprintf("seed: unknown good status %q", st.status))
		}
		s.Type, s.Status, s.Warnings = model.StatusGood, st.status, append([]string{}, st.warnings...)
	}
	return s
}

// Load inserts the mock accounts and trackings. Hashing the passwords takes a moment.
// If the seed admin account already exists (a persistent store seeded on an earlier run),
// it does nothing and reports loaded=false.
func Load(ctx context.Context, s store.Store) (loaded bool, err error) {
	if _, err := s.AccountByID(ctx, Alice); err == nil {
		return false, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return false, err
	}
	for _, a := range accounts {
		err := s.CreateAccount(ctx, model.Account{
			ID:             a.id,
			Email:          a.email,
			PasswordHashed: password.Hash(a.password),
			Name:           a.name,
			CreateTime:     t(a.created),
			Role:           a.role,
		})
		if err != nil {
			return false, fmt.Errorf("seed account %s: %w", a.email, err)
		}
	}
	for _, tr := range trackings {
		m := tr.Tracking
		m.CreationTime = t(tr.created)
		if err := s.CreateTracking(ctx, m, tr.steps[0].build(m)); err != nil {
			return false, fmt.Errorf("seed tracking %s: %w", m.TrackingNumber, err)
		}
		for _, st := range tr.steps[1:] {
			if err := s.AddStatus(ctx, m.TrackingNumber, st.build(m)); err != nil {
				return false, fmt.Errorf("seed status %s: %w", m.TrackingNumber, err)
			}
		}
	}
	return true, nil
}
