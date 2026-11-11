package app

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type sessionContextKey struct{}
type authenticated struct {
	User User
	CSRF string
	Hash []byte
}

func identity(r *http.Request) authenticated {
	value, _ := r.Context().Value(sessionContextKey{}).(authenticated)
	return value
}

func cleanEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || len(value) > 254 {
		return "", invalid("Alamat email tidak valid.")
	}
	return value, nil
}

func (a *App) login(w http.ResponseWriter, r *http.Request) error {
	if !a.allow(r, "login", 5) {
		return &problem{Status: 429, Code: "RATE_LIMITED", Message: "Percobaan masuk terlalu sering. Coba lagi dalam satu menit."}
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	email, _ := cleanEmail(input.Email)
	var user User
	hash := a.dummyHash
	err := a.db.QueryRow(r.Context(), `SELECT u.id::text,u.email,u.name,u.role,COALESCE(u.barber_id::text,''),u.active,u.password_hash
		FROM staff u LEFT JOIN barbers b ON b.id=u.barber_id
		WHERE u.email=$1 AND u.active AND (u.role='admin' OR b.active)`, email).
		Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.BarberID, &user.Active, &hash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	valid := verifyPassword(hash, input.Password)
	if err != nil || !valid {
		return &problem{Status: 401, Code: "INVALID_LOGIN", Message: "Email atau kata sandi tidak sesuai."}
	}
	token, csrf := randomHex(32), randomHex(32)
	expires := a.now().Add(12 * time.Hour)
	_, err = a.db.Exec(r.Context(), "INSERT INTO sessions(token_hash,staff_id,csrf_token,expires_at) VALUES($1,$2,$3,$4)", sessionDigest(token), user.ID, csrf, expires)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: "gilir_session", Value: token, Path: "/", HttpOnly: true, Secure: a.secureCookies,
		SameSite: http.SameSiteLaxMode, MaxAge: 12 * 60 * 60, Expires: expires})
	return respond(w, 200, map[string]any{"user": user, "csrfToken": csrf})
}

func (a *App) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("gilir_session")
		if err != nil || len(cookie.Value) != 64 {
			a.fail(w, &problem{Status: 401, Code: "LOGIN_REQUIRED", Message: "Silakan masuk kembali."})
			return
		}
		var auth authenticated
		auth.Hash = sessionDigest(cookie.Value)
		err = a.db.QueryRow(r.Context(), `SELECT u.id::text,u.email,u.name,u.role,COALESCE(u.barber_id::text,''),u.active,s.csrf_token
			FROM sessions s JOIN staff u ON u.id=s.staff_id LEFT JOIN barbers b ON b.id=u.barber_id
			WHERE s.token_hash=$1 AND s.expires_at>$2 AND u.active AND (u.role='admin' OR b.active)`, auth.Hash, a.now()).
			Scan(&auth.User.ID, &auth.User.Email, &auth.User.Name, &auth.User.Role, &auth.User.BarberID, &auth.User.Active, &auth.CSRF)
		if errors.Is(err, pgx.ErrNoRows) {
			a.fail(w, &problem{Status: 401, Code: "LOGIN_REQUIRED", Message: "Sesi berakhir. Silakan masuk kembali."})
			return
		}
		if err != nil {
			a.fail(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionContextKey{}, auth)))
	})
}

func (a *App) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if identity(r).User.Role != "admin" {
			a.fail(w, &problem{Status: 403, Code: "FORBIDDEN", Message: "Tindakan ini memerlukan akses admin."})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := identity(r).CSRF
		token := r.Header.Get("X-CSRF-Token")
		if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(token)) != 1 {
			a.fail(w, &problem{Status: 403, Code: "CSRF_INVALID", Message: "Sesi formulir perlu dimuat ulang."})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) session(w http.ResponseWriter, r *http.Request) error {
	auth := identity(r)
	return respond(w, 200, map[string]any{"user": auth.User, "csrfToken": auth.CSRF})
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) error {
	if _, err := a.db.Exec(r.Context(), "DELETE FROM sessions WHERE token_hash=$1", identity(r).Hash); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: "gilir_session", Value: "", Path: "/", HttpOnly: true, Secure: a.secureCookies,
		SameSite: http.SameSiteLaxMode, MaxAge: -1})
	return respond(w, 200, map[string]bool{"ok": true})
}

func (a *App) changePassword(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	var previous string
	if err := a.db.QueryRow(r.Context(), "SELECT password_hash FROM staff WHERE id=$1", identity(r).User.ID).Scan(&previous); err != nil {
		return err
	}
	if !verifyPassword(previous, input.CurrentPassword) {
		return invalid("Kata sandi saat ini tidak sesuai.")
	}
	hash, err := PasswordHash(input.NewPassword)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), "UPDATE staff SET password_hash=$1 WHERE id=$2 AND password_hash=$3", hash, identity(r).User.ID, previous)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return conflict("PASSWORD_CHANGED", "Kata sandi sudah berubah. Silakan masuk kembali.")
	}
	if _, err = tx.Exec(r.Context(), "DELETE FROM sessions WHERE staff_id=$1", identity(r).User.ID); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: "gilir_session", Value: "", Path: "/", HttpOnly: true, Secure: a.secureCookies, MaxAge: -1})
	return respond(w, 200, map[string]bool{"ok": true})
}
