package app

import (
	"context"
	"slices"
	"time"
	_ "time/tzdata"
)

var jakarta, _ = time.LoadLocation("Asia/Jakarta")

func dayStart(date string) (time.Time, error) {
	value, err := time.ParseInLocation("2006-01-02", date, jakarta)
	if err != nil || value.Format("2006-01-02") != date {
		return time.Time{}, invalid("Tanggal harus menggunakan format YYYY-MM-DD.")
	}
	return value, nil
}

func withinCalendar(catalog Catalog, date string, start, duration int, serviceID, barberID string) bool {
	day, err := dayStart(date)
	if err != nil || start < 0 || start+duration > 1440 || duration < 1 {
		return false
	}
	serviceActive := false
	for _, service := range catalog.Services {
		if service.ID == serviceID && service.Active {
			serviceActive = true
		}
	}
	if !serviceActive || slices.Contains(catalog.Profile.ClosedDates, date) {
		return false
	}
	for _, barber := range catalog.Barbers {
		if barber.ID != barberID || !barber.Active || !slices.Contains(barber.ServiceIDs, serviceID) {
			continue
		}
		end := start + duration
		if slices.Contains(barber.DaysOff, int(day.Weekday())) ||
			start < max(catalog.Profile.Open, barber.Start) || end > min(catalog.Profile.Close, barber.End) {
			return false
		}
		if barber.BreakStart != barber.BreakEnd && start < barber.BreakEnd && barber.BreakStart < end {
			return false
		}
		for _, block := range catalog.TimeOff {
			if block.BarberID == barberID && block.Date == date && start < block.End && block.Start < end {
				return false
			}
		}
		return true
	}
	return false
}

func validateTime(date string, start int, mode string, now time.Time) (time.Time, error) {
	day, err := dayStart(date)
	if err != nil {
		return time.Time{}, err
	}
	currentDay := now.In(jakarta).Format("2006-01-02")
	lastDate := now.In(jakarta).AddDate(0, 0, 30).Format("2006-01-02")
	if start < 0 || start >= 1440 || date < currentDay || date > lastDate {
		return time.Time{}, invalid("Tanggal atau jam di luar jendela booking.")
	}
	if mode == "walk-in" && date != currentDay {
		return time.Time{}, invalid("Walk-in dicatat pada hari kunjungan.")
	}
	if mode != "walk-in" && start%15 != 0 {
		return time.Time{}, invalid("Jam mulai harus pada interval 15 menit.")
	}
	at := day.Add(time.Duration(start) * time.Minute)
	if mode == "online" && at.Before(now.Add(30*time.Minute)) {
		return time.Time{}, invalid("Booking online paling cepat 30 menit dari waktu server.")
	}
	if mode != "online" && at.Before(now.Truncate(time.Minute)) {
		return time.Time{}, invalid("Jam kunjungan sudah lewat.")
	}
	return at, nil
}

func (a *App) available(ctx context.Context, db queryer, catalog Catalog, date string, start, duration int, serviceID, barberID, mode, ignoreID string) (time.Time, error) {
	at, err := validateTime(date, start, mode, a.now())
	if err != nil {
		return time.Time{}, err
	}
	if !withinCalendar(catalog, date, start, duration, serviceID, barberID) {
		return time.Time{}, invalid("Waktu tidak sesuai jadwal atau layanan barber.")
	}
	var overlap bool
	err = db.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM bookings WHERE barber_id=$1
		AND status IN ('confirmed','serving','completed')
		AND starts_at < $2 AND ends_at > $3 AND id::text <> $4)`,
		barberID, at.Add(time.Duration(duration)*time.Minute), at, ignoreID).Scan(&overlap)
	if err != nil {
		return time.Time{}, err
	}
	if overlap {
		return time.Time{}, conflict("SLOT_TAKEN", "Jam ini sudah terisi. Silakan pilih jam lain.")
	}
	return at, nil
}

func (a *App) affected(ctx context.Context, db queryer, catalog Catalog) ([]string, error) {
	rows, err := db.Query(ctx, bookingSelect+" WHERE b.status='serving' OR (b.status='confirmed' AND b.starts_at >= $1)", a.now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	codes := []string{}
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		if !withinCalendar(catalog, booking.Date, booking.Start, booking.Duration, booking.ServiceID, booking.BarberID) {
			codes = append(codes, booking.Code)
		}
	}
	return codes, rows.Err()
}
