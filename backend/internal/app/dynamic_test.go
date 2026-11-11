package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func readCatalog(t *testing.T, response *httptest.ResponseRecorder) Catalog {
	t.Helper()
	var catalog Catalog
	if err := json.Unmarshal(response.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	return catalog
}
func completeTestBooking(t *testing.T, f *fixture, admin loginSession, booking Booking) Booking {
	t.Helper()
	f.now = time.Date(2026, 11, 12, booking.Start/60, booking.Start%60, 0, 0, jakarta)
	response := f.request("POST", "/api/v1/staff/bookings/"+booking.ID+"/status", map[string]any{"status": "serving", "version": booking.Version}, &admin, "")
	expectStatus(t, response, 200)
	booking = readBooking(t, response)
	response = f.request("POST", "/api/v1/staff/bookings/"+booking.ID+"/status", map[string]any{"status": "completed", "version": booking.Version}, &admin, "")
	expectStatus(t, response, 200)
	return readBooking(t, response)
}
func addBarberAccount(t *testing.T, f *fixture, admin loginSession) loginSession {
	t.Helper()
	response := f.request("POST", "/api/v1/admin/staff", map[string]string{"email": "raka@gilir.test", "name": "Raka", "password": "Barber-password-2026", "barberId": testRaka}, &admin, "")
	expectStatus(t, response, 200)
	return f.login(t, "raka@gilir.test", "Barber-password-2026")
}

func TestIntegrationDeleteAccountsAndPreserveHistory(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	barber := addBarberAccount(t, f, admin)
	var ownerID, staffID string
	f.pool.QueryRow(context.Background(), "SELECT id::text FROM staff WHERE email='owner@gilir.test'").Scan(&ownerID)
	f.pool.QueryRow(context.Background(), "SELECT id::text FROM staff WHERE email='raka@gilir.test'").Scan(&staffID)
	expectStatus(t, f.request("DELETE", "/api/v1/admin/staff/"+ownerID, nil, &admin, ""), 422)
	expectStatus(t, f.request("DELETE", "/api/v1/admin/staff/"+ownerID, nil, &barber, ""), 403)
	created := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 600), nil, "")
	expectStatus(t, created, 201)
	booking := readBooking(t, created)
	completeTestBooking(t, f, barber, booking)
	response := f.request("DELETE", "/api/v1/admin/staff/"+staffID, nil, &admin, "")
	expectStatus(t, response, 200)
	if strings.Contains(response.Body.String(), staffID) {
		t.Fatal("deleted staff still listed")
	}
	expectStatus(t, f.request("GET", "/api/v1/auth/session", nil, &barber, ""), 401)
	expectStatus(t, f.request("POST", "/api/v1/auth/login", map[string]string{"email": "raka@gilir.test", "password": "Barber-password-2026"}, nil, ""), 401)
	detail := f.request("GET", "/api/v1/staff/bookings/"+booking.ID, nil, &admin, "")
	expectStatus(t, detail, 200)
	if !strings.Contains(detail.Body.String(), "Raka") {
		t.Fatal("actor history lost")
	}
	// Email and barber assignments can be reused by a replacement account, without reactivating the old session.
	addBarberAccount(t, f, admin)
	expectStatus(t, f.request("GET", "/api/v1/auth/session", nil, &barber, ""), 401)
	var replacementID string
	f.pool.QueryRow(context.Background(), "SELECT id::text FROM staff WHERE email='raka@gilir.test' AND deleted_at IS NULL").Scan(&replacementID)
	if replacementID == staffID {
		t.Fatal("deleted identity was reused")
	}
}

func TestIntegrationDeleteCatalogAndKeepBookingSnapshots(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	barber := addBarberAccount(t, f, admin)
	response := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 600), nil, "")
	expectStatus(t, response, 201)
	booking := readBooking(t, response)
	for _, path := range []string{"/admin/barbers/" + testRaka, "/admin/services/" + testService} {
		blocked := f.request("DELETE", "/api/v1"+path, map[string]int{"version": 1}, &admin, "")
		expectStatus(t, blocked, 409)
		if !strings.Contains(blocked.Body.String(), "BOOKINGS_AFFECTED") {
			t.Fatal(blocked.Body.String())
		}
	}
	completeTestBooking(t, f, admin, booking)
	expectStatus(t, f.request("DELETE", "/api/v1/admin/barbers/"+testRaka, map[string]int{"version": 1}, &barber, ""), 403)
	expectStatus(t, f.request("DELETE", "/api/v1/admin/barbers/"+testRaka, map[string]int{"version": 1}, &admin, ""), 200)
	expectStatus(t, f.request("GET", "/api/v1/auth/session", nil, &barber, ""), 401)
	expectStatus(t, f.request("DELETE", "/api/v1/admin/services/"+testService, map[string]int{"version": 1}, &admin, ""), 200)
	catalog := readCatalog(t, f.request("GET", "/api/v1/admin/catalog", nil, &admin, ""))
	for _, service := range catalog.Services {
		if service.ID == testService {
			t.Fatal("deleted service listed")
		}
	}
	for _, person := range catalog.Barbers {
		if person.ID == testRaka {
			t.Fatal("deleted barber listed")
		}
	}
	current := f.request("GET", "/api/v1/bookings/current", nil, nil, booking.Token)
	expectStatus(t, current, 200)
	saved := readBooking(t, current)
	if saved.ServiceName != "Potong rambut" || saved.Price != 65000 || saved.BarberName != "Raka" {
		t.Fatal("historical booking snapshot was damaged")
	}
	if err := Seed(context.Background(), f.pool); err != nil {
		t.Fatal(err)
	}
	catalog = readCatalog(t, f.request("GET", "/api/v1/public", nil, nil, ""))
	for _, service := range catalog.Services {
		if service.ID == testService {
			t.Fatal("seed restored deleted service")
		}
	}
	// Completed historical visits can still submit their own genuine review.
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/review", map[string]any{"rating": 5, "displayName": "Arga", "comment": "Hasil potong rapi."}, nil, booking.Token), 201)
}

