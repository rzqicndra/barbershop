package app

import (
	"net/http"
	"time"
)

func (a *App) publicAvailability(w http.ResponseWriter, r *http.Request) error {
	return a.catalogAvailability(w, r, "online")
}

func (a *App) adminAvailability(w http.ResponseWriter, r *http.Request) error {
	mode := r.URL.Query().Get("mode")
	if mode != "admin" && mode != "walk-in" {
		return invalid("Mode jadwal tidak sesuai.")
	}
	return a.catalogAvailability(w, r, mode)
}

func (a *App) catalogAvailability(w http.ResponseWriter, r *http.Request, mode string) error {
	catalog, err := loadCatalog(r.Context(), a.db, false)
	if err != nil {
		return err
	}
	serviceID, barberID := r.URL.Query().Get("serviceId"), r.URL.Query().Get("barberId")
	if !validID(serviceID) || !validID(barberID) {
		return invalid("Pilih layanan dan barber yang sesuai.")
	}
	for _, service := range catalog.Services {
		if service.ID == serviceID && service.Active {
			return a.writeSlots(w, r, catalog, serviceID, barberID, service.Duration, mode, "")
		}
	}
	return invalid("Layanan belum tersedia.")
}

func (a *App) customerAvailability(w http.ResponseWriter, r *http.Request) error {
	booking, err := a.customerTarget(r.Context(), r, a.db)
	if err != nil {
		return err
	}
	return a.bookingAvailability(w, r, booking, true)
}

func (a *App) adminBookingAvailability(w http.ResponseWriter, r *http.Request) error {
	booking, err := a.staffTarget(r.Context(), r, a.db)
	if err != nil {
		return err
	}
	return a.bookingAvailability(w, r, booking, false)
}

func (a *App) bookingAvailability(w http.ResponseWriter, r *http.Request, booking Booking, customer bool) error {
	barberID, mode := r.URL.Query().Get("barberId"), "admin"
	if barberID == "" {
		barberID = booking.BarberID
	}
	if customer {
		if barberID != booking.BarberID {
			return invalid("Pilih jadwal untuk barber yang sama.")
		}
		mode = "online"
	}
	if !validID(barberID) {
		return invalid("Barber tidak valid.")
	}
	catalog, err := loadCatalog(r.Context(), a.db, false)
	if err != nil {
		return err
	}
	return a.writeSlots(w, r, catalog, booking.ServiceID, barberID, booking.Duration, mode, booking.ID)
}

func (a *App) writeSlots(w http.ResponseWriter, r *http.Request, catalog Catalog, serviceID, barberID string, duration int, mode, ignoreID string) error {
	date := r.URL.Query().Get("date")
	day, err := dayStart(date)
	if err != nil {
		return err
	}
	if _, err := validateTime(date, max(catalog.Profile.Open, 0), mode, a.now()); err != nil {
		// An earlier first slot on the current day is filtered below.
		current := a.now().In(jakarta)
		if date < current.Format("2006-01-02") || date > current.AddDate(0, 0, 30).Format("2006-01-02") || (mode == "walk-in" && date != current.Format("2006-01-02")) {
			return err
		}
	}
	rows, err := a.db.Query(r.Context(), `SELECT starts_at,ends_at FROM bookings
		WHERE barber_id=$1 AND starts_at >= $2 AND starts_at < $3 AND id::text<>$4
		AND status IN ('confirmed','serving','completed')`, barberID, day, day.AddDate(0, 0, 1), ignoreID)
	if err != nil {
		return err
	}
	type reserved struct{ start, end time.Time }
	busy := []reserved{}
	for rows.Next() {
		var period reserved
		if err := rows.Scan(&period.start, &period.end); err != nil {
			rows.Close()
			return err
		}
		busy = append(busy, period)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	candidates := []int{}
	first := (catalog.Profile.Open + 14) / 15 * 15
	if mode == "walk-in" {
		now := a.now().In(jakarta)
		minute := now.Hour()*60 + now.Minute()
		if minute%15 != 0 {
			candidates = append(candidates, minute)
		}
	}
	for start := first; start+duration <= catalog.Profile.Close; start += 15 {
		candidates = append(candidates, start)
	}
	available := []int{}
	for _, start := range candidates {
		at, err := validateTime(date, start, mode, a.now())
		if err != nil || !withinCalendar(catalog, date, start, duration, serviceID, barberID) {
			continue
		}
		end := at.Add(time.Duration(duration) * time.Minute)
		free := true
		for _, period := range busy {
			if at.Before(period.end) && period.start.Before(end) {
				free = false
				break
			}
		}
		if free {
			available = append(available, start)
		}
	}
	return respond(w, 200, map[string]any{"slots": available, "serverNow": a.now().UTC()})
}
