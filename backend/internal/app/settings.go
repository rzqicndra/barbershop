package app

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func checkAffected(codes []string) error {
	if len(codes) > 0 {
		shown := codes
		if len(shown) > 3 {
			shown = shown[:3]
		}
		return conflict("BOOKINGS_AFFECTED", fmt.Sprintf("Perubahan memengaruhi %d booking (%s). Tangani booking tersebut di agenda sebelum menyimpan.", len(codes), strings.Join(shown, ", ")))
	}
	return nil
}

func (a *App) updateProfile(w http.ResponseWriter, r *http.Request) error {
	var input Profile
	if err := decode(r, &input); err != nil {
		return err
	}
	input.Name, input.Location = strings.TrimSpace(input.Name), strings.TrimSpace(input.Location)
	if len(input.Name) < 2 || len(input.Name) > 80 || len(input.Location) < 2 || len(input.Location) > 200 ||
		input.Open < 0 || input.Close > 1440 || input.Open >= input.Close {
		return invalid("Periksa nama, lokasi, dan jam buka usaha.")
	}
	for _, date := range input.ClosedDates {
		if _, err := dayStart(date); err != nil {
			return err
		}
	}
	if input.ClosedDates == nil {
		input.ClosedDates = []string{}
	}
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		catalog, err := loadCatalog(r.Context(), tx, false)
		if err != nil {
			return err
		}
		if catalog.Profile.Version != input.Version {
			return conflict("SETTINGS_CHANGED", "Pengaturan sudah berubah. Muat ulang halaman.")
		}
		if err := validatePublicProfile(r.Context(), tx, &input); err != nil {
			return err
		}
		catalog.Profile = input
		codes, err := a.affected(r.Context(), tx, catalog)
		if err != nil {
			return err
		}
		if err := checkAffected(codes); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `UPDATE shop SET name=$1,location=$2,opens_at=$3,closes_at=$4,closed_dates=$5,maps_url=$6,maps_embed_url=$7,instagram_url=$8,whatsapp=$9,wordmark=$10,logo_image_url=$11,
 hero_eyebrow=$12,hero_title=$13,hero_description=$14,hero_image_url=$15,footer_tagline=$16,gallery_title=$17,gallery_description=$18,version=version+1 WHERE id=1`,
			input.Name, input.Location, input.Open, input.Close, input.ClosedDates, input.MapsURL, input.MapsEmbedURL, input.InstagramURL, input.WhatsApp, input.Wordmark, input.LogoImageURL,
			input.HeroEyebrow, input.HeroTitle, input.HeroDescription, input.HeroImageURL, input.FooterTagline, input.GalleryTitle, input.GalleryDescription)
		return err
	})
	if err != nil {
		return err
	}
	return a.adminCatalog(w, r)
}

func validateService(input *Service) error {
	input.Name, input.Description = strings.TrimSpace(input.Name), strings.TrimSpace(input.Description)
	input.Category = strings.TrimSpace(input.Category)
	if input.Category == "" {
		input.Category = "Potong & styling"
	}
	if utf8.RuneCountInString(input.Category) > 50 {
		return invalid("Kategori maksimal 50 karakter.")
	}
	if utf8.RuneCountInString(input.Name) < 2 || utf8.RuneCountInString(input.Name) > 80 || len(input.Description) > 500 ||
		input.Price < 0 || input.Price > 10000000 || input.Duration < 15 || input.Duration > 360 {
		return invalid("Periksa nama layanan, harga, dan durasi 15 sampai 360 menit.")
	}
	return nil
}

func (a *App) createService(w http.ResponseWriter, r *http.Request) error {
	var input Service
	if err := decode(r, &input); err != nil {
		return err
	}
	if err := validateService(&input); err != nil {
		return err
	}
	input.ID = newID()
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		_, err := tx.Exec(r.Context(), "INSERT INTO services(id,name,description,price,duration,active,category) VALUES($1,$2,$3,$4,$5,$6,$7)",
			input.ID, input.Name, input.Description, input.Price, input.Duration, input.Active, input.Category)
		return err
	})
	if err != nil {
		return err
	}
	return a.adminCatalog(w, r)
}

