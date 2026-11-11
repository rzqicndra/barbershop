package app

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const reviewSelect = `SELECT r.id::text,r.rating,r.display_name,r.comment,b.service_name,b.barber_name,r.created_at FROM reviews r JOIN bookings b ON b.id=r.booking_id`

func scanReview(row rowScanner) (Review, error) {
	var review Review
	err := row.Scan(&review.ID, &review.Rating, &review.DisplayName, &review.Comment, &review.ServiceName, &review.BarberName, &review.CreatedAt)
	return review, err
}

func readReviews(ctx context.Context, db queryer, offset, limit int) ([]Review, error) {
	rows, err := db.Query(ctx, reviewSelect+" WHERE b.status='completed' ORDER BY r.created_at DESC,r.id DESC OFFSET $1 LIMIT $2", offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reviews := []Review{}
	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func loadBookingReview(ctx context.Context, db queryer, booking *Booking) error {
	review, err := scanReview(db.QueryRow(ctx, reviewSelect+" WHERE r.booking_id=$1", booking.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	booking.Review = &review
	return nil
}

func (a *App) publicReviews(w http.ResponseWriter, r *http.Request) error {
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > 100000 {
			return invalid("Halaman ulasan tidak valid.")
		}
		offset = parsed
	}
	reviews, err := readReviews(r.Context(), a.db, offset, 12)
	if err != nil {
		return err
	}
	return respond(w, 200, reviews)
}

func (a *App) createReview(w http.ResponseWriter, r *http.Request) error {
	if !a.allow(r, "review", 10) {
		return &problem{Status: 429, Code: "RATE_LIMITED", Message: "Pengiriman ulasan terlalu sering. Coba lagi dalam satu menit."}
	}
	var input struct {
		Rating      int    `json:"rating"`
		DisplayName string `json:"displayName"`
		Comment     string `json:"comment"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	input.DisplayName, input.Comment = strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.Comment)
	if input.Rating < 1 || input.Rating > 5 || utf8.RuneCountInString(input.DisplayName) < 2 || utf8.RuneCountInString(input.DisplayName) > 40 || utf8.RuneCountInString(input.Comment) > 1000 {
		return invalid("Pilih 1–5 bintang, nama tampilan 2–40 karakter, dan ulasan maksimal 1.000 karakter.")
	}
	var result Review
	replayed := false
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		booking, err := a.customerTarget(r.Context(), r, tx)
		if err != nil {
			return err
		}
		if booking.Status != "completed" {
			return conflict("REVIEW_NOT_READY", "Ulasan hanya dapat diberikan setelah kunjungan berstatus Selesai.")
		}
		if err = loadBookingReview(r.Context(), tx, &booking); err != nil {
			return err
		}
		if booking.Review != nil {
			previous := booking.Review
			if previous.Rating != input.Rating || previous.DisplayName != input.DisplayName || previous.Comment != input.Comment {
				return conflict("REVIEW_EXISTS", "Kunjungan ini sudah memiliki ulasan.")
			}
			result = *previous
			replayed = true
			return nil
		}
		id := newID()
		_, err = tx.Exec(r.Context(), "INSERT INTO reviews(id,booking_id,rating,display_name,comment,created_at) VALUES($1,$2,$3,$4,$5,$6)", id, booking.ID, input.Rating, input.DisplayName, input.Comment, a.now())
		if err != nil {
			return err
		}
		result, err = scanReview(tx.QueryRow(r.Context(), reviewSelect+" WHERE r.id=$1", id))
		return err
	})
	if err != nil {
		return err
	}
	status := 201
	if replayed {
		status = 200
	}
	return respond(w, status, result)
}
