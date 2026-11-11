package app

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestProxyIdentityAndRateLimit(t *testing.T) {
	prefixes, err := proxyPrefixes([]string{"172.30.50.10/32", "2001:db8::10/128"})
	if err != nil {
		t.Fatal(err)
	}
	a := &App{trustedProxies: prefixes, now: func() time.Time { return time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC) }, limiter: requestLimiter{entries: map[string]limitEntry{}}}
	for _, item := range []struct{ remote, header, expected string }{
		{"198.51.100.2:1000", "203.0.113.4", "198.51.100.2"},
		{"172.30.50.10:1000", "203.0.113.4", "203.0.113.4"},
		{"172.30.50.11:1000", "203.0.113.4", "172.30.50.11"},
		{"172.30.50.10:1000", "203.0.113.4,203.0.113.5", "172.30.50.10"},
		{"172.30.50.10:1000", "", "172.30.50.10"},
		{"[2001:db8::10]:1000", "2001:db8:1::4", "2001:db8:1::4"},
		{"[::ffff:172.30.50.10]:1000", "::ffff:203.0.113.4", "203.0.113.4"},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = item.remote
		r.Header.Set("X-Real-IP", item.header)
		r.Header.Set("X-Forwarded-For", "192.0.2.99")
		if got := a.clientIP(r); got != item.expected {
			t.Errorf("%s/%s: got %s want %s", item.remote, item.header, got, item.expected)
		}
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "172.30.50.10:1000"
	request.Header.Set("X-Real-IP", "203.0.113.4")
	for range 5 {
		if !a.allow(request, "login", 5) {
			t.Fatal("early rate limit")
		}
	}
	if a.allow(request, "login", 5) {
		t.Fatal("rate limit was bypassed")
	}
	request.Header.Set("X-Real-IP", "203.0.113.5")
	if !a.allow(request, "login", 5) {
		t.Fatal("unrelated visitor shares the proxy rate limit")
	}
	for _, invalid := range []string{"0.0.0.0/0", "::/0", "bad", "172.30.50.10"} {
		if _, err := proxyPrefixes([]string{invalid}); err == nil {
			t.Errorf("accepted invalid proxy %s", invalid)
		}
	}
}

func TestIntegrationOneServingPerBarber(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	firstResponse := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 600), nil, "")
	expectStatus(t, firstResponse, 201)
	first := readBooking(t, firstResponse)
	secondResponse := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 645), nil, "")
	expectStatus(t, secondResponse, 201)
	second := readBooking(t, secondResponse)
	f.now = time.Date(2026, 11, 12, 10, 45, 0, 0, jakarta)
	bookings := []Booking{first, second}
	responses := make([]*httptest.ResponseRecorder, 2)
	var wg sync.WaitGroup
	for index := range bookings {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			responses[i] = f.request("POST", "/api/v1/staff/bookings/"+bookings[i].ID+"/status", map[string]any{"status": "serving", "version": bookings[i].Version}, &admin, "")
		}(index)
	}
	wg.Wait()
	winner, loser := -1, -1
	for i, response := range responses {
		if response.Code == 200 {
			winner = i
		} else {
			expectStatus(t, response, 409)
			loser = i
		}
	}
	if winner < 0 || loser < 0 {
		t.Fatal("two visits started simultaneously")
	}
	// The database also rejects a second active service outside the API.
	_, err := f.pool.Exec(context.Background(), "UPDATE bookings SET status='serving' WHERE id=$1", bookings[loser].ID)
	var constraint *pgconn.PgError
	if !errors.As(err, &constraint) || constraint.Code != "23505" {
		t.Fatalf("serving constraint missing: %v", err)
	}
	active := readBooking(t, responses[winner])
	completed := f.request("POST", "/api/v1/staff/bookings/"+active.ID+"/status", map[string]any{"status": "completed", "version": active.Version}, &admin, "")
	expectStatus(t, completed, 200)
	expectStatus(t, f.request("POST", "/api/v1/staff/bookings/"+bookings[loser].ID+"/status", map[string]any{"status": "serving", "version": bookings[loser].Version}, &admin, ""), 200)
}

func TestIntegrationExpensiveRequestLimits(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	for range cap(f.app.passwordSlots) {
		f.app.passwordSlots <- struct{}{}
	}
	response := f.request("POST", "/api/v1/auth/login", map[string]string{"email": "unknown@gilir.test", "password": "Unknown-password-2026"}, nil, "")
	expectStatus(t, response, 503)
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("missing retry guidance")
	}
	for range cap(f.app.passwordSlots) {
		<-f.app.passwordSlots
	}
	f.app.uploadDirectory = t.TempDir()
	for range cap(f.app.uploadSlots) {
		f.app.uploadSlots <- struct{}{}
	}
	expectStatus(t, f.request("POST", "/api/v1/admin/media", nil, &admin, ""), 503)
	for range cap(f.app.uploadSlots) {
		<-f.app.uploadSlots
	}
	expectStatus(t, f.request("GET", "/api/v1/health", nil, nil, ""), 200)
}
