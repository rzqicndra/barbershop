package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {
	Origin            string
	ExtraOrigins      []string
	TokenKey          []byte
	SecureCookies     bool
	TrustedProxyCIDRs []string
	UploadDirectory   string
	WebDirectory      string
	Clock             func() time.Time
}

type App struct {
	db              *pgxpool.Pool
	origin          string
	origins         []string
	tokenKey        []byte
	secureCookies   bool
	trustedProxies  []netip.Prefix
	uploadSlots     chan struct{}
	passwordSlots   chan struct{}
	uploadDirectory string
	webDirectory    string
	now             func() time.Time
	dummyHash       string
	limiter         requestLimiter
}

func New(pool *pgxpool.Pool, options Options) (*App, error) {
	if len(options.TokenKey) < 32 || options.Origin == "" {
		return nil, errors.New("PUBLIC_ORIGIN and a TOKEN_KEY of at least 32 bytes are required")
	}
	if options.Clock == nil {
		options.Clock = time.Now
	}
	proxies, err := proxyPrefixes(options.TrustedProxyCIDRs)
	if err != nil {
		return nil, err
	}
	dummy, err := PasswordHash(randomHex(20))
	if err != nil {
		return nil, err
	}
	return &App{
		db: pool, origin: strings.TrimRight(options.Origin, "/"),
		origins:  append([]string{strings.TrimRight(options.Origin, "/")}, options.ExtraOrigins...),
		tokenKey: options.TokenKey, secureCookies: options.SecureCookies,
		trustedProxies: proxies, uploadSlots: make(chan struct{}, 2), passwordSlots: make(chan struct{}, 2),
		uploadDirectory: options.UploadDirectory, webDirectory: options.WebDirectory, now: options.Clock, dummyHash: dummy,
		limiter: requestLimiter{entries: map[string]limitEntry{}},
	}, nil
}

type endpoint func(http.ResponseWriter, *http.Request) error

func (a *App) handle(fn endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			a.fail(w, err)
		}
	}
}

func (a *App) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(a.headers)
	r.Use(a.recover)
	r.Use(a.staffNavigation)
	r.Get("/api/v1/health", a.handle(a.health))
	r.Get("/api/v1/public", a.handle(a.publicCatalog))
	r.Get("/api/v1/reviews", a.handle(a.publicReviews))
	r.Get("/media/{name}", a.handle(a.media))
	r.Get("/api/v1/availability", a.handle(a.publicAvailability))
	r.Post("/api/v1/bookings", a.handle(a.createPublicBooking))
	r.Get("/api/v1/bookings/current", a.handle(a.customerBooking))
	r.Get("/api/v1/bookings/current/availability", a.handle(a.customerAvailability))
	r.Post("/api/v1/bookings/current/reschedule", a.handle(a.customerReschedule))
	r.Post("/api/v1/bookings/current/cancel", a.handle(a.customerCancel))
	r.Post("/api/v1/bookings/current/review", a.handle(a.createReview))
	r.Post("/api/v1/auth/login", a.handle(a.login))
	r.Group(func(staff chi.Router) {
		staff.Use(a.requireSession)
		staff.Get("/api/v1/auth/session", a.handle(a.session))
		staff.Group(func(write chi.Router) {
			write.Use(a.requireCSRF)
			write.Post("/api/v1/auth/logout", a.handle(a.logout))
			write.Post("/api/v1/auth/password", a.handle(a.changePassword))
			write.Post("/api/v1/staff/bookings/{id}/status", a.handle(a.changeStatus))
		})
		staff.Get("/api/v1/staff/bookings", a.handle(a.listBookings))
		staff.Get("/api/v1/staff/bookings/{id}", a.handle(a.staffBooking))
		staff.Group(func(admin chi.Router) {
			admin.Use(a.requireAdmin)
			admin.Get("/api/v1/admin/catalog", a.handle(a.adminCatalog))
			admin.Get("/api/v1/admin/staff", a.handle(a.listStaff))
			admin.Get("/api/v1/admin/availability", a.handle(a.adminAvailability))
			admin.Get("/api/v1/admin/bookings/{id}/availability", a.handle(a.adminBookingAvailability))
			admin.Group(func(write chi.Router) {
				write.Use(a.requireCSRF)
				write.Post("/api/v1/admin/bookings", a.handle(a.createWalkIn))
				write.Post("/api/v1/admin/bookings/{id}/reschedule", a.handle(a.adminReschedule))
				write.Post("/api/v1/admin/bookings/{id}/cancel", a.handle(a.adminCancel))
				write.Put("/api/v1/admin/profile", a.handle(a.updateProfile))
				write.Post("/api/v1/admin/services", a.handle(a.createService))
				write.Put("/api/v1/admin/services/{id}", a.handle(a.updateService))
				write.Post("/api/v1/admin/barbers", a.handle(a.createBarber))
				write.Put("/api/v1/admin/barbers/{id}", a.handle(a.updateBarber))
				write.Post("/api/v1/admin/time-off", a.handle(a.createTimeOff))
				write.Delete("/api/v1/admin/time-off/{id}", a.handle(a.deleteTimeOff))
				write.Post("/api/v1/admin/staff", a.handle(a.createStaff))
				write.Put("/api/v1/admin/staff/{id}", a.handle(a.updateStaff))
				write.Post("/api/v1/admin/staff/{id}/password", a.handle(a.resetStaffPassword))
				write.Delete("/api/v1/admin/staff/{id}", a.handle(a.deleteStaff))
				write.Delete("/api/v1/admin/services/{id}", a.handle(a.deleteService))
				write.Delete("/api/v1/admin/barbers/{id}", a.handle(a.deleteBarber))
				write.Post("/api/v1/admin/media", a.handle(a.uploadMedia))
				write.Post("/api/v1/admin/gallery", a.handle(a.saveGallery))
				write.Put("/api/v1/admin/gallery/{id}", a.handle(a.saveGallery))
				write.Delete("/api/v1/admin/gallery/{id}", a.handle(a.deleteContent))
				write.Post("/api/v1/admin/faqs", a.handle(a.saveFAQ))
				write.Put("/api/v1/admin/faqs/{id}", a.handle(a.saveFAQ))
				write.Delete("/api/v1/admin/faqs/{id}", a.handle(a.deleteContent))
			})
		})
	})
	r.Handle("/api/*", a.handle(func(w http.ResponseWriter, r *http.Request) error {
		return &problem{Status: 404, Code: "NOT_FOUND", Message: "Endpoint tidak ditemukan."}
	}))
	r.NotFound(a.web)
	return r
}

