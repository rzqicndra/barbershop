package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testService = "11111111-1111-4111-8111-111111111111"
const testRaka = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const testDimas = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

type fixture struct {
	app     *App
	pool    *pgxpool.Pool
	handler http.Handler
	now     time.Time
}
type loginSession struct {
	cookie *http.Cookie
	csrf   string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run real PostgreSQL integration tests")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(config.ConnConfig.Database, "_test") {
		t.Fatal("integration database name must end in _test")
	}
	root, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	schema := "gilir_test_" + randomHex(8)
	if _, err = root.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	config.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		root.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		root.Close()
	})
	if err = Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(ctx, pool); err != nil {
		t.Fatalf("migration not repeatable: %v", err)
	}
	if err = Ready(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = Seed(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = Seed(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = CreateOwner(ctx, pool, "owner@gilir.test", "Owner", "Owner-password-2026"); err != nil {
		t.Fatal(err)
	}
	f := &fixture{pool: pool, now: time.Date(2026, 11, 12, 8, 0, 0, 0, jakarta)}
	f.app, err = New(pool, Options{Origin: "http://gilir.test", TokenKey: []byte("test-key-0123456789012345678901234"), Clock: func() time.Time { return f.now }})
	if err != nil {
		t.Fatal(err)
	}
	f.handler = f.app.Handler()
	return f
}
func (f *fixture) request(method, path string, body any, session *loginSession, token string) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://gilir.test")
	if session != nil {
		req.AddCookie(session.cookie)
		req.Header.Set("X-CSRF-Token", session.csrf)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	f.handler.ServeHTTP(res, req)
	return res
}
func expectStatus(t *testing.T, res *httptest.ResponseRecorder, status int) {
	t.Helper()
	if res.Code != status {
		t.Fatalf("status %d, want %d: %s", res.Code, status, res.Body.String())
	}
}
func readBooking(t *testing.T, res *httptest.ResponseRecorder) Booking {
	t.Helper()
	var value Booking
	if err := json.Unmarshal(res.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func (f *fixture) login(t *testing.T, email, password string) loginSession {
	t.Helper()
	res := f.request("POST", "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil, "")
	expectStatus(t, res, 200)
	var result struct {
		CSRF string `json:"csrfToken"`
	}
	json.Unmarshal(res.Body.Bytes(), &result)
	cookies := res.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("session cookie flags incorrect")
	}
	return loginSession{cookie: cookies[0], csrf: result.CSRF}
}
func bookingBody(barber string, start int) createRequest {
	return createRequest{RequestKey: newID(), ServiceID: testService, BarberID: barber, Date: "2026-11-12", Start: start, Customer: "Arga Test", Phone: "081200000000", OfferVersion: 1}
}

func TestIntegrationConcurrentBookingAndIdempotency(t *testing.T) {
	f := newFixture(t)
	requests := []createRequest{bookingBody(testRaka, 600), bookingBody(testRaka, 600)}
	responses := make([]*httptest.ResponseRecorder, 2)
	var group sync.WaitGroup
	for i := range requests {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			responses[index] = f.request("POST", "/api/v1/bookings", requests[index], nil, "")
		}(i)
	}
	group.Wait()
	success := 0
	for i, res := range responses {
		if res.Code == 201 {
			success++
			continue
		}
		expectStatus(t, res, 409)
		if !strings.Contains(res.Body.String(), "SLOT_TAKEN") {
			t.Fatalf("unexpected conflict at %d: %s", i, res.Body.String())
		}
	}
	if success != 1 {
		t.Fatalf("accepted %d overlapping reservations", success)
	}
	winner := 0
	if responses[1].Code == 201 {
		winner = 1
	}
	original := readBooking(t, responses[winner])
	replay := f.request("POST", "/api/v1/bookings", requests[winner], nil, "")
	expectStatus(t, replay, 200)
	repeated := readBooking(t, replay)
	if original.ID != repeated.ID || original.Code != repeated.Code || original.Token != repeated.Token {
		t.Fatal("idempotent retry changed booking")
	}
	altered := requests[winner]
	altered.Customer = "Different customer"
	expectStatus(t, f.request("POST", "/api/v1/bookings", altered, nil, ""), 409)
	// Half-open ranges allow the next service to begin exactly at the previous end.
	expectStatus(t, f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 645), nil, ""), 201)
	// A different barber has an independent calendar.
	expectStatus(t, f.request("POST", "/api/v1/bookings", bookingBody(testDimas, 660), nil, ""), 201)
	var count int
	f.pool.QueryRow(context.Background(), "SELECT count(*) FROM bookings").Scan(&count)
	if count != 3 {
		t.Fatalf("expected 3 persisted reservations, got %d", count)
	}
	// The database also refuses overlap when application-level checks are bypassed.
	_, dbErr := f.pool.Exec(context.Background(), `INSERT INTO bookings
        (id,code,service_id,service_name,duration,price,barber_id,starts_at,ends_at,customer,phone,source,status,token_nonce)
        SELECT $1,$2,service_id,service_name,duration,price,barber_id,starts_at,ends_at,customer,phone,source,status,token_nonce FROM bookings WHERE id=$3`, newID(), bookingCode(), original.ID)
	var constraint *pgconn.PgError
	if !errors.As(dbErr, &constraint) || constraint.Code != "23P01" {
		t.Fatalf("database exclusion missing: %v", dbErr)
	}
	expectStatus(t, f.request("GET", "/api/v1/bookings/current", nil, nil, original.Token), 200)
	expectStatus(t, f.request("GET", "/api/v1/bookings/current", nil, nil, "bad-token"), 404)
	// A new App instance and different client can read the same persisted private link.
	second, err := New(f.pool, Options{Origin: "http://gilir.test", TokenKey: f.app.tokenKey, Clock: func() time.Time { return f.now }})
	if err != nil {
		t.Fatal(err)
	}
	f.handler = second.Handler()
	expectStatus(t, f.request("GET", "/api/v1/bookings/current", nil, nil, original.Token), 200)
}

func TestIntegrationRescheduleSnapshotAndScheduleGuard(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	first := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 660), nil, "")
	expectStatus(t, first, 201)
	booking := readBooking(t, first)
	second := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 720), nil, "")
	expectStatus(t, second, 201)
	move := changeRequest{Date: booking.Date, Start: 720, BarberID: testRaka, Version: booking.Version}
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/reschedule", move, nil, booking.Token), 409)
	current := readBooking(t, f.request("GET", "/api/v1/bookings/current", nil, nil, booking.Token))
	if current.Start != 660 || current.Version != 1 {
		t.Fatal("failed move changed original reservation")
	}
	catalog, err := loadCatalog(context.Background(), f.pool, false)
	if err != nil {
		t.Fatal(err)
	}
	service := catalog.Services[0]
	service.Price = 99000
	service.Duration = 60
	expectStatus(t, f.request("PUT", "/api/v1/admin/services/"+service.ID, service, &admin, ""), 200)
	// A stale displayed price must be reviewed before a new booking is accepted.
	expectStatus(t, f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 900), nil, ""), 409)
	move.Start = 900
	updated := f.request("POST", "/api/v1/bookings/current/reschedule", move, nil, booking.Token)
	expectStatus(t, updated, 200)
	moved := readBooking(t, updated)
	if moved.Price != 65000 || moved.Duration != 45 || moved.Version != 2 {
		t.Fatal("confirmed snapshot changed during reschedule")
	}
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/cancel", changeRequest{Version: 1}, nil, booking.Token), 409)
	block := TimeOff{BarberID: testRaka, Date: booking.Date, Start: 900, End: 960, Reason: "Izin pribadi"}
	res := f.request("POST", "/api/v1/admin/time-off", block, &admin, "")
	expectStatus(t, res, 409)
	if !strings.Contains(res.Body.String(), "BOOKINGS_AFFECTED") {
		t.Fatal(res.Body.String())
	}
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/cancel", changeRequest{Version: 2}, nil, booking.Token), 200)
	expectStatus(t, f.request("POST", "/api/v1/admin/time-off", block, &admin, ""), 200)
	public := f.request("GET", "/api/v1/public", nil, nil, "")
	expectStatus(t, public, 200)
	if strings.Contains(public.Body.String(), "Izin pribadi") || strings.Contains(public.Body.String(), "081200000000") {
		t.Fatal("public catalog leaked private data")
	}
	availability := f.request("GET", fmt.Sprintf("/api/v1/availability?date=%s&serviceId=%s&barberId=%s", booking.Date, testService, testRaka), nil, nil, "")
	expectStatus(t, availability, 200)
	var slots struct {
		Slots []int `json:"slots"`
	}
	json.Unmarshal(availability.Body.Bytes(), &slots)
	for _, minute := range slots.Slots {
		if minute >= 900 && minute < 960 {
			t.Fatal("time off exposed as available")
		}
	}
}

