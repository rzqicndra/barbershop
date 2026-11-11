package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func activeBookingGuard(r *http.Request, tx pgx.Tx, column, id string) error {
	rows, err := tx.Query(r.Context(), "SELECT code FROM bookings WHERE "+column+"=$1 AND status IN ('confirmed','serving') ORDER BY starts_at", id)
	if err != nil {
		return err
	}
	codes := []string{}
	for rows.Next() {
		var code string
		if err = rows.Scan(&code); err != nil {
			rows.Close()
			return err
		}
		codes = append(codes, code)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	return checkAffected(codes)
}

func (a *App) deleteService(w http.ResponseWriter, r *http.Request) error {
	return a.deleteCatalogItem(w, r, false)
}
func (a *App) deleteBarber(w http.ResponseWriter, r *http.Request) error {
	return a.deleteCatalogItem(w, r, true)
}

func (a *App) deleteCatalogItem(w http.ResponseWriter, r *http.Request, barber bool) error {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		return invalid("Referensi data tidak valid.")
	}
	var input struct {
		Version int `json:"version"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	table, column := "services", "service_id"
	if barber {
		table, column = "barbers", "barber_id"
	}
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		var version int
		err := tx.QueryRow(r.Context(), "SELECT version FROM "+table+" WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&version)
		if err == pgx.ErrNoRows {
			return &problem{Status: 404, Code: "NOT_FOUND", Message: "Data sudah dihapus atau tidak ditemukan."}
		}
		if err != nil {
			return err
		}
		if version != input.Version {
			return conflict("SETTINGS_CHANGED", "Data sudah berubah. Muat ulang sebelum menghapus.")
		}
		if err := activeBookingGuard(r, tx, column, id); err != nil {
			return err
		}
		if _, err = tx.Exec(r.Context(), "UPDATE "+table+" SET active=false,deleted_at=$1,version=version+1 WHERE id=$2", a.now(), id); err != nil {
			return err
		}
		if barber {
			if _, err = tx.Exec(r.Context(), "DELETE FROM sessions WHERE staff_id IN (SELECT id FROM staff WHERE barber_id=$1)", id); err != nil {
				return err
			}
			if _, err = tx.Exec(r.Context(), "UPDATE staff SET active=false,deleted_at=$1 WHERE barber_id=$2 AND deleted_at IS NULL", a.now(), id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return a.adminCatalog(w, r)
}

func (a *App) deleteStaff(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		return invalid("Referensi akun tidak valid.")
	}
	if id == identity(r).User.ID {
		return invalid("Akun yang sedang dipakai tidak dapat dihapus.")
	}
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		var role string
		var active bool
		err := tx.QueryRow(r.Context(), "SELECT role,active FROM staff WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&role, &active)
		if err == pgx.ErrNoRows {
			return &problem{Status: 404, Code: "NOT_FOUND", Message: "Akun tidak ditemukan."}
		}
		if err != nil {
			return err
		}
		if role == "admin" && active {
			var remaining int
			if err = tx.QueryRow(r.Context(), "SELECT count(*) FROM staff WHERE role='admin' AND active AND deleted_at IS NULL AND id<>$1", id).Scan(&remaining); err != nil {
				return err
			}
			if remaining < 1 {
				return invalid("Minimal satu Admin aktif harus dipertahankan.")
			}
		}
		if _, err = tx.Exec(r.Context(), "UPDATE staff SET active=false,deleted_at=$1 WHERE id=$2", a.now(), id); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), "DELETE FROM sessions WHERE staff_id=$1", id)
		return err
	})
	if err != nil {
		return err
	}
	return a.listStaff(w, r)
}
