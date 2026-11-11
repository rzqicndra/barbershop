package app

import (
	"testing"
	"time"
)

func TestCalendarBoundaries(t *testing.T) {
	catalog := Catalog{Profile: Profile{Open: 600, Close: 1260}, Services: []Service{{ID: "service", Active: true}}, Barbers: []Barber{{ID: "barber", Active: true, ServiceIDs: []string{"service"}, Start: 660, End: 1200, BreakStart: 840, BreakEnd: 885, DaysOff: []int{1}}}}
	cases := []struct {
		name, date      string
		start, duration int
		want            bool
	}{
		{"before barber starts", "2026-11-12", 645, 45, false},
		{"ends exactly at break", "2026-11-12", 795, 45, true},
		{"crosses break", "2026-11-12", 810, 45, false},
		{"starts exactly after break", "2026-11-12", 885, 45, true},
		{"ends exactly at shift end", "2026-11-12", 1155, 45, true},
		{"crosses shift end", "2026-11-12", 1170, 45, false},
		{"weekly day off", "2026-11-16", 660, 45, false},
		{"invalid date", "2026-02-30", 660, 45, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := withinCalendar(catalog, tc.date, tc.start, tc.duration, "service", "barber"); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	catalog.TimeOff = []TimeOff{{BarberID: "barber", Date: "2026-11-12", Start: 900, End: 960}}
	if withinCalendar(catalog, "2026-11-12", 915, 45, "service", "barber") {
		t.Fatal("time off must block booking")
	}
	catalog.Profile.ClosedDates = []string{"2026-11-12"}
	if withinCalendar(catalog, "2026-11-12", 660, 45, "service", "barber") {
		t.Fatal("shop closure must block booking")
	}
}

func TestOnlineWindowAndWalkIn(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, jakarta)
	cases := []struct {
		name, date, mode string
		start            int
		ok               bool
	}{
		{"minimum lead inclusive", "2026-11-12", "online", 630, true},
		{"lead too short", "2026-11-12", "online", 615, false},
		{"off grid", "2026-11-12", "online", 631, false},
		{"last calendar day", "2026-12-12", "online", 600, true},
		{"outside calendar window", "2026-12-13", "online", 600, false},
		{"walk in current minute", "2026-11-12", "walk-in", 600, true},
		{"walk in arbitrary minute", "2026-11-12", "walk-in", 607, true},
		{"walk in tomorrow", "2026-11-13", "walk-in", 600, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateTime(tc.date, tc.start, tc.mode, now)
			if (err == nil) != tc.ok {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
	if _, err := validateTime("2026-11-12", 630, "online", now.Add(time.Second)); err == nil {
		t.Fatal("lead time must include server seconds")
	}
}

func TestPrivateTokenAndPassword(t *testing.T) {
	application := &App{tokenKey: []byte("01234567890123456789012345678901")}
	booking := Booking{ID: newID(), tokenNonce: randomHex(16)}
	token := application.bookingToken(booking)
	id, nonce, ok := application.parseBookingToken(token)
	if !ok || id != booking.ID || nonce != booking.tokenNonce {
		t.Fatal("token must round trip")
	}
	if _, _, ok = application.parseBookingToken("x" + token[1:]); ok {
		t.Fatal("modified token accepted")
	}
	different := &App{tokenKey: []byte("different-key-01234567890123456789")}
	if _, _, ok = different.parseBookingToken(token); ok {
		t.Fatal("token must be bound to signing key")
	}
	hash, err := PasswordHash("Correct-password-2026")
	if err != nil || !verifyPassword(hash, "Correct-password-2026") || verifyPassword(hash, "wrong") {
		t.Fatal("password verification failed")
	}
	if _, err = PasswordHash("short"); err == nil {
		t.Fatal("short password accepted")
	}
}
