package app

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func loadCatalog(ctx context.Context, db queryer, public bool) (Catalog, error) {
	catalog := Catalog{Services: []Service{}, Barbers: []Barber{}, TimeOff: []TimeOff{}, Gallery: []GalleryPhoto{}, FAQs: []FAQ{}, Reviews: []Review{}}
	err := db.QueryRow(ctx, `SELECT name, location, opens_at, closes_at, closed_dates, version,
 maps_url,maps_embed_url,instagram_url,whatsapp,wordmark,logo_image_url,hero_eyebrow,hero_title,hero_description,hero_image_url,footer_tagline,gallery_title,gallery_description FROM shop WHERE id=1`).
		Scan(&catalog.Profile.Name, &catalog.Profile.Location, &catalog.Profile.Open, &catalog.Profile.Close, &catalog.Profile.ClosedDates, &catalog.Profile.Version,
			&catalog.Profile.MapsURL, &catalog.Profile.MapsEmbedURL, &catalog.Profile.InstagramURL, &catalog.Profile.WhatsApp, &catalog.Profile.Wordmark,
			&catalog.Profile.LogoImageURL, &catalog.Profile.HeroEyebrow, &catalog.Profile.HeroTitle, &catalog.Profile.HeroDescription, &catalog.Profile.HeroImageURL,
			&catalog.Profile.FooterTagline, &catalog.Profile.GalleryTitle, &catalog.Profile.GalleryDescription)
	if err != nil {
		return catalog, err
	}
	serviceRows, err := db.Query(ctx, "SELECT id::text,name,description,price,duration,active,version,category FROM services WHERE deleted_at IS NULL AND (NOT $1 OR active) ORDER BY created_order,id", public)
	if err != nil {
		return catalog, err
	}
	for serviceRows.Next() {
		var service Service
		if err := serviceRows.Scan(&service.ID, &service.Name, &service.Description, &service.Price, &service.Duration, &service.Active, &service.Version, &service.Category); err != nil {
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
		FROM barbers b LEFT JOIN barber_services bs ON bs.barber_id=b.id AND EXISTS(SELECT 1 FROM services s WHERE s.id=bs.service_id AND s.deleted_at IS NULL)
		WHERE b.deleted_at IS NULL AND (NOT $1 OR b.active) GROUP BY b.id ORDER BY b.created_order,b.id`, public)
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
	blockRows, err := db.Query(ctx, "SELECT id::text,barber_id::text,date,starts_at,ends_at,reason FROM time_off WHERE barber_id IN (SELECT id FROM barbers WHERE deleted_at IS NULL) ORDER BY date,starts_at")
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
