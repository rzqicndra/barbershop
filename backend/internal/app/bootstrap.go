package app

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Seed installs the original UI's example catalog without creating customer data.
// Repeating this command never overwrites an owner's edited catalog.
func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT id FROM shop WHERE id=1 FOR UPDATE"); err != nil {
		return err
	}
	services := []Service{
		{ID: "11111111-1111-4111-8111-111111111111", Name: "Potong rambut", Description: "Konsultasi singkat, potong, dan styling ringan.", Price: 65000, Duration: 45},
		{ID: "22222222-2222-4222-8222-222222222222", Name: "Potong + cuci", Description: "Potong rambut dengan cuci dan styling setelahnya.", Price: 85000, Duration: 60},
		{ID: "33333333-3333-4333-8333-333333333333", Name: "Rapikan jenggot", Description: "Rapikan garis pipi, kumis, dan jenggot.", Price: 35000, Duration: 30},
	}
	for _, service := range services {
		if _, err = tx.Exec(ctx, `INSERT INTO services(id,name,description,price,duration) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO NOTHING`,
			service.ID, service.Name, service.Description, service.Price, service.Duration); err != nil {
			return err
		}
	}
	barbers := []Barber{
		{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Name: "Raka", Specialty: "Scissor cut dan klasik", Start: 600, End: 1230, BreakStart: 840, BreakEnd: 885, DaysOff: []int{1}, ServiceIDs: []string{services[0].ID, services[1].ID, services[2].ID}},
		{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Name: "Dimas", Specialty: "Fade dan tekstur pendek", Start: 660, End: 1260, BreakStart: 900, BreakEnd: 945, DaysOff: []int{3}, ServiceIDs: []string{services[0].ID, services[1].ID}},
		{ID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", Name: "Naufal", Specialty: "Potongan pendek dan jenggot", Start: 600, End: 1140, BreakStart: 780, BreakEnd: 825, DaysOff: []int{2}, ServiceIDs: []string{services[0].ID, services[1].ID, services[2].ID}},
	}
	for _, barber := range barbers {
		tag, err := tx.Exec(ctx, `INSERT INTO barbers(id,name,specialty,starts_at,ends_at,break_start,break_end,days_off) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO NOTHING`,
			barber.ID, barber.Name, barber.Specialty, barber.Start, barber.End, barber.BreakStart, barber.BreakEnd, barber.DaysOff)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		for _, serviceID := range barber.ServiceIDs {
			if _, err = tx.Exec(ctx, "INSERT INTO barber_services(barber_id,service_id) VALUES($1,$2)", barber.ID, serviceID); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

// CreateOwner is deliberately available through the server CLI, never a public API.
func CreateOwner(ctx context.Context, pool *pgxpool.Pool, email, name, password string) error {
	email, err := cleanEmail(email)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 80 {
		return errors.New("nama admin harus berisi 2 sampai 80 karakter")
	}
	hash, err := PasswordHash(password)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, "INSERT INTO staff(id,email,name,password_hash,role) VALUES($1,$2,$3,$4,'admin')", newID(), email, name, hash)
	return err
}

// Ready prevents an old application from silently starting with a newer schema.
func Ready(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	var count int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil {
		return errors.New("database belum dimigrasikan, jalankan perintah migrate")
	}
	if count != len(files) {
		return errors.New("versi skema berbeda, jalankan migrasi yang sesuai dengan rilis ini")
	}
	for _, file := range files {
		var found bool
		err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)", "migrations/"+file.Name()).Scan(&found)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("migrasi belum lengkap")
		}
	}
	return nil
}
