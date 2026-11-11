package app

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

const bookingSelect = `SELECT b.id::text,b.code,b.service_id::text,b.service_name,b.duration,b.price,
	b.barber_id::text,b.starts_at,b.ends_at,b.customer,b.phone,b.source,b.status,b.version,b.token_nonce,b.barber_name FROM bookings b`

type rowScanner interface{ Scan(...any) error }

func scanBooking(row rowScanner) (Booking, error) {
	var b Booking
	err := row.Scan(&b.ID, &b.Code, &b.ServiceID, &b.ServiceName, &b.Duration, &b.Price, &b.BarberID,
		&b.startsAt, &b.endsAt, &b.Customer, &b.Phone, &b.Source, &b.Status, &b.Version, &b.tokenNonce, &b.BarberName)
	at := b.startsAt.In(jakarta)
	b.Date = at.Format("2006-01-02")
	b.Start = at.Hour()*60 + at.Minute()
	return b, err
}

func findBooking(ctx context.Context, db queryer, id string) (Booking, error) {
	if !validID(id) {
		return Booking{}, &problem{Status: 404, Code: "NOT_FOUND", Message: "Booking tidak ditemukan."}
	}
	b, err := scanBooking(db.QueryRow(ctx, bookingSelect+" WHERE b.id=$1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, &problem{Status: 404, Code: "NOT_FOUND", Message: "Booking tidak ditemukan."}
	}
	return b, err
}

func (a *App) customerTarget(ctx context.Context, r *http.Request, db queryer) (Booking, error) {
	id, nonce, valid := a.parseBookingToken(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if !valid {
		return Booking{}, &problem{Status: 404, Code: "BOOKING_LINK_INVALID", Message: "Tautan booking tidak tersedia."}
	}
	b, err := findBooking(ctx, db, id)
	if err != nil {
		return Booking{}, err
	}
	if !hmac.Equal([]byte(b.tokenNonce), []byte(nonce)) {
		return Booking{}, &problem{Status: 404, Code: "BOOKING_LINK_INVALID", Message: "Tautan booking tidak tersedia."}
	}
	if a.now().After(b.endsAt.AddDate(0, 0, 30)) {
		return Booking{}, &problem{Status: 410, Code: "BOOKING_LINK_EXPIRED", Message: "Tautan sudah berakhir. Hubungi barbershop."}
	}
	return b, nil
}

func (a *App) staffTarget(ctx context.Context, r *http.Request, db queryer) (Booking, error) {
	b, err := findBooking(ctx, db, chi.URLParam(r, "id"))
	if err != nil {
		return Booking{}, err
	}
	user := identity(r).User
	if user.Role != "admin" && user.BarberID != b.BarberID {
		return Booking{}, &problem{Status: 404, Code: "NOT_FOUND", Message: "Booking tidak ditemukan."}
	}
	return b, nil
}

func history(ctx context.Context, db queryer, b *Booking) error {
	rows, err := db.Query(ctx, "SELECT action,actor_name,created_at,detail FROM booking_history WHERE booking_id=$1 ORDER BY id", b.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	b.History = []History{}
	for rows.Next() {
		var h History
		var at time.Time
		if err := rows.Scan(&h.Action, &h.By, &at, &h.Detail); err != nil {
			return err
		}
		h.At = at.UnixMilli()
		b.History = append(b.History, h)
	}
	return rows.Err()
}

func (a *App) record(ctx context.Context, db queryer, bookingID, action, detail string, user User) error {
	var actorID any
	actorName := "Pelanggan"
	if user.ID != "" {
		actorID, actorName = user.ID, user.Name
	}
	_, err := db.Exec(ctx, "INSERT INTO booking_history(booking_id,action,actor_id,actor_name,detail,created_at) VALUES($1,$2,$3,$4,$5,$6)",
		bookingID, action, actorID, actorName, detail, a.now())
	return err
}

func (a *App) customerBooking(w http.ResponseWriter, r *http.Request) error {
	b, err := a.customerTarget(r.Context(), r, a.db)
	if err != nil {
		return err
	}
	if err := loadBookingReview(r.Context(), a.db, &b); err != nil {
		return err
	}
	return respond(w, 200, b)
}

func (a *App) staffBooking(w http.ResponseWriter, r *http.Request) error {
	b, err := a.staffTarget(r.Context(), r, a.db)
	if err != nil {
		return err
	}
	if identity(r).User.Role == "admin" {
		b.Token = a.bookingToken(b)
		if err := history(r.Context(), a.db, &b); err != nil {
			return err
		}
	} else {
		b.Phone = ""
	}
	return respond(w, 200, b)
}

func (a *App) listBookings(w http.ResponseWriter, r *http.Request) error {
	day, err := dayStart(r.URL.Query().Get("date"))
	if err != nil {
		return err
	}
	user := identity(r).User
	rows, err := a.db.Query(r.Context(), bookingSelect+` WHERE b.starts_at >= $1 AND b.starts_at < $2
		AND ($3='admin' OR b.barber_id::text=$4) ORDER BY b.starts_at,b.id`, day, day.AddDate(0, 0, 1), user.Role, user.BarberID)
	if err != nil {
		return err
	}
	defer rows.Close()
	bookings := []Booking{}
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return err
		}
		if user.Role != "admin" {
			b.Phone = ""
		}
		bookings = append(bookings, b)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return respond(w, 200, bookings)
}

type createRequest struct {
	RequestKey   string `json:"requestKey"`
	ServiceID    string `json:"serviceId"`
	BarberID     string `json:"barberId"`
	Date         string `json:"date"`
	Start        int    `json:"start"`
	Customer     string `json:"customer"`
	Phone        string `json:"phone"`
	OfferVersion int    `json:"offerVersion"`
}

var phonePattern = regexp.MustCompile(`^(?:\+62|62|0)8[0-9]{8,11}$`)

func (input *createRequest) validate(source string) error {
	input.Customer = strings.TrimSpace(input.Customer)
	input.Phone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(input.Phone))
	if !validID(input.RequestKey) || !validID(input.ServiceID) || !validID(input.BarberID) {
		return invalid("Referensi layanan, barber, atau permintaan tidak valid.")
	}
	if utf8.RuneCountInString(input.Customer) < 2 || utf8.RuneCountInString(input.Customer) > 80 {
		return invalid("Nama pelanggan harus berisi 2 sampai 80 karakter.")
	}
	if (source == "online" || input.Phone != "") && !phonePattern.MatchString(input.Phone) {
		return invalid("Nomor WhatsApp Indonesia tidak valid.")
	}
	return nil
}

func (a *App) createPublicBooking(w http.ResponseWriter, r *http.Request) error {
	return a.create(w, r, "online")
}

func (a *App) createWalkIn(w http.ResponseWriter, r *http.Request) error {
	return a.create(w, r, "walk-in")
}

func (a *App) create(w http.ResponseWriter, r *http.Request, source string) error {
	if !a.allow(r, "create-booking", 15) {
		return &problem{Status: 429, Code: "RATE_LIMITED", Message: "Pemesanan terlalu sering. Coba kembali dalam satu menit."}
	}
	var input createRequest
	if err := decode(r, &input); err != nil {
		return err
	}
	if err := input.validate(source); err != nil {
		return err
	}
	user := identity(r).User
	scope := "online"
	if source == "walk-in" {
		scope = user.ID
	}
	bodyHash := requestHash(input)
	var result Booking
	replayed := false
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		var storedHash, encoded []byte
		var bookingID string
		err := tx.QueryRow(r.Context(), "SELECT payload_hash,booking_id::text,response FROM booking_requests WHERE key=$1 AND actor_scope=$2 AND expires_at>$3",
			input.RequestKey, scope, a.now()).Scan(&storedHash, &bookingID, &encoded)
		if err == nil {
			if !hmac.Equal(storedHash, bodyHash) {
				return conflict("REQUEST_KEY_REUSED", "Kunci permintaan sudah digunakan untuk isian yang berbeda.")
			}
			if err := json.Unmarshal(encoded, &result); err != nil {
				return err
			}
			current, err := findBooking(r.Context(), tx, bookingID)
			if err != nil {
				return err
			}
			if source == "online" || user.Role == "admin" {
				result.Token = a.bookingToken(current)
			}
			replayed = true
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		catalog, err := loadCatalog(r.Context(), tx, false)
		if err != nil {
			return err
		}
		var service Service
		for _, candidate := range catalog.Services {
			if candidate.ID == input.ServiceID && candidate.Active {
				service = candidate
			}
		}
		if service.ID == "" || input.OfferVersion != service.Version {
			return conflict("OFFER_CHANGED", "Layanan, harga, atau durasi berubah. Tinjau dan konfirmasi ulang.")
		}
		startsAt, err := a.available(r.Context(), tx, catalog, input.Date, input.Start, service.Duration, service.ID, input.BarberID, source, "")
		if err != nil {
			return err
		}
		result = Booking{
			ID: newID(), Date: input.Date, Start: input.Start, ServiceID: service.ID,
			ServiceName: service.Name, Duration: service.Duration, Price: service.Price,
			BarberID: input.BarberID, Customer: input.Customer, Phone: input.Phone,
			Source: source, Status: "confirmed", Version: 1,
			startsAt: startsAt, endsAt: startsAt.Add(time.Duration(service.Duration) * time.Minute), tokenNonce: randomHex(16),
		}
		for _, barber := range catalog.Barbers {
			if barber.ID == result.BarberID {
				result.BarberName = barber.Name
			}
		}
		for attempt := 0; attempt < 3; attempt++ {
			result.Code = bookingCode()
			tag, err := tx.Exec(r.Context(), `INSERT INTO bookings
				(id,code,service_id,service_name,duration,price,barber_id,starts_at,ends_at,customer,phone,source,status,token_nonce,barber_name)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'confirmed',$13,$14) ON CONFLICT(code) DO NOTHING`,
				result.ID, result.Code, result.ServiceID, result.ServiceName, result.Duration, result.Price, result.BarberID,
				result.startsAt, result.endsAt, result.Customer, result.Phone, result.Source, result.tokenNonce, result.BarberName)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 1 {
				break
			}
			if attempt == 2 {
				return errors.New("unable to allocate booking reference")
			}
		}
		if err := a.record(r.Context(), tx, result.ID, "Booking dibuat", "", user); err != nil {
			return err
		}
		encoded, err = json.Marshal(result)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(r.Context(), "DELETE FROM booking_requests WHERE expires_at <= $1", a.now()); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), "INSERT INTO booking_requests(key,actor_scope,payload_hash,booking_id,response,expires_at) VALUES($1,$2,$3,$4,$5,$6)",
			input.RequestKey, scope, bodyHash, result.ID, encoded, a.now().Add(24*time.Hour))
		if err == nil && (source == "online" || user.Role == "admin") {
			result.Token = a.bookingToken(result)
		}
		return err
	})
	if err != nil {
		return err
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	return respond(w, status, result)
}