func (a *App) updateService(w http.ResponseWriter, r *http.Request) error {
	var input Service
	if err := decode(r, &input); err != nil {
		return err
	}
	if err := validateService(&input); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if !validID(id) {
		return invalid("Referensi layanan tidak valid.")
	}
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		catalog, err := loadCatalog(r.Context(), tx, false)
		if err != nil {
			return err
		}
		found := false
		for index, current := range catalog.Services {
			if current.ID == id {
				if current.Version != input.Version {
					return conflict("SETTINGS_CHANGED", "Layanan sudah berubah. Muat ulang halaman.")
				}
				input.ID = id
				catalog.Services[index] = input
				found = true
			}
		}
		if !found {
			return &problem{Status: 404, Code: "NOT_FOUND", Message: "Layanan tidak ditemukan."}
		}
		codes, err := a.affected(r.Context(), tx, catalog)
		if err != nil {
			return err
		}
		if err := checkAffected(codes); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), "UPDATE services SET name=$1,description=$2,price=$3,duration=$4,active=$5,category=$7,version=version+1 WHERE id=$6",
			input.Name, input.Description, input.Price, input.Duration, input.Active, id, input.Category)
		return err
	})
	if err != nil {
		return err
	}
	return a.adminCatalog(w, r)
}

func validateBarber(input *Barber) error {
	input.Name, input.Specialty = strings.TrimSpace(input.Name), strings.TrimSpace(input.Specialty)
	if len(input.Name) < 2 || len(input.Name) > 80 || len(input.Specialty) > 200 || input.Start < 0 || input.End > 1440 || input.Start >= input.End ||
		input.BreakStart < 0 || input.BreakEnd > 1440 || input.BreakStart > input.BreakEnd ||
		(input.BreakStart != input.BreakEnd && (input.BreakStart < input.Start || input.BreakEnd > input.End)) {
		return invalid("Periksa nama barber, jam kerja, dan rentang istirahat.")
	}
	for _, day := range input.DaysOff {
		if day < 0 || day > 6 {
			return invalid("Hari libur tidak valid.")
		}
	}
	for _, id := range input.ServiceIDs {
		if !validID(id) {
			return invalid("Layanan barber tidak valid.")
		}
	}
	if input.DaysOff == nil {
		input.DaysOff = []int{}
	}
	if input.ServiceIDs == nil {
		input.ServiceIDs = []string{}
	}
	return nil
}

func (a *App) createBarber(w http.ResponseWriter, r *http.Request) error {
	return a.saveBarber(w, r, true)
}
func (a *App) updateBarber(w http.ResponseWriter, r *http.Request) error {
	return a.saveBarber(w, r, false)
}

