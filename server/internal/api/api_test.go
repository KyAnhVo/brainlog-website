package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/api"
	"github.com/KyAnhVo/brainlog-website/server/internal/seed"
	"github.com/KyAnhVo/brainlog-website/server/internal/store/memstore"
)

// Fixed "now" just after the newest seeded data.
var now0 = time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type env struct {
	t     *testing.T
	srv   *httptest.Server
	clock *clock
}

func newEnv(t *testing.T) *env {
	t.Helper()
	st := memstore.New()
	if _, err := seed.Load(context.Background(), st); err != nil {
		t.Fatal(err)
	}
	c := &clock{t: now0}
	h := api.New(api.Config{
		Store: st, Now: c.now, CookieSecure: false,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, clock: c}
}

// browser is an HTTP client with its own cookie jar.
func (e *env) browser() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

type resp struct {
	status int
	body   map[string]any
	raw    []byte
	header http.Header
	cookie []*http.Cookie
}

func (e *env) do(c *http.Client, method, path string, body any) resp {
	e.t.Helper()
	var rdr io.Reader
	if body != nil {
		if s, ok := body.(string); ok {
			rdr = strings.NewReader(s)
		} else {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		}
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		e.t.Fatalf("%s %s: Content-Type %q, body %s", method, path, ct, raw)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		e.t.Fatalf("%s %s: body is not a JSON object: %s", method, path, raw)
	}
	if res.StatusCode >= 400 {
		if code, _ := m["errorCode"].(float64); int(code) != res.StatusCode || m["errorMsg"] == nil {
			e.t.Fatalf("%s %s: error body is not an ApiError: %s", method, path, raw)
		}
	}
	return resp{res.StatusCode, m, raw, res.Header, res.Cookies()}
}

func (e *env) login(email, pw string) *http.Client {
	e.t.Helper()
	c := e.browser()
	if r := e.do(c, "POST", "/api/admin/login", map[string]string{"email": email, "password": pw}); r.status != 200 {
		e.t.Fatalf("login %s: %d %s", email, r.status, r.raw)
	}
	return c
}

func expect(t *testing.T, r resp, status int) {
	t.Helper()
	if r.status != status {
		t.Fatalf("status = %d, want %d; body %s", r.status, status, r.raw)
	}
}

func trackingNumbers(r resp) []string {
	var out []string
	for _, tr := range r.body["trackings"].([]any) {
		out = append(out, tr.(map[string]any)["trackingNumber"].(string))
	}
	return out
}

func TestCustomerTracking(t *testing.T) {
	e := newEnv(t)
	c := e.browser()

	r := e.do(c, "GET", "/api/tracking/BLJ-good-code-fail-shipment", nil)
	expect(t, r, 200)
	details := r.body["details"].(map[string]any)
	for _, absent := range []string{"cont", "seal", "physicalDetails", "creatorId", "customerId"} {
		if _, ok := details[absent]; ok {
			t.Errorf("details has %q, should be omitted", absent)
		}
	}
	statuses := r.body["statuses"].([]any)
	first, last := statuses[0].(map[string]any), statuses[len(statuses)-1].(map[string]any)
	if w, ok := first["warnings"].([]any); !ok || len(w) != 0 {
		t.Errorf("good status warnings = %v, want present and empty", first["warnings"])
	}
	if _, ok := last["warnings"]; ok || last["statusType"] != "bad" {
		t.Errorf("bad status = %v, want no warnings field", last)
	}
	if _, ok := first["creatorId"]; ok {
		t.Error("customer status leaks creatorId")
	}
	if first["occurringTime"] != "2026-10-15T09:11:00.000Z" {
		t.Errorf("time format: %v", first["occurringTime"])
	}

	// Tie on occurringTime: the later-entered status is last.
	r = e.do(c, "GET", "/api/tracking/BLJ-good-code", nil)
	statuses = r.body["statuses"].([]any)
	if s := statuses[len(statuses)-1].(map[string]any)["status"]; s != "Inland trucking to Place of Delivery" {
		t.Errorf("last status = %v", s)
	}

	expect(t, e.do(c, "GET", "/api/tracking/nope", nil), 404)
	expect(t, e.do(c, "GET", "/api/whatever", nil), 404)
}

func TestAuth(t *testing.T) {
	e := newEnv(t)
	anon := e.browser()

	for _, p := range []string{"/api/admin/me", "/api/admin/tracking", "/api/admin/tracking/BLJ-good-code", "/api/admin/account/x"} {
		expect(t, e.do(anon, "GET", p, nil), 401)
	}
	expect(t, e.do(anon, "POST", "/api/admin/login", map[string]string{"email": "admin@brainlog.test", "password": "nope"}), 401)
	expect(t, e.do(anon, "POST", "/api/admin/login", map[string]string{"email": "ghost@brainlog.test", "password": "x"}), 401)
	expect(t, e.do(anon, "POST", "/api/admin/login", `{"email": 1}`), 400)
	expect(t, e.do(anon, "POST", "/api/admin/login", `{"email":"a","password":"b","extra":1}`), 400)

	c := e.browser()
	r := e.do(c, "POST", "/api/admin/login", map[string]string{"email": "ADMIN@brainlog.test", "password": "Admin123!"})
	expect(t, r, 200)
	if r.body["name"] != "Alice Nguyen" || r.body["role"] != "admin" || r.body["account_created"] != "2026-01-05T08:00:00.000Z" {
		t.Fatalf("login body = %s", r.raw)
	}
	ck := r.cookie[0]
	if ck.Name != api.SessionCookieName || !ck.HttpOnly || ck.SameSite != http.SameSiteLaxMode || ck.MaxAge != 7*24*3600 || len(ck.Value) != 32 {
		t.Fatalf("cookie = %+v", ck)
	}
	expect(t, e.do(c, "GET", "/api/admin/me", nil), 200)

	// Cookie says 7 days but the session row lasts 30; the session itself expires at 30 days.
	e.clock.advance(29 * 24 * time.Hour)
	expect(t, e.do(c, "GET", "/api/admin/me", nil), 200)
	e.clock.advance(2 * 24 * time.Hour)
	expect(t, e.do(c, "GET", "/api/admin/me", nil), 401)
}

func TestLogoutInvalidatesSession(t *testing.T) {
	e := newEnv(t)
	c := e.login("staff@brainlog.test", "Staff123!")
	// Keep a copy of the cookie, as an attacker who stole it would.
	stolen := e.browser()
	u := e.srv.URL + "/"
	req, _ := http.NewRequest("GET", u, nil)
	stolen.Jar.SetCookies(req.URL, c.Jar.Cookies(req.URL))

	expect(t, e.do(c, "POST", "/api/admin/logout", map[string]any{}), 200)
	expect(t, e.do(c, "GET", "/api/admin/me", nil), 401)
	expect(t, e.do(stolen, "GET", "/api/admin/me", nil), 401)
	// Logging out again is harmless.
	expect(t, e.do(c, "POST", "/api/admin/logout", map[string]any{}), 200)
}

func TestListTrackings(t *testing.T) {
	e := newEnv(t)
	c := e.login("staff@brainlog.test", "Staff123!")

	r := e.do(c, "GET", "/api/admin/tracking", nil)
	expect(t, r, 200)
	if r.body["page"] != 1.0 || r.body["pageCount"] != 2.0 || r.body["pageSize"] != 20.0 {
		t.Fatalf("paging = %v %v %v", r.body["page"], r.body["pageCount"], r.body["pageSize"])
	}
	got := trackingNumbers(r)
	if len(got) != 20 || got[0] != "BLJ-2026-0020" || got[1] != "BLJ-2026-0019" {
		t.Fatalf("default order starts %v", got[:3])
	}
	row := r.body["trackings"].([]any)[0].(map[string]any)
	ls := row["lastStatus"].(map[string]any)
	if row["creatorId"] == nil || row["creationTime"] == nil || ls["creatorId"] == nil || ls["creationTime"] == nil {
		t.Fatalf("admin row missing admin fields: %v", row)
	}

	cases := []struct {
		query string
		want  []string
	}{
		{"?pol=da&sortBy=trackingNumber", []string{"BLJ-2026-0003", "BLJ-2026-0014", "BLJ-2026-0020", "BLJ-good-code-fail-shipment"}},
		{"?pol=HO%20CHI&sortBy=trackingNumber", []string{"BLJ-2026-0001", "BLJ-2026-0002", "BLJ-2026-0009", "BLJ-2026-0012", "BLJ-2026-0018"}},
		{"?trackingNumber=FAIL", []string{"BLJ-good-code-fail-shipment"}},
		{"?trackingNumber=zzz", nil},
		// Tie on pod. asc is the exact reverse of desc, so the tie-break flips to lastStatus ASC.
		{"?pod=busan&sortBy=pod", []string{"BLJ-good-code-fail-shipment", "BLJ-2026-0016"}},
		{"?pod=busan&sortBy=pod&sortDirection=desc", []string{"BLJ-2026-0016", "BLJ-good-code-fail-shipment"}},
		{"?creationTimeMin=2026-11-14T00:00:00.000Z", []string{"BLJ-2026-0020", "BLJ-2026-0019", "BLJ-2026-0018"}},
		{"?lastStatusOccurringTimeMax=2026-09-27T13:59:59.999Z", nil},
		{"?lastStatusOccurringTimeMax=2026-09-27T14:00:00.000Z", []string{"BLJ-2026-0006"}}, // inclusive bound
		{"?lastStatusOccurringTimeMax=2026-10-03T00:00:00.000Z", []string{"BLJ-2026-0001", "BLJ-2026-0006"}},
	}
	for _, tc := range cases {
		r := e.do(c, "GET", "/api/admin/tracking"+tc.query, nil)
		expect(t, r, 200)
		if got := trackingNumbers(r); !slices.Equal(got, tc.want) {
			t.Errorf("%s -> %v, want %v", tc.query, got, tc.want)
		}
	}

	// Empty result: pageCount 0, page 1 is fine, page 2 is not.
	r = e.do(c, "GET", "/api/admin/tracking?trackingNumber=zzz", nil)
	if r.body["pageCount"] != 0.0 || len(r.body["trackings"].([]any)) != 0 {
		t.Fatalf("empty = %s", r.raw)
	}

	// asc is the exact reverse of desc, for every sortBy.
	for _, sb := range []string{"trackingNumber", "pol", "pod", "lastStatusOccurringTime", "creationTime"} {
		d := trackingNumbers(e.do(c, "GET", "/api/admin/tracking?pageSize=100&sortDirection=desc&sortBy="+sb, nil))
		a := trackingNumbers(e.do(c, "GET", "/api/admin/tracking?pageSize=100&sortDirection=asc&sortBy="+sb, nil))
		slices.Reverse(a)
		if !slices.Equal(a, d) {
			t.Errorf("sortBy=%s: asc is not the reverse of desc", sb)
		}
	}

	// Pages partition the result.
	var all []string
	for p := 1; p <= 3; p++ {
		all = append(all, trackingNumbers(e.do(c, "GET", "/api/admin/tracking?pageSize=10&page="+string(rune('0'+p)), nil))...)
	}
	full := trackingNumbers(e.do(c, "GET", "/api/admin/tracking?pageSize=100", nil))
	if !slices.Equal(all, full) || len(full) != 24 {
		t.Errorf("pages %v != full %v", all, full)
	}

	bad := []string{
		"?pageSize=15", "?pageSize=abc", "?page=0", "?page=-1", "?page=1.5", "?page=", "?page=99999999999",
		"?page=4&pageSize=10", "?trackingNumber=zzz&page=2",
		"?sortBy=foo", "?sortBy=", "?sortDirection=up",
		"?creationTimeMin=2026-02-30T00:00:00.000Z", "?creationTimeMin=2026-11-01T00:00:00Z",
		"?creationTimeMin=2026-11-10T00:00:00.000Z&creationTimeMax=2026-11-01T00:00:00.000Z",
		"?lastStatusOccurringTimeMin=2026-11-10T00:00:00.000Z&lastStatusOccurringTimeMax=2026-11-01T00:00:00.000Z",
		"?lastStatusOccurringTimeMax=2026-11-17T00:06:00.000Z", // beyond now + 5 min
		"?page=1&page=2",
	}
	for _, q := range bad {
		if r := e.do(c, "GET", "/api/admin/tracking"+q, nil); r.status != 400 {
			t.Errorf("%s -> %d, want 400", q, r.status)
		}
	}
	// Within the 5-minute skew buffer is fine; unknown params are ignored.
	expect(t, e.do(c, "GET", "/api/admin/tracking?creationTimeMax=2026-11-17T00:04:00.000Z&foo=bar", nil), 200)
}

func TestAddStatus(t *testing.T) {
	e := newEnv(t)
	c := e.login("ops@brainlog.test", "Ops12345!")
	path := "/api/admin/tracking/BLJ-good-code/status"
	at := "2026-11-16T12:00:00.000Z"

	bad := []any{
		map[string]any{"statusType": "good", "status": "Nope", "warnings": []string{}, "occurringTime": at, "location": "x"},
		map[string]any{"statusType": "good", "status": "Finished", "warnings": []string{" "}, "occurringTime": at, "location": "x"},
		map[string]any{"statusType": "good", "status": "Finished", "occurringTime": "2026-11-17T00:06:00.000Z", "location": "x"},
		map[string]any{"statusType": "good", "status": "Finished", "occurringTime": "2026-11-16", "location": "x"},
		map[string]any{"statusType": "good", "status": "Finished", "occurringTime": at, "location": "  "},
		map[string]any{"statusType": "bad", "status": "  ", "occurringTime": at, "location": "x"},
		map[string]any{"statusType": "bad", "status": "Lost", "warnings": []string{}, "occurringTime": at, "location": "x"},
		map[string]any{"statusType": "meh", "status": "Lost", "occurringTime": at, "location": "x"},
		map[string]any{"statusType": "bad", "status": "Lost", "occurringTime": at, "location": "x", "id": "forged"},
	}
	for i, b := range bad {
		if r := e.do(c, "POST", path, b); r.status != 400 {
			t.Errorf("bad[%d] -> %d %s", i, r.status, r.raw)
		}
	}

	// Wrong content type.
	req, _ := http.NewRequest("POST", e.srv.URL+path, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "text/plain")
	res, _ := c.Do(req)
	res.Body.Close()
	if res.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("text/plain -> %d", res.StatusCode)
	}

	expect(t, e.do(c, "POST", "/api/admin/tracking/nope/status",
		map[string]any{"statusType": "bad", "status": "x", "occurringTime": at, "location": "x"}), 404)

	r := e.do(c, "POST", path, map[string]any{
		"statusType": "good", "status": "Arrived at Place of Delivery", "warnings": []string{" Late "},
		"occurringTime": at, "location": " Jurong, SG ",
	})
	expect(t, r, 200)
	if r.body["creatorId"] != seed.Chloe || r.body["creationTime"] != "2026-11-17T00:00:00.000Z" || r.body["location"] != "Jurong, SG" {
		t.Fatalf("created = %s", r.raw)
	}
	if w := r.body["warnings"].([]any); len(w) != 1 || w[0] != "Late" {
		t.Fatalf("warnings = %v", w)
	}

	// It is now the last status, so BLJ-good-code leads the default list.
	list := e.do(c, "GET", "/api/admin/tracking", nil)
	if got := trackingNumbers(list)[0]; got != "BLJ-good-code" {
		t.Fatalf("first = %s", got)
	}
	// Backfilled status (older occurringTime) lands in the middle, not at the end.
	expect(t, e.do(c, "POST", path, map[string]any{
		"statusType": "bad", "status": "Paperwork lost", "occurringTime": "2026-10-16T00:00:00.000Z", "location": "x",
	}), 200)
	d := e.do(c, "GET", "/api/admin/tracking/BLJ-good-code", nil)
	st := d.body["statuses"].([]any)
	if st[len(st)-1].(map[string]any)["status"] != "Arrived at Place of Delivery" || st[2].(map[string]any)["status"] != "Paperwork lost" {
		t.Fatalf("order after backfill: %s", d.raw)
	}
}