func TestIntegrationCompletedReviewsAndRetry(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	response := f.request("POST", "/api/v1/bookings", bookingBody(testRaka, 600), nil, "")
	expectStatus(t, response, 201)
	booking := readBooking(t, response)
	body := map[string]any{"rating": 5, "displayName": "Arga", "comment": "Pelayanan rapi dan sesuai jadwal."}
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/review", body, nil, booking.Token), 409)
	_, err := f.pool.Exec(context.Background(), "INSERT INTO reviews(id,booking_id,rating,display_name) VALUES($1,$2,5,'Arga')", newID(), booking.ID)
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) || pgError.Code != "23514" {
		t.Fatalf("database accepted an unfinished visit: %v", err)
	}
	completeTestBooking(t, f, admin, booking)
	replies := make([]*httptest.ResponseRecorder, 2)
	var group sync.WaitGroup
	for i := range replies {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			replies[index] = f.request("POST", "/api/v1/bookings/current/review", body, nil, booking.Token)
		}(i)
	}
	group.Wait()
	created, replayed := 0, 0
	for _, reply := range replies {
		if reply.Code == 201 {
			created++
		} else if reply.Code == 200 {
			replayed++
		} else {
			t.Fatal(reply.Body.String())
		}
	}
	if created != 1 || replayed != 1 {
		t.Fatal("review retry not idempotent")
	}
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/review", map[string]any{"rating": 1, "displayName": "Arga", "comment": "Different review"}, nil, booking.Token), 409)
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/review", body, nil, "invalid"), 404)
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/review", map[string]any{"rating": 6, "displayName": "Arga"}, nil, booking.Token), 422)
	response = f.request("GET", "/api/v1/public", nil, nil, "")
	expectStatus(t, response, 200)
	catalog := readCatalog(t, response)
	if catalog.Rating.Count != 1 || catalog.Rating.Average != 5 || len(catalog.Reviews) != 1 {
		t.Fatal("rating aggregate not updated")
	}
	if strings.Contains(response.Body.String(), booking.ID) || strings.Contains(response.Body.String(), booking.Phone) || strings.Contains(response.Body.String(), booking.Token) {
		t.Fatal("review leaked booking identity or contact")
	}
	current := readBooking(t, f.request("GET", "/api/v1/bookings/current", nil, nil, booking.Token))
	if current.Review == nil || current.Review.Rating != 5 {
		t.Fatal("saved review not attached to private booking")
	}
	f.now = f.now.AddDate(0, 0, 31)
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/review", body, nil, booking.Token), 410)
}

func TestIntegrationWalkInCustomerReviewLink(t *testing.T) {
	f := newFixture(t)
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	f.now = time.Date(2026, 11, 12, 10, 7, 0, 0, jakarta)
	request := bookingBody(testRaka, 607)
	request.Phone = ""
	response := f.request("POST", "/api/v1/admin/bookings", request, &admin, "")
	expectStatus(t, response, 201)
	booking := readBooking(t, response)
	if booking.Token == "" {
		t.Fatal("walk-in customer link missing")
	}
	expectStatus(t, f.request("GET", "/api/v1/bookings/current", nil, nil, booking.Token), 200)
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/review", map[string]any{"rating": 4, "displayName": "Tamu"}, nil, booking.Token), 409)
	completeTestBooking(t, f, admin, booking)
	expectStatus(t, f.request("POST", "/api/v1/bookings/current/review", map[string]any{"rating": 4, "displayName": "Tamu"}, nil, booking.Token), 201)
}