func (a *App) saveBarber(w http.ResponseWriter, r *http.Request, creating bool) error {
	var input Barber
	if err := decode(r, &input); err != nil {
		return err
	}
	if err := validateBarber(&input); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if creating {
		id = newID()
	} else if !validID(id) {
		return invalid("Referensi barber tidak valid.")
	}
	input.ID = id
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		catalog, err := loadCatalog(r.Context(), tx, false)
		if err != nil {
			return err
		}
		for _, serviceID := range input.ServiceIDs {
			found := false
			for _, service := range catalog.Services {
				if service.ID == serviceID {
					found = true
				}
			}
			if !found {
				return invalid("Ada layanan barber yang tidak ditemukan.")
			}
		}
		if creating {
			_, err = tx.Exec(r.Context(), `INSERT INTO barbers(id,name,specialty,starts_at,ends_at,break_start,break_end,days_off,active)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, input.Name, input.Specialty, input.Start, input.End, input.BreakStart, input.BreakEnd, input.DaysOff, input.Active)
			if err != nil {
				return err
			}
		} else {
			found := false
			for index, current := range catalog.Barbers {
				if current.ID == id {
					if current.Version != input.Version {
						return conflict("SETTINGS_CHANGED", "Data barber sudah berubah. Muat ulang halaman.")
					}
					catalog.Barbers[index] = input
					found = true
				}
			}
			if !found {
				return &problem{Status: 404, Code: "NOT_FOUND", Message: "Barber tidak ditemukan."}
			}
			codes, err := a.affected(r.Context(), tx, catalog)
			if err != nil {
				return err
			}
			if err := checkAffected(codes); err != nil {
				return err
			}
			_, err = tx.Exec(r.Context(), `UPDATE barbers SET name=$1,specialty=$2,starts_at=$3,ends_at=$4,
				break_start=$5,break_end=$6,days_off=$7,active=$8,version=version+1 WHERE id=$9`,
				input.Name, input.Specialty, input.Start, input.End, input.BreakStart, input.BreakEnd, input.DaysOff, input.Active, id)
			if err != nil {
				return err
			}
			if !input.Active {
				if _, err = tx.Exec(r.Context(), "DELETE FROM sessions WHERE staff_id IN (SELECT id FROM staff WHERE barber_id=$1)", id); err != nil {
					return err
				}
			}
		}
		if _, err = tx.Exec(r.Context(), "DELETE FROM barber_services WHERE barber_id=$1", id); err != nil {
			return err
		}
		for _, serviceID := range input.ServiceIDs {
			if _, err = tx.Exec(r.Context(), "INSERT INTO barber_services(barber_id,service_id) VALUES($1,$2) ON CONFLICT DO NOTHING", id, serviceID); err != nil {
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

func (a *App) createTimeOff(w http.ResponseWriter, r *http.Request) error {
	var input TimeOff
	if err := decode(r, &input); err != nil {
		return err
	}
	if !validID(input.BarberID) || input.Start < 0 || input.End > 1440 || input.Start >= input.End || len(input.Reason) > 500 {
		return invalid("Periksa barber dan rentang jam izin.")
	}
	if _, err := dayStart(input.Date); err != nil {
		return err
	}
	input.ID = newID()
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		catalog, err := loadCatalog(r.Context(), tx, false)
		if err != nil {
			return err
		}
		found := false
		for _, barber := range catalog.Barbers {
			if barber.ID == input.BarberID {
				found = true
			}
		}
		if !found {
			return invalid("Barber tidak ditemukan.")
		}
		catalog.TimeOff = append(catalog.TimeOff, input)
		codes, err := a.affected(r.Context(), tx, catalog)
		if err != nil {
			return err
		}
		if err := checkAffected(codes); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), "INSERT INTO time_off(id,barber_id,date,starts_at,ends_at,reason) VALUES($1,$2,$3,$4,$5,$6)",
			input.ID, input.BarberID, input.Date, input.Start, input.End, strings.TrimSpace(input.Reason))
		return err
	})
	if err != nil {
		return err
	}
	return a.adminCatalog(w, r)
}

func (a *App) deleteTimeOff(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		return invalid("Referensi izin tidak valid.")
	}
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(), "DELETE FROM time_off WHERE id=$1", id)
		if err == nil && tag.RowsAffected() == 0 {
			return &problem{Status: 404, Code: "NOT_FOUND", Message: "Izin tidak ditemukan."}
		}
		return err
	})
	if err != nil {
		return err
	}
	return a.adminCatalog(w, r)
}

func (a *App) listStaff(w http.ResponseWriter, r *http.Request) error {
	rows, err := a.db.Query(r.Context(), "SELECT id::text,email,name,role,COALESCE(barber_id::text,''),active FROM staff WHERE deleted_at IS NULL ORDER BY role,email")
	if err != nil {
		return err
	}
	defer rows.Close()
	staff := []User{}
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.BarberID, &user.Active); err != nil {
			return err
		}
		staff = append(staff, user)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return respond(w, 200, staff)
}

func (a *App) createStaff(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
		BarberID string `json:"barberId"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	email, err := cleanEmail(input.Email)
	if err != nil {
		return err
	}
	if !validID(input.BarberID) || len(strings.TrimSpace(input.Name)) < 2 || len(input.Name) > 80 {
		return invalid("Isi nama dan pilih profil barber.")
	}
	hash, err := PasswordHash(input.Password)
	if err != nil {
		return err
	}
	err = locked(r.Context(), a, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM barbers WHERE id=$1 AND deleted_at IS NULL)", input.BarberID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return invalid("Profil barber tidak ditemukan.")
		}
		_, err := tx.Exec(r.Context(), "INSERT INTO staff(id,email,name,password_hash,role,barber_id) VALUES($1,$2,$3,$4,'barber',$5)",
			newID(), email, strings.TrimSpace(input.Name), hash, input.BarberID)
		return err
	})
	if err != nil {
		return err
	}
	return a.listStaff(w, r)
}

func (a *App) updateStaff(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		Active bool `json:"active"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if !validID(id) {
		return invalid("Akun tidak valid.")
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), "UPDATE staff SET active=$1 WHERE id=$2 AND role='barber' AND deleted_at IS NULL", input.Active, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return invalid("Pengaturan ini digunakan untuk akun barber.")
	}
	if !input.Active {
		if _, err = tx.Exec(r.Context(), "DELETE FROM sessions WHERE staff_id=$1", id); err != nil {
			return err
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return a.listStaff(w, r)
}

func (a *App) resetStaffPassword(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		Password string `json:"password"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	id := chi.URLParam(r, "id")
	if !validID(id) {
		return invalid("Akun tidak valid.")
	}
	hash, err := PasswordHash(input.Password)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), "UPDATE staff SET password_hash=$1 WHERE id=$2 AND role='barber' AND deleted_at IS NULL", hash, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return invalid("Akun barber tidak ditemukan.")
	}
	if _, err = tx.Exec(r.Context(), "DELETE FROM sessions WHERE staff_id=$1", id); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]bool{"ok": true})
}