func TestAccounts(t *testing.T) {
	e := newEnv(t)
	user := e.login("staff@brainlog.test", "Staff123!")
	admin := e.login("admin@brainlog.test", "Admin123!")

	newAcc := map[string]string{"email": "new@brainlog.test", "password": "longenough", "name": " New Person ", "role": "user"}
	expect(t, e.do(user, "POST", "/api/admin/createAccount", newAcc), 401)
	expect(t, e.do(user, "POST", "/api/admin/deleteAccount", map[string]string{"email": "ops@brainlog.test"}), 401)

	for _, b := range []map[string]string{
		{"email": "not-an-email", "password": "longenough", "name": "x", "role": "user"},
		{"email": "Bob <bob@x.test>", "password": "longenough", "name": "x", "role": "user"},
		{"email": "a@x.test", "password": "short", "name": "x", "role": "user"},
		{"email": "a@x.test", "password": "longenough", "name": "  ", "role": "user"},
		{"email": "a@x.test", "password": "longenough", "name": "x", "role": "root"},
	} {
		if r := e.do(admin, "POST", "/api/admin/createAccount", b); r.status != 400 {
			t.Errorf("%v -> %d", b, r.status)
		}
	}
	expect(t, e.do(admin, "POST", "/api/admin/createAccount",
		map[string]string{"email": "OPS@brainlog.test", "password": "longenough", "name": "x", "role": "user"}), 409)

	r := e.do(admin, "POST", "/api/admin/createAccount", newAcc)
	expect(t, r, 200)
	id := r.body["id"].(string)
	if r.body["name"] != "New Person" || len(id) != 36 || id[14] != '7' {
		t.Fatalf("create = %s", r.raw)
	}
	expect(t, e.do(admin, "GET", "/api/admin/account/"+id, nil), 200)
	expect(t, e.do(admin, "GET", "/api/admin/account/"+seed.Deleted, nil), 404)
	newbie := e.login("new@brainlog.test", "longenough")

	expect(t, e.do(admin, "POST", "/api/admin/deleteAccount", map[string]string{}), 400)
	expect(t, e.do(admin, "POST", "/api/admin/deleteAccount", map[string]string{"email": "a", "id": "b"}), 400)
	expect(t, e.do(admin, "POST", "/api/admin/deleteAccount", map[string]string{"email": "ghost@brainlog.test"}), 404)
	expect(t, e.do(admin, "POST", "/api/admin/deleteAccount", map[string]string{"id": id}), 200)
	// Deleting the account killed its sessions.
	expect(t, e.do(newbie, "GET", "/api/admin/me", nil), 401)

	// Deleting yourself ends your own session and clears the cookie.
	r = e.do(admin, "POST", "/api/admin/deleteAccount", map[string]string{"email": "Admin@Brainlog.test"})
	expect(t, r, 200)
	if len(r.cookie) != 1 || r.cookie[0].MaxAge >= 0 {
		t.Errorf("self-delete did not clear cookie: %v", r.cookie)
	}
	expect(t, e.do(admin, "GET", "/api/admin/me", nil), 401)
	expect(t, e.do(e.browser(), "POST", "/api/admin/login", map[string]string{"email": "admin@brainlog.test", "password": "Admin123!"}), 401)
}
