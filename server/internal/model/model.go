// Package model holds the domain types shared by the store and the HTTP layer.
// All times are UTC with millisecond precision (see apitime.Normalize).
package model

import (
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type Account struct {
	ID             string // UUIDv7
	Email          string
	PasswordHashed string // Argon2id PHC string
	Name           string
	CreateTime     time.Time
	Role           Role
}

type Session struct {
	ID        ids.SessionID
	UserID    string
	StartTime time.Time
	EndsTime  time.Time
	IsValid   bool
}

// Active reports whether the session can authenticate a request at now.
func (s Session) Active(now time.Time) bool {
	return s.IsValid && now.Before(s.EndsTime)
}

// GoodStatusNames is the fixed, ordered list from requirements/api-shape.md.
var GoodStatusNames = []string{
	"Tracking created", "Ex-work", "Customs clearance at POL", "Arrived at POL",
	"Left POL", "Arrived at POT", "Left POT", "Arrived at POD", "Left POD",
	"Customs clearance at POD", "Inland trucking to Place of Delivery",
	"Arrived at Place of Delivery", "Finished",
}

var possibleStatuses = func() map[string]bool {
	m := make(map[string]bool, len(GoodStatusNames))
	for _, n := range GoodStatusNames {
		m[n] = true
	}
	return m
}()

func IsGoodStatusName(s string) bool { return possibleStatuses[s] }

const StatusTrackingCreated = "Tracking created"

type StatusType string

const (
	StatusGood StatusType = "good"
	StatusBad  StatusType = "bad"
)

type Status struct {
	ID            string
	Type          StatusType
	Status        string   // a GoodStatusNames entry when Type == good; any nonempty text when bad
	Warnings      []string // good only; never nil for good
	OccurringTime time.Time
	Location      string
	CreatorID     string
	CreationTime  time.Time
}

// StatusLess is the canonical status order: occurringTime ASC, ties by creationTime ASC,
// then id for determinism. The last status in this order is a tracking's last status.
func StatusLess(a, b Status) bool {
	if !a.OccurringTime.Equal(b.OccurringTime) {
		return a.OccurringTime.Before(b.OccurringTime)
	}
	if !a.CreationTime.Equal(b.CreationTime) {
		return a.CreationTime.Before(b.CreationTime)
	}
	return a.ID < b.ID
}

type Dimensions struct {
	LengthCm float64
	WidthCm  float64
	HeightCm float64
}

type PhysicalDetails struct {
	PcsCount   *int
	WeightKg   float64
	Dimensions Dimensions
}

type Tracking struct {
	TrackingNumber string
	Cont           string // "" = absent
	Seal           string // "" = absent
	POL            string
	POD            string
	Physical       *PhysicalDetails
	CreatorID      string
	CreationTime   time.Time
	CustomerID     string // "" = absent
}

// TrackingWithLast is a list row: the tracking plus its last status.
type TrackingWithLast struct {
	Tracking
	LastStatus Status
}
