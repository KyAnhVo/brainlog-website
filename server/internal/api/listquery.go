package api

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/apitime"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
)

var (
	sortFields = []store.SortField{
		store.SortTrackingNumber, store.SortPOL, store.SortPOD,
		store.SortLastStatusOccurringTime, store.SortCreationTime,
	}
	pageSizes  = []int{10, 20, 50, 100}
	integerRe  = regexp.MustCompile(`^-?\d+$`)
	epoch      = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	listParams = []string{
		"trackingNumber", "pol", "pod",
		"creationTimeMin", "creationTimeMax", "lastStatusOccurringTimeMin", "lastStatusOccurringTimeMax",
		"sortBy", "sortDirection", "page", "pageSize",
	}
)

// listQuery is a validated GET /api/admin/tracking query.
type listQuery struct {
	filter   store.TrackingFilter
	page     int
	pageSize int
}

func defaultDesc(f store.SortField) bool {
	switch f {
	case store.SortTrackingNumber, store.SortPOL, store.SortPOD:
		return false
	default:
		return true
	}
}

// parseListQuery applies defaults and every 400 rule from api-shape.md that can be
// checked without the result set. The pageCount < page rule is checked by the caller.
// Unknown parameters are ignored. A known parameter given more than once is a 400.
func parseListQuery(v url.Values, now time.Time) (listQuery, error) {
	for _, name := range listParams {
		if len(v[name]) > 1 {
			return listQuery{}, fmt.Errorf("%s is given more than once", name)
		}
	}

	timeParam := func(name string, fallback time.Time) (time.Time, error) {
		if !v.Has(name) {
			return fallback, nil
		}
		t, err := apitime.Parse(v.Get(name))
		if err != nil {
			return time.Time{}, fmt.Errorf("%s is not a valid time (yyyy-mm-ddThh:mm:ss.xxxZ)", name)
		}
		if t.After(now.Add(AcceptedBuffer)) {
			return time.Time{}, fmt.Errorf("%s is in the future", name)
		}
		return t, nil
	}
	intParam := func(name string, fallback int) (int, error) {
		if !v.Has(name) {
			return fallback, nil
		}
		s := v.Get(name)
		if !integerRe.MatchString(s) {
			return 0, fmt.Errorf("%s must be an integer", name)
		}
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%s is out of range", name)
		}
		return int(n), nil
	}

	var q listQuery
	f := &q.filter
	f.TrackingNumber = v.Get("trackingNumber")
	f.POL = v.Get("pol")
	f.POD = v.Get("pod")

	var err error
	if f.CreationTimeMin, err = timeParam("creationTimeMin", epoch); err != nil {
		return q, err
	}
	if f.CreationTimeMax, err = timeParam("creationTimeMax", now); err != nil {
		return q, err
	}
	if f.LastStatusOccurringTimeMin, err = timeParam("lastStatusOccurringTimeMin", epoch); err != nil {
		return q, err
	}
	if f.LastStatusOccurringTimeMax, err = timeParam("lastStatusOccurringTimeMax", now); err != nil {
		return q, err
	}
	if f.CreationTimeMin.After(f.CreationTimeMax) {
		return q, fmt.Errorf("creationTimeMin is after creationTimeMax")
	}
	if f.LastStatusOccurringTimeMin.After(f.LastStatusOccurringTimeMax) {
		return q, fmt.Errorf("lastStatusOccurringTimeMin is after lastStatusOccurringTimeMax")
	}

	f.SortBy = store.SortLastStatusOccurringTime
	if v.Has("sortBy") {
		sb := store.SortField(v.Get("sortBy"))
		if !slices.Contains(sortFields, sb) {
			return q, fmt.Errorf("sortBy %q is not supported", sb)
		}
		f.SortBy = sb
	}

	f.Desc = defaultDesc(f.SortBy)
	if v.Has("sortDirection") {
		switch v.Get("sortDirection") {
		case "asc":
			f.Desc = false
		case "desc":
			f.Desc = true
		default:
			return q, fmt.Errorf(`sortDirection must be "asc" or "desc"`)
		}
	}

	if q.page, err = intParam("page", 1); err != nil {
		return q, err
	}
	if q.page < 1 {
		return q, fmt.Errorf("page must be at least 1")
	}
	if q.pageSize, err = intParam("pageSize", 20); err != nil {
		return q, err
	}
	if !slices.Contains(pageSizes, q.pageSize) {
		return q, fmt.Errorf("pageSize must be one of 10, 20, 50, 100")
	}

	f.Offset = (q.page - 1) * q.pageSize
	f.Limit = q.pageSize
	return q, nil
}