type changeRequest struct {
	Date     string `json:"date"`
	Start    int    `json:"start"`
	BarberID string `json:"barberId"`
	Version  int    `json:"version"`
	Reason   string `json:"reason"`
}

func (a *App) customerReschedule(w http.ResponseWriter, r *http.Request) error {
	return a.modify(w, r, true, true)
}
func (a *App) customerCancel(w http.ResponseWriter, r *http.Request) error {
	return a.modify(w, r, true, false)
}
func (a *App) adminReschedule(w http.ResponseWriter, r *http.Request) error {
	return a.modify(w, r, false, true)
}
func (a *App) adminCancel(w http.ResponseWriter, r *http.Request) error {
	return a.modify(w, r, false, false)
}

func (a *App) modify(w http.ResponseWriter, r *http.Request, customer, moving bool) error {
	var input changeRequest
	if err := decode(r, &input); err != nil {
		return err
	}
	if !customer && (strings.TrimSpace(input.Reason) == "" || len(input.Reason) > 500) {
		return invalid("Isi alasan perubahan, maksimal 500 karakter.")
	}
	var result Booking
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		var previous Booking
		var err error
		if customer {
			previous, err = a.customerTarget(r.Context(), r, tx)
		} else {
			previous, err = a.staffTarget(r.Context(), r, tx)
		}
		if err != nil {
			return err
		}
		if previous.Version != input.Version {
			return conflict("BOOKING_CHANGED", "Booking telah berubah. Muat ulang detail sebelum mencoba.")
		}
		remaining := previous.startsAt.Sub(a.now())
		if previous.Status != "confirmed" || remaining <= 0 || (customer && remaining < 2*time.Hour) {
			return invalid("Booking sudah melewati batas perubahan atau statusnya tidak sesuai.")
		}
		result = previous
		action, detail := "Booking dibatalkan", strings.TrimSpace(input.Reason)
		if moving {
			barberID, mode := input.BarberID, "admin"
			if barberID == "" {
				barberID = previous.BarberID
			}
			if customer {
				if barberID != previous.BarberID {
					return invalid("Perubahan pelanggan menggunakan barber yang sama.")
				}
				mode = "online"
			}
			catalog, err := loadCatalog(r.Context(), tx, false)
			if err != nil {
				return err
			}
			startsAt, err := a.available(r.Context(), tx, catalog, input.Date, input.Start, previous.Duration, previous.ServiceID, barberID, mode, previous.ID)
			if err != nil {
				return err
			}
			result.Date, result.Start, result.BarberID = input.Date, input.Start, barberID
			for _, barber := range catalog.Barbers {
				if barber.ID == barberID {
					result.BarberName = barber.Name
				}
			}
			result.startsAt, result.endsAt = startsAt, startsAt.Add(time.Duration(result.Duration)*time.Minute)
			action = "Jadwal diubah"
			detail = fmt.Sprintf("%s %02d.%02d (%s) -> %s %02d.%02d (%s). %s",
				previous.Date, previous.Start/60, previous.Start%60, previous.BarberID,
				result.Date, result.Start/60, result.Start%60, result.BarberID, detail)
		} else {
			result.Status = "cancelled"
		}
		result.Version++
		_, err = tx.Exec(r.Context(), "UPDATE bookings SET barber_id=$1,starts_at=$2,ends_at=$3,status=$4,version=$5,barber_name=$7 WHERE id=$6",
			result.BarberID, result.startsAt, result.endsAt, result.Status, result.Version, result.ID, result.BarberName)
		if err != nil {
			return err
		}
		return a.record(r.Context(), tx, result.ID, action, detail, identity(r).User)
	})
	if err != nil {
		return err
	}
	return respond(w, 200, result)
}

