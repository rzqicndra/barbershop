package app

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func loadCatalog(ctx context.Context, db queryer, public bool) (Catalog, error) {
	catalog := Catalog{Services: []Service{}, Barbers: []Barber{}, TimeOff: []TimeOff{}}
	err := db.QueryRow(ctx, "SELECT name, location, opens_at, closes_at, closed_dates, version FROM shop WHERE id=1").
		Scan(&catalog.Profile.Name, &catalog.Profile.Location, &catalog.Profile.Open, &catalog.Profile.Close, &catalog.Profile.ClosedDates, &catalog.Profile.Version)
	if err != nil {
		return catalog, err
	}
	serviceRows, err := db.Query(ctx, "SELECT id::text,name,description,price,duration,active,version FROM services WHERE NOT $1 OR active ORDER BY created_order,id", public)
	if err != nil {
		return catalog, err
	}
	for serviceRows.Next() {
		var service Service
		if err := serviceRows.Scan(&service.ID, &service.Name, &service.Description, &service.Price, &service.Duration, &service.Active, &service.Version); err != nil {
			serviceRows.Close()
			return catalog, err
		}
		catalog.Services = append(catalog.Services, service)
	}
	err = serviceRows.Err()
	serviceRows.Close()
	if err != nil {
		return catalog, err
	}
	barberRows, err := db.Query(ctx, `SELECT b.id::text,b.name,b.specialty,b.starts_at,b.ends_at,b.break_start,b.break_end,b.days_off,b.active,b.version,
		COALESCE(array_agg(bs.service_id::text ORDER BY bs.service_id) FILTER(WHERE bs.service_id IS NOT NULL),'{}')
		FROM barbers b LEFT JOIN barber_services bs ON bs.barber_id=b.id
		WHERE NOT $1 OR b.active GROUP BY b.id ORDER BY b.created_order,b.id`, public)
	if err != nil {
		return catalog, err
	}
	for barberRows.Next() {
		var barber Barber
		err := barberRows.Scan(&barber.ID, &barber.Name, &barber.Specialty, &barber.Start, &barber.End, &barber.BreakStart, &barber.BreakEnd,
			&barber.DaysOff, &barber.Active, &barber.Version, &barber.ServiceIDs)
		if err != nil {
			barberRows.Close()
			return catalog, err
		}
		catalog.Barbers = append(catalog.Barbers, barber)
	}
	err = barberRows.Err()
	barberRows.Close()
	if err != nil {
		return catalog, err
	}
	if public {
		return catalog, nil
	}
	blockRows, err := db.Query(ctx, "SELECT id::text,barber_id::text,date,starts_at,ends_at,reason FROM time_off ORDER BY date,starts_at")
	if err != nil {
		return catalog, err
	}
	defer blockRows.Close()
	for blockRows.Next() {
		var block TimeOff
		if err := blockRows.Scan(&block.ID, &block.BarberID, &block.Date, &block.Start, &block.End, &block.Reason); err != nil {
			return catalog, err
		}
		if public {
			block.Reason = ""
		}
		catalog.TimeOff = append(catalog.TimeOff, block)
	}
	return catalog, blockRows.Err()
}

func locked(ctx context.Context, a *App, fn func(pgx.Tx) error) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT id FROM shop WHERE id=1 FOR UPDATE"); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
