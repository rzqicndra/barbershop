package app

import (
	"net/http"
	"strings"
)

func staffHome(role string) string {
	if role == "barber" {
		return "/admin/jadwal"
	}
	return "/admin/agenda"
}

func staffPageAllowed(role, path string) bool {
	if path == "/admin/akun" {
		return true
	}
	if role == "admin" {
		return path == "/admin/agenda" || path == "/admin/pengaturan"
	}
	return role == "barber" && path == "/admin/jadwal"
}

// staffNavigation handles HTML entry points only. API endpoints retain JSON
// authorization responses, and static assets never require a session query.
func (a *App) staffNavigation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimRight(r.URL.Path, "/")
		staffPage := path == "/admin" || strings.HasPrefix(path, "/admin/")
		publicEntry := path == "" || path == "/index.html"
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) || (!staffPage && !publicEntry) {
			next.ServeHTTP(w, r)
			return
		}
		auth, valid, err := a.sessionFromRequest(r)
		if err != nil {
			a.fail(w, err)
			return
		}
		destination := ""
		if valid {
			if publicEntry || !staffPageAllowed(auth.User.Role, path) {
				destination = staffHome(auth.User.Role)
			}
		} else if staffPage && path != "/admin" {
			destination = "/admin"
		}
		if destination == "" && staffPage && r.URL.Path != path {
			destination = path
		}
		if destination != "" {
			w.Header().Set("Cache-Control", "no-store")
			http.Redirect(w, r, destination, http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