func TestIntegrationAuthorizationAndStatus(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	create := map[string]string{"email": "raka@gilir.test", "name": "Raka Staff", "password": "Barber-password-2026", "barberId": testRaka}
	expectStatus(t, f.request("POST", "/api/v1/admin/staff", create, &admin, ""), 200)
	barber := f.login(t, "raka@gilir.test", "Barber-password-2026")
	expectStatus(t, f.request("GET", "/api/v1/admin/catalog", nil, &barber, ""), 403)
	expectStatus(t, f.request("GET", "/api/v1/staff/bookings?date=2026-11-12", nil, nil, ""), 401)
	ownResponse := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 660), nil, "")
	expectStatus(t, ownResponse, 201)
	own := readBooking(t, ownResponse)
	otherResponse := f.request("POST", "/api/v1/bookings", bookingBody(testDimas, 660), nil, "")
	expectStatus(t, otherResponse, 201)
	other := readBooking(t, otherResponse)
	expectStatus(t, f.request("GET", "/api/v1/staff/bookings/"+other.ID, nil, &barber, ""), 404)
	list := f.request("GET", "/api/v1/staff/bookings?date=2026-11-12", nil, &barber, "")
	expectStatus(t, list, 200)
	if strings.Contains(list.Body.String(), other.ID) || strings.Contains(list.Body.String(), "phone") {
		t.Fatal("barber saw another barber's booking or contact")
	}
	badCSRF := barber
	badCSRF.csrf = ""
	expectStatus(t, f.request("POST", "/api/v1/staff/bookings/"+own.ID+"/status", map[string]any{"status": "serving", "version": 1}, &badCSRF, ""), 403)
	// Status cannot start early. At the scheduled start the same action is valid.
	expectStatus(t, f.request("POST", "/api/v1/staff/bookings/"+own.ID+"/status", map[string]any{"status": "serving", "version": 1}, &barber, ""), 422)
	f.now = time.Date(2026, 11, 12, 11, 0, 0, 0, jakarta)
	expectStatus(t, f.request("POST", "/api/v1/staff/bookings/"+own.ID+"/status", map[string]any{"status": "serving", "version": 1}, &barber, ""), 200)
	expectStatus(t, f.request("POST", "/api/v1/staff/bookings/"+own.ID+"/status", map[string]any{"status": "completed", "version": 1}, &barber, ""), 409)
	expectStatus(t, f.request("POST", "/api/v1/staff/bookings/"+own.ID+"/status", map[string]any{"status": "completed", "version": 2}, &barber, ""), 200)
	var staffID string
	f.pool.QueryRow(context.Background(), "SELECT id::text FROM staff WHERE email='raka@gilir.test'").Scan(&staffID)
	expectStatus(t, f.request("PUT", "/api/v1/admin/staff/"+staffID, map[string]bool{"active": false}, &admin, ""), 200)
	expectStatus(t, f.request("GET", "/api/v1/auth/session", nil, &barber, ""), 401)
	// A private link remains scoped and expires relative to the saved appointment.
	f.now = f.now.AddDate(0, 0, 31)
	expectStatus(t, f.request("GET", "/api/v1/bookings/current", nil, nil, own.Token), 410)
}

