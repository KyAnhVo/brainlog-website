package api

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/KyAnhVo/brainlog-website/server/internal/apitime"
	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
)

// GET /api/tracking/{trackingNumber} (customer-facing, public)
func (s *Server) handleCustomerTracking(w http.ResponseWriter, r *http.Request) {
	t, statuses, err := s.store.Tracking(r.Context(), r.PathValue("trackingNumber"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "No shipment found for this tracking code")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	res := trackingResponse{Details: toTrackingBase(t), Statuses: make([]statusCustomerDTO, len(statuses))}
	for i, st := range statuses {
		res.Statuses[i] = toStatusCustomer(st)
	}
	writeJSON(w, http.StatusOK, res)
}

// GET /api/admin/tracking
func (s *Server) handleListTrackings(w http.ResponseWriter, r *http.Request) {
	q, err := parseListQuery(r.URL.Query(), s.now())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rows, total, err := s.store.ListTrackings(r.Context(), q.filter)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	pageCount := (total + q.pageSize - 1) / q.pageSize
	if pageCount < q.page && !(pageCount == 0 && q.page == 1) {
		writeError(w, http.StatusBadRequest, "page is past the last page")
		return
	}
	res := trackingListResponse{
		Trackings: make([]trackingDTO, len(rows)),
		Page:      q.page,
		PageCount: pageCount,
		PageSize:  q.pageSize,
	}
	for i, row := range rows {
		res.Trackings[i] = trackingDTO{
			trackingDetailsDTO: toTrackingDetails(row.Tracking),
			LastStatus:         toStatus(row.LastStatus),
		}
	}
	writeJSON(w, http.StatusOK, res)
}

// GET /api/admin/tracking/{trackingNumber}
func (s *Server) handleAdminTracking(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("trackingNumber")
	t, statuses, err := s.store.Tracking(r.Context(), number)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "No tracking found with number "+number)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	res := adminTrackingResponse{Details: toTrackingDetails(t), Statuses: make([]statusDTO, len(statuses))}
	for i, st := range statuses {
		res.Statuses[i] = toStatus(st)
	}
	writeJSON(w, http.StatusOK, res)
}

type statusCreateRequest struct {
	StatusType    string    `json:"statusType"`
	Status        string    `json:"status"`
	Warnings      *[]string `json:"warnings"`
	OccurringTime string    `json:"occurringTime"`
	Location      string    `json:"location"`
}

const (
	maxStatusTextLen = 500
	maxLocationLen   = 200
	maxWarnings      = 20
)

// POST /api/admin/tracking/{trackingNumber}/status
func (s *Server) handleAddStatus(w http.ResponseWriter, r *http.Request) {
	var req statusCreateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	now := s.now()
	st, msg := validateStatusCreate(req, now)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	st.ID = ids.NewUUIDv7(now)
	st.CreatorID = currentAccount(r.Context()).ID
	st.CreationTime = now

	number := r.PathValue("trackingNumber")
	err := s.store.AddStatus(r.Context(), number, st)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "No tracking found with number "+number)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toStatus(st))
}

// validateStatusCreate returns the status to store, or a client-facing error message.
// The caller fills in ID, CreatorID and CreationTime.
func validateStatusCreate(req statusCreateRequest, now time.Time) (model.Status, string) {
	occurring, err := apitime.Parse(req.OccurringTime)
	if err != nil {
		return model.Status{}, "occurringTime is not a valid time (yyyy-mm-ddThh:mm:ss.xxxZ)"
	}
	// "Occurring time is always in the past", with the usual clock-skew allowance.
	if occurring.After(now.Add(AcceptedBuffer)) {
		return model.Status{}, "occurringTime must not be in the future"
	}
	location := strings.TrimSpace(req.Location)
	if location == "" {
		return model.Status{}, "location must not be empty"
	}
	if utf8.RuneCountInString(location) > maxLocationLen {
		return model.Status{}, "location is too long"
	}
	st := model.Status{OccurringTime: occurring, Location: location}

	switch model.StatusType(req.StatusType) {
	case model.StatusGood:
		if !model.IsGoodStatusName(req.Status) {
			return model.Status{}, "status is not one of the good status names"
		}
		warnings := []string{}
		if req.Warnings != nil {
			if len(*req.Warnings) > maxWarnings {
				return model.Status{}, "too many warnings"
			}
			for _, w := range *req.Warnings {
				w = strings.TrimSpace(w)
				if w == "" {
					return model.Status{}, "warnings must not contain empty strings"
				}
				if utf8.RuneCountInString(w) > maxStatusTextLen {
					return model.Status{}, "a warning is too long"
				}
				warnings = append(warnings, w)
			}
		}
		st.Type, st.Status, st.Warnings = model.StatusGood, req.Status, warnings
	case model.StatusBad:
		if req.Warnings != nil {
			return model.Status{}, "bad statuses do not have warnings"
		}
		text := strings.TrimSpace(req.Status)
		if text == "" {
			return model.Status{}, "status must not be empty"
		}
		if utf8.RuneCountInString(text) > maxStatusTextLen {
			return model.Status{}, "status is too long"
		}
		st.Type, st.Status = model.StatusBad, text
	default:
		return model.Status{}, `statusType must be "good" or "bad"`
	}
	return st, ""
}