func (a *App) changeStatus(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		Status  string `json:"status"`
		Version int    `json:"version"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	var result Booking
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		previous, err := a.staffTarget(r.Context(), r, tx)
		if err != nil {
			return err
		}
		if previous.Version != input.Version {
			return conflict("BOOKING_CHANGED", "Booking telah berubah. Muat ulang detail.")
		}
		valid := previous.Status == "confirmed" && input.Status == "serving" && !a.now().Before(previous.startsAt)
		valid = valid || previous.Status == "confirmed" && input.Status == "no-show" && !a.now().Before(previous.endsAt)
		valid = valid || previous.Status == "serving" && input.Status == "completed"
		if !valid {
			return invalid("Perubahan status atau waktu pelayanan tidak sesuai.")
		}
		if input.Status == "serving" {
			var busy bool
			if err := tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM bookings WHERE barber_id=$1 AND status='serving' AND id<>$2)", previous.BarberID, previous.ID).Scan(&busy); err != nil {
				return err
			}
			if busy {
				return conflict("BARBER_BUSY", "Barber masih melayani kunjungan lain. Catat selesai pada kunjungan tersebut sebelum memulai layanan berikutnya.")
			}
		}
		result = previous
		result.Status, result.Version = input.Status, previous.Version+1
		if _, err := tx.Exec(r.Context(), "UPDATE bookings SET status=$1,version=$2 WHERE id=$3", result.Status, result.Version, result.ID); err != nil {
			return err
		}
		return a.record(r.Context(), tx, result.ID, input.Status, "", identity(r).User)
	})
	if err != nil {
		return err
	}
	if identity(r).User.Role != "admin" {
		result.Phone = ""
	}
	return respond(w, 200, result)
}