func uploadPhoto(t *testing.T, f *fixture, session *loginSession, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "customer.jpg")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(data)
	writer.Close()
	req := httptest.NewRequest("POST", "/api/v1/admin/media", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://gilir.test")
	if session != nil {
		req.AddCookie(session.cookie)
		req.Header.Set("X-CSRF-Token", session.csrf)
	}
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, req)
	return response
}
func TestIntegrationDynamicContentAndGalleryUploads(t *testing.T) {
	f := newFixture(t)
	f.app.uploadDirectory = t.TempDir()
	admin := f.login(t, "owner@gilir.test", "Owner-password-2026")
	barber := addBarberAccount(t, f, admin)
	catalog := readCatalog(t, f.request("GET", "/api/v1/admin/catalog", nil, &admin, ""))
	profile := catalog.Profile
	profile.Name = "Studio Rambut"
	profile.Wordmark = "studio"
	profile.HeroTitle = "Rambut baru.\nPilihanmu."
	profile.InstagramURL = "@studio.rambut"
	profile.WhatsApp = "081200000000"
	profile.MapsURL = "https://maps.app.goo.gl/example"
	profile.MapsEmbedURL = `<iframe src="https://www.google.com/maps/embed?pb=example" width="600" allowfullscreen></iframe>`
	response := f.request("PUT", "/api/v1/admin/profile", profile, &admin, "")
	expectStatus(t, response, 200)
	catalog = readCatalog(t, response)
	if catalog.Profile.WhatsApp != "6281200000000" || catalog.Profile.InstagramURL != "https://www.instagram.com/studio.rambut/" || strings.Contains(catalog.Profile.MapsEmbedURL, "<iframe") {
		t.Fatal("contact normalization failed")
	}
	profile = catalog.Profile
	profile.MapsEmbedURL = "https://evil.test/maps/embed"
	expectStatus(t, f.request("PUT", "/api/v1/admin/profile", profile, &admin, ""), 422)
	picture := image.NewRGBA(image.Rect(0, 0, 12, 16))
	picture.Set(2, 2, color.RGBA{R: 120, G: 80, B: 40, A: 255})
	var encoded bytes.Buffer
	jpeg.Encode(&encoded, picture, nil)
	expectStatus(t, uploadPhoto(t, f, nil, encoded.Bytes()), 401)
	expectStatus(t, uploadPhoto(t, f, &barber, encoded.Bytes()), 403)
	expectStatus(t, uploadPhoto(t, f, &admin, []byte(`<svg onload="alert(1)"></svg>`)), 422)
	response = uploadPhoto(t, f, &admin, encoded.Bytes())
	expectStatus(t, response, 201)
	var asset MediaAsset
	json.Unmarshal(response.Body.Bytes(), &asset)
	if asset.URL == "" || asset.Width != 12 || asset.Height != 16 {
		t.Fatal("image metadata missing")
	}
	expectStatus(t, f.request("GET", asset.URL, nil, nil, ""), 404)
	expectStatus(t, f.request("GET", asset.URL, nil, &admin, ""), 200)
	photo := GalleryPhoto{ImageURL: asset.URL, Title: "Crop rapi", AltText: "Hasil potong crop dari belakang", Caption: "Hasil karya tim", Active: true, SortOrder: 1}
	expectStatus(t, f.request("POST", "/api/v1/admin/gallery", photo, &admin, ""), 422)
	photo.ConsentConfirmed = true
	response = f.request("POST", "/api/v1/admin/gallery", photo, &admin, "")
	expectStatus(t, response, 200)
	catalog = readCatalog(t, response)
	if len(catalog.Gallery) != 1 {
		t.Fatal("gallery item not stored")
	}
	photo = catalog.Gallery[0]
	response = f.request("GET", asset.URL, nil, nil, "")
	expectStatus(t, response, 200)
	if response.Header().Get("Content-Type") != "image/jpeg" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("photo response headers unsafe")
	}
	public := readCatalog(t, f.request("GET", "/api/v1/public", nil, nil, ""))
	if len(public.Gallery) != 1 {
		t.Fatal("published photo missing")
	}
	photo.Active = false
	photo.ConsentConfirmed = false
	response = f.request("PUT", "/api/v1/admin/gallery/"+photo.ID, photo, &admin, "")
	expectStatus(t, response, 200)
	catalog = readCatalog(t, response)
	expectStatus(t, f.request("GET", asset.URL, nil, nil, ""), 404)
	expectStatus(t, f.request("DELETE", "/api/v1/admin/gallery/"+photo.ID, map[string]int{"version": catalog.Gallery[0].Version}, &admin, ""), 200)
	public = readCatalog(t, f.request("GET", "/api/v1/public", nil, nil, ""))
	if len(public.Gallery) != 0 {
		t.Fatal("deleted gallery photo still public")
	}
	faq := FAQ{Question: "Bagaimana memilih warna?", Answer: "Hubungi tim untuk konsultasi sebelum memilih layanan.", SortOrder: 4, Active: true}
	response = f.request("POST", "/api/v1/admin/faqs", faq, &admin, "")
	expectStatus(t, response, 200)
	catalog = readCatalog(t, response)
	for _, item := range catalog.FAQs {
		if item.Question == faq.Question {
			faq = item
		}
	}
	if faq.ID == "" {
		t.Fatal("dynamic FAQ missing")
	}
	expectStatus(t, f.request("DELETE", "/api/v1/admin/faqs/"+faq.ID, map[string]int{"version": faq.Version}, &barber, ""), 403)
	expectStatus(t, f.request("DELETE", "/api/v1/admin/faqs/"+faq.ID, map[string]int{"version": faq.Version}, &admin, ""), 200)
}
