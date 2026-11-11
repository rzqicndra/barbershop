package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Optional acceptance test: real HTTPS proxy, secure sessions, image upload,
// verified reviews and a dump/restore into a different disposable database.
func TestDeploymentRoundTrip(t *testing.T) {
	proxyBinary, pgTools := os.Getenv("E2E_PROXY"), os.Getenv("PG_TOOLS_DIR")
	if proxyBinary == "" || pgTools == "" {
		t.Skip("set E2E_PROXY and PG_TOOLS_DIR for local deployment acceptance")
	}
	f := newFixture(t)
	ctx := context.Background()
	root, _ := filepath.Abs("../../..")
	uploadDir := t.TempDir()
	f.app.uploadDirectory = uploadDir
	f.app.secureCookies = true
	f.app.trustedProxies, _ = proxyPrefixes([]string{"127.0.0.1/32"})
	var clock atomic.Int64
	clock.Store(f.now.UnixMilli())
	f.app.now = func() time.Time { return time.UnixMilli(clock.Load()) }
	backend := httptest.NewServer(f.handler)
	defer backend.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	origin := "https://" + address
	f.app.origin = origin
	f.app.origins = []string{origin}
	config, err := os.ReadFile(filepath.Join(root, "deploy/Caddyfile"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(config), "{$ACME_EMAIL}", "operator@example.test")
	text = strings.ReplaceAll(text, "{$DOMAIN} {", address+" {\n\ttls internal")
	text = strings.ReplaceAll(text, "admin off", fmt.Sprintf("admin off\n\tauto_https disable_redirects\n\tskip_install_trust\n\tstorage file_system {\n\t\troot %q\n\t}", t.TempDir()))
	text = strings.ReplaceAll(text, "app:8080", strings.TrimPrefix(backend.URL, "http://"))
	configPath := filepath.Join(t.TempDir(), "Caddyfile")
	os.WriteFile(configPath, []byte(text), 0600)
	logPath := filepath.Join(t.TempDir(), "proxy.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(proxyBinary, "run", "--config", configPath, "--adapter", "caddyfile")
	command.Stdout, command.Stderr = logFile, logFile
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { command.Process.Signal(syscall.SIGTERM); command.Wait(); logFile.Close() }()
	jar, _ := cookiejar.New(nil)
	// Self-signed trust bypass is restricted to this local test fixture.
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, Jar: jar, Timeout: 5 * time.Second}
	ready := false
	for range 60 {
		response, err := client.Get(origin + "/api/v1/health")
		if err == nil {
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if response.StatusCode == 200 {
				ready = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		logs, _ := os.ReadFile(logPath)
		t.Fatalf("HTTPS proxy not ready: %s", logs)
	}
	var csrf string
	request := func(method, path string, body any, token string) ([]byte, *http.Response) {
		var payload []byte
		if body != nil {
			payload, _ = json.Marshal(body)
		}
		r, _ := http.NewRequest(method, origin+path, bytes.NewReader(payload))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		contents, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode >= 400 {
			t.Fatalf("%s %s returned %d: %s", method, path, response.StatusCode, contents)
		}
		return contents, response
	}
	contents, response := request("POST", "/api/v1/auth/login", map[string]string{"email": "owner@gilir.test", "password": "Owner-password-2026"}, "")
	var session struct {
		CSRF string `json:"csrfToken"`
	}
	json.Unmarshal(contents, &session)
	csrf = session.CSRF
	if len(response.Cookies()) != 1 || !response.Cookies()[0].Secure || !response.Cookies()[0].HttpOnly {
		t.Fatal("secure session flags lost through proxy")
	}
	if response.Header.Get("Strict-Transport-Security") == "" {
		t.Fatal("HTTPS protection header missing")
	}
	contents, _ = request("POST", "/api/v1/bookings", bookingBody(testRaka, 600), "")
	var booking Booking
	json.Unmarshal(contents, &booking)
	clock.Store(time.Date(2026, 11, 12, 10, 1, 0, 0, jakarta).UnixMilli())
	for _, status := range []string{"serving", "completed"} {
		contents, _ = request("POST", "/api/v1/staff/bookings/"+booking.ID+"/status", map[string]any{"status": status, "version": booking.Version}, "")
		var changed Booking
		json.Unmarshal(contents, &changed)
		booking.Version = changed.Version
	}
	request("POST", "/api/v1/bookings/current/review", map[string]any{"rating": 5, "displayName": "Pengunjung uji", "comment": "Pengujian HTTPS dan pemulihan."}, booking.Token)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "fixture.png")
	png.Encode(part, image.NewRGBA(image.Rect(0, 0, 16, 20)))
	writer.Close()
	upload, _ := http.NewRequest("POST", origin+"/api/v1/admin/media", &body)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	upload.Header.Set("X-CSRF-Token", csrf)
	upload.Header.Set("Origin", origin)
	uploaded, err := client.Do(upload)
	if err != nil {
		t.Fatal(err)
	}
	defer uploaded.Body.Close()
	if uploaded.StatusCode != 201 {
		b, _ := io.ReadAll(uploaded.Body)
		t.Fatalf("upload failed: %d %s", uploaded.StatusCode, b)
	}
	var media MediaAsset
	if err = json.NewDecoder(uploaded.Body).Decode(&media); err != nil {
		t.Fatal(err)
	}
	request("POST", "/api/v1/admin/gallery", GalleryPhoto{ImageURL: media.URL, Title: "Uji pemulihan", AltText: "Ilustrasi pengujian foto", Active: true, ConsentConfirmed: true}, "")
	imageData, _ := request("GET", media.URL, nil, "")
	originalHash := sha256.Sum256(imageData)
	// Different socket peers remain separate behind the same proxy. User-supplied
	// X-Real-IP values cannot rotate the first peer out of its rate limit.
	for _, peer := range []string{"127.0.0.2", "127.0.0.3"} {
		transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, DialContext: (&net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP(peer)}}).DialContext}
		visitor := &http.Client{Transport: transport, Timeout: 5 * time.Second}
		attempts := 1
		if peer == "127.0.0.2" {
			attempts = 6
		}
		for index := 0; index < attempts; index++ {
			r, _ := http.NewRequest("POST", origin+"/api/v1/auth/login", strings.NewReader(`{"email":"unknown@gilir.test","password":"Unknown-password-2026"}`))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", origin)
			r.Header.Set("X-Real-IP", fmt.Sprintf("203.0.113.%d", index+1))
			res, err := visitor.Do(r)
			if err != nil {
				t.Fatal(err)
			}
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			want := 401
			if index == 5 {
				want = 429
			}
			if res.StatusCode != want {
				t.Fatalf("proxy peer %s attempt %d status %d want %d", peer, index, res.StatusCode, want)
			}
		}
		transport.CloseIdleConnections()
	}
	// Restore the schema into a different *_test database, then serve its image
	// using a copy of the uploaded files. No production connection is changed.
	schema := strings.Split(f.pool.Config().ConnConfig.RuntimeParams["search_path"], ",")[0]
	backup := filepath.Join(t.TempDir(), "database.dump")
	dump := exec.Command(filepath.Join(pgTools, "pg_dump"), "--dbname", os.Getenv("TEST_DATABASE_URL"), "--schema", schema, "--format=custom", "--file", backup)
	if out, err := dump.CombinedOutput(); err != nil {
		t.Fatalf("dump: %v %s", err, out)
	}
	restoredName := "gilir_restore_" + randomHex(6) + "_test"
	if _, err = f.pool.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{restoredName}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer f.pool.Exec(ctx, "DROP DATABASE "+pgx.Identifier{restoredName}.Sanitize())
	configCopy := f.pool.Config()
	configCopy.ConnConfig.Database = restoredName
	restored, err := pgxpool.NewWithConfig(ctx, configCopy)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if _, err = restored.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	restore := exec.Command(filepath.Join(pgTools, "pg_restore"), "--dbname", restoredName, "--host", configCopy.ConnConfig.Host, "--port", fmt.Sprint(configCopy.ConnConfig.Port), "--username", configCopy.ConnConfig.User, "--exit-on-error", "--no-owner", "--no-privileges", backup)
	restore.Env = append(os.Environ(), "PGPASSWORD="+configCopy.ConnConfig.Password)
	if out, err := restore.CombinedOutput(); err != nil {
		t.Fatalf("restore: %v %s", err, out)
	}
	copyDir := t.TempDir()
	filename := strings.TrimPrefix(media.URL, "/media/")
	source, err := os.ReadFile(filepath.Join(uploadDir, filename))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(copyDir, filename), source, 0600); err != nil {
		t.Fatal(err)
	}
	restoredApp, err := New(restored, Options{Origin: "http://restore.test", TokenKey: f.app.tokenKey, UploadDirectory: copyDir, Clock: f.app.now})
	if err != nil {
		t.Fatal(err)
	}
	imageResponse := httptest.NewRecorder()
	restoredApp.Handler().ServeHTTP(imageResponse, httptest.NewRequest("GET", media.URL, nil))
	if imageResponse.Code != 200 || sha256.Sum256(imageResponse.Body.Bytes()) != originalHash {
		t.Fatal("restored image bytes differ")
	}
	var rating int
	var name string
	if err = restored.QueryRow(ctx, "SELECT r.rating,b.barber_name FROM reviews r JOIN bookings b ON b.id=r.booking_id WHERE b.id=$1", booking.ID).Scan(&rating, &name); err != nil || rating != 5 || name != "Raka" {
		t.Fatalf("restored booking/review mismatch: %d %s %v", rating, name, err)
	}
	t.Log("HTTPS proxy, Secure/HttpOnly cookies, separate client limits, upload, completed review, database restore and identical photo bytes passed")
}
