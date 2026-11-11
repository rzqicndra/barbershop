package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// The control route exists only in this _test.go harness. It is excluded from
// production binaries and uses an isolated *_test database fixture.
func TestBrowserAcceptance(t *testing.T) {
	if os.Getenv("E2E_BROWSER") != "1" {
		t.Skip("set E2E_BROWSER=1 after npm ci and npm run build")
	}
	f := newFixture(t)
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "frontend/dist/index.html")); err != nil {
		t.Fatal("build frontend before browser acceptance")
	}
	f.app.webDirectory = filepath.Join(root, "frontend/dist")
	f.app.uploadDirectory = t.TempDir()
	f.app.secureCookies = true
	var clock atomic.Int64
	clock.Store(f.now.UnixMilli())
	f.app.now = func() time.Time { return time.UnixMilli(clock.Load()) }
	control := randomHex(24)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__test/advance" && r.Method == "POST" && r.Header.Get("X-Test-Control") == control {
			booking, err := findBooking(r.Context(), f.pool, r.URL.Query().Get("bookingId"))
			if err != nil {
				http.Error(w, "fixture not found", 404)
				return
			}
			clock.Store(booking.startsAt.Add(time.Minute).UnixMilli())
			w.WriteHeader(http.StatusNoContent)
			return
		}
		f.handler.ServeHTTP(w, r)
	}))
	defer server.Close()
	f.app.origin = server.URL
	f.app.origins = []string{server.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", filepath.Join(root, "scripts/browser-smoke.cjs"))
	command.Dir = root
	command.Env = append(os.Environ(), "E2E_URL="+server.URL, "E2E_CONTROL_TOKEN="+control)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("browser acceptance: %v\n%s", err, output)
	}
	t.Log(string(output))
	var visits, reviews, photos int
	err = f.pool.QueryRow(context.Background(), "SELECT (SELECT count(*) FROM bookings WHERE status='completed'),(SELECT count(*) FROM reviews),(SELECT count(*) FROM gallery_photos WHERE active)").Scan(&visits, &reviews, &photos)
	if err != nil || visits != 1 || reviews != 1 || photos != 3 {
		t.Fatalf("browser writes were not persisted: visits=%d reviews=%d photos=%d error=%v", visits, reviews, photos, err)
	}
}