func TestIntegrationWalkInCutoffAndHTTPValidation(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	first := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 600), nil, "")
	expectStatus(t, first, 201)
	booking := readBooking(t, first)
	// At exactly two hours cancellation is allowed. One second later it is denied.
	f.now = f.now.Add(time.Second)
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/cancel", changeRequest{Version: 1}, nil, booking.Token), 422)
	f.now = f.now.Add(-time.Second)
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/cancel", changeRequest{Version: 1}, nil, booking.Token), 200)
	replacement := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 600), nil, "")
	expectStatus(t, replacement, 201)
	f.now = time.Date(2026, 11, 12, 10, 47, 39, 0, jakarta)
	walkIn := bookingBody(testRaka, 647)
	walkIn.Phone = ""
	expectStatus(t, f.request("POST", "/api/v1/admin/bookings", walkIn, &admin, ""), 201)
	// This is the same calendar as online reservations, including partial overlaps.
	future := bookingBody(testRaka, 675)
	expectStatus(t, f.request("POST", "/api/v1/bookings", future, nil, ""), 422)
	future.Start = 690
	expectStatus(t, f.request("POST", "/api/v1/bookings", future, nil, ""), 409)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"x","password":"x"}`))
	req.Header.Set("Content-Type", "text/plain")
	res := httptest.NewRecorder()
	f.handler.ServeHTTP(res, req)
	expectStatus(t, res, 415)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"x","password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://unrelated.test")
	res = httptest.NewRecorder()
	f.handler.ServeHTTP(res, req)
	expectStatus(t, res, 403)
	req = httptest.NewRequest("POST", "/api/v1/bookings", strings.NewReader(`{"requestKey":"x","unexpected":true}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	f.handler.ServeHTTP(res, req)
	expectStatus(t, res, 422)
}
