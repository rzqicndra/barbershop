package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gilir.local/backend/internal/app"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		slog.Error("gilir berhenti", "reason", err.Error())
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "healthcheck" {
		client := &http.Client{Timeout: 3 * time.Second}
		response, err := client.Get(env("HEALTHCHECK_URL", "http://127.0.0.1:8080/api/v1/health"))
		if err != nil {
			return errors.New("aplikasi tidak merespons pemeriksaan kesehatan")
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return errors.New("aplikasi atau database belum sehat")
		}
		return nil
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL belum diisi")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return errors.New("DATABASE_URL tidak valid")
	}
	config.MaxConns = 10
	config.MinConns = 1
	config.ConnConfig.ConnectTimeout = 5 * time.Second
	config.ConnConfig.RuntimeParams["statement_timeout"] = "10000"
	config.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "15000"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return errors.New("koneksi database tidak dapat dibuat")
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		return errors.New("database tidak dapat dijangkau, periksa DATABASE_URL dan status PostgreSQL")
	}
	switch command {
	case "migrate":
		if err = app.Migrate(ctx, pool); err != nil {
			return err
		}
		fmt.Println("Migrasi selesai.")
		return nil
	case "seed":
		if err = app.Ready(ctx, pool); err != nil {
			return err
		}
		if err = app.Seed(ctx, pool); err != nil {
			return err
		}
		fmt.Println("Katalog contoh siap. Tidak ada akun atau booking contoh yang dibuat.")
		return nil
	case "admin:create":
		if err = app.Ready(ctx, pool); err != nil {
			return err
		}
		return createOwner(ctx, pool)
	case "serve":
	default:
		return errors.New("perintah tersedia: serve, migrate, seed, admin:create, healthcheck")
	}
	if err = app.Ready(ctx, pool); err != nil {
		return err
	}
	origin := strings.TrimRight(os.Getenv("PUBLIC_ORIGIN"), "/")
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("PUBLIC_ORIGIN harus berupa origin, misalnya http://localhost:8080")
	}
	key, err := hex.DecodeString(os.Getenv("TOKEN_KEY"))
	if err != nil || len(key) < 32 {
		return errors.New("TOKEN_KEY harus berupa minimal 64 digit heksadesimal acak")
	}
	secure, err := strconv.ParseBool(env("COOKIE_SECURE", "false"))
	if err != nil || (parsed.Scheme == "https" && !secure) {
		return errors.New("COOKIE_SECURE harus true untuk PUBLIC_ORIGIN HTTPS")
	}
	if os.Getenv("APP_ENV") == "production" && (parsed.Scheme != "https" || !secure || os.Getenv("EXTRA_ORIGINS") != "") {
		return errors.New("produksi memerlukan PUBLIC_ORIGIN HTTPS, COOKIE_SECURE=true, dan EXTRA_ORIGINS kosong")
	}
	extra := []string{}
	if raw := os.Getenv("EXTRA_ORIGINS"); raw != "" {
		for _, value := range strings.Split(raw, ",") {
			extra = append(extra, strings.TrimSpace(value))
		}
	}
	application, err := app.New(pool, app.Options{Origin: origin, ExtraOrigins: extra, TokenKey: key, SecureCookies: secure, TrustedProxyCIDRs: strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ","), UploadDirectory: env("UPLOAD_DIR", "../data/uploads"), WebDirectory: env("WEB_DIR", "../frontend/dist")})
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: env("ADDR", ":8080"), Handler: application.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 16 * 1024,
	}
	failures := make(chan error, 1)
	go func() { failures <- server.ListenAndServe() }()
	slog.Info("gilir siap", "address", server.Addr, "origin", origin)
	select {
	case err := <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func createOwner(ctx context.Context, pool *pgxpool.Pool) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("jalankan admin:create melalui terminal interaktif")
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Email admin: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Print("Nama admin: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Print("Kata sandi (12–128 karakter, tidak tampil): ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return err
	}
	fmt.Print("Ulangi kata sandi: ")
	confirmation, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return err
	}
	if string(password) != string(confirmation) {
		return errors.New("konfirmasi kata sandi tidak cocok")
	}
	if err = app.CreateOwner(ctx, pool, strings.TrimSpace(email), strings.TrimSpace(name), string(password)); err != nil {
		return err
	}
	fmt.Println("Admin berhasil dibuat. Masuk melalui /admin.")
	return nil
}