func (a *App) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Server-Time", a.now().UTC().Format(time.RFC3339Nano))
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; frame-src https://www.google.com https://google.com")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/admin") || r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Cache-Control", "no-store")
		}
		if a.secureCookies {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			origin := r.Header.Get("Origin")
			if origin != "" && !contains(a.origins, origin) {
				a.fail(w, &problem{Status: 403, Code: "ORIGIN_DENIED", Message: "Asal permintaan tidak diizinkan."})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("request panic", "path", r.URL.Path)
				a.fail(w, errors.New("internal request failure"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func decode(r *http.Request, dst any) error {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return &problem{Status: 415, Code: "JSON_REQUIRED", Message: "Gunakan Content-Type application/json."}
	}
	r.Body = http.MaxBytesReader(nil, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return invalid("Isi permintaan JSON tidak valid.")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return invalid("Kirim satu objek JSON untuk setiap permintaan.")
	}
	return nil
}

func respond(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(value)
}

func (a *App) fail(w http.ResponseWriter, err error) {
	var p *problem
	if !errors.As(err, &p) {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
			p = &problem{Status: 409, Code: "SLOT_TAKEN", Message: "Jam ini sudah terisi. Pilih jam lain."}
		} else if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			p = &problem{Status: 409, Code: "DUPLICATE", Message: "Data yang sama sudah terdaftar."}
		} else {
			slog.Error("request failed", "error", err.Error())
			p = &problem{Status: 500, Code: "INTERNAL_ERROR", Message: "Permintaan belum berhasil. Silakan coba kembali."}
		}
	}
	if p.Status == http.StatusTooManyRequests || p.Status == http.StatusServiceUnavailable {
		w.Header().Set("Retry-After", "60")
	}
	respond(w, p.Status, map[string]any{"error": p})
}

func (a *App) web(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.NotFound(w, r)
		return
	}
	if a.webDirectory == "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(a.webDirectory, filepath.FromSlash(filepath.Clean("/"+r.URL.Path)))
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	if filepath.Ext(r.URL.Path) != "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(a.webDirectory, "index.html"))
}

func (a *App) health(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.db.Ping(ctx); err != nil {
		return &problem{Status: 503, Code: "NOT_READY", Message: "Database belum siap."}
	}
	return respond(w, 200, map[string]string{"status": "ok"})
}

func (a *App) publicCatalog(w http.ResponseWriter, r *http.Request) error {
	catalog, err := loadCatalog(r.Context(), a.db, true)
	if err != nil {
		return err
	}
	if err := loadContent(r.Context(), a.db, &catalog, true); err != nil {
		return err
	}
	return respond(w, 200, catalog)
}

func (a *App) adminCatalog(w http.ResponseWriter, r *http.Request) error {
	catalog, err := loadCatalog(r.Context(), a.db, false)
	if err != nil {
		return err
	}
	if err := loadContent(r.Context(), a.db, &catalog, false); err != nil {
		return err
	}
	return respond(w, 200, catalog)
}

func contains(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

type limitEntry struct {
	start time.Time
	count int
}
type requestLimiter struct {
	mu      sync.Mutex
	entries map[string]limitEntry
}

func (a *App) allow(r *http.Request, scope string, maximum int) bool {
	key := scope + ":" + a.clientIP(r)
	a.limiter.mu.Lock()
	defer a.limiter.mu.Unlock()
	now := a.now()
	if len(a.limiter.entries) > 10000 {
		for id, value := range a.limiter.entries {
			if now.Sub(value.start) >= time.Minute {
				delete(a.limiter.entries, id)
			}
		}
		if len(a.limiter.entries) > 10000 {
			return false
		}
	}
	entry := a.limiter.entries[key]
	if now.Sub(entry.start) >= time.Minute {
		entry = limitEntry{start: now}
	}
	if entry.count >= maximum {
		return false
	}
	entry.count++
	a.limiter.entries[key] = entry
	return true
}

func sessionDigest(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}
