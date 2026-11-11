package app

import (
	"context"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

var iframeSource = regexp.MustCompile(`(?i)\bsrc\s*=\s*["']([^"']+)["']`)
var instagramName = regexp.MustCompile(`^[A-Za-z0-9._]{1,30}$`)

func cleanPublicURL(raw, kind string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if kind == "embed" && strings.HasPrefix(raw, "<") {
		source := iframeSource.FindStringSubmatch(raw)
		if len(source) != 2 {
			return "", invalid("Kode sematan Maps tidak memiliki URL src yang valid.")
		}
		raw = html.UnescapeString(source[1])
	}
	if kind == "instagram" && !strings.Contains(raw, "://") {
		name := strings.TrimPrefix(raw, "@")
		if instagramName.MatchString(name) {
			raw = "https://www.instagram.com/" + name + "/"
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" || len(raw) > 4000 {
		return "", invalid("Tautan harus menggunakan HTTPS yang valid.")
	}
	host := strings.ToLower(parsed.Hostname())
	allowed := false
	switch kind {
	case "instagram":
		allowed = host == "www.instagram.com" || host == "instagram.com"
	case "maps":
		allowed = (host == "www.google.com" || host == "google.com") && strings.HasPrefix(parsed.Path, "/maps") || host == "maps.google.com" || host == "maps.app.goo.gl"
	case "embed":
		allowed = (host == "www.google.com" || host == "google.com") && (parsed.Path == "/maps/embed" || strings.HasPrefix(parsed.Path, "/maps/embed/"))
	}
	if !allowed {
		return "", invalid("Gunakan tautan resmi Google Maps atau Instagram sesuai kolomnya.")
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func validatePublicProfile(ctx context.Context, db queryer, p *Profile) error {
	var err error
	if p.MapsURL, err = cleanPublicURL(p.MapsURL, "maps"); err != nil {
		return err
	}
	if p.MapsEmbedURL, err = cleanPublicURL(p.MapsEmbedURL, "embed"); err != nil {
		return err
	}
	if p.InstagramURL, err = cleanPublicURL(p.InstagramURL, "instagram"); err != nil {
		return err
	}
	p.WhatsApp = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(p.WhatsApp))
	if p.WhatsApp != "" {
		if !phonePattern.MatchString(p.WhatsApp) {
			return invalid("Nomor WhatsApp usaha tidak valid.")
		}
		p.WhatsApp = strings.TrimPrefix(p.WhatsApp, "+")
		if strings.HasPrefix(p.WhatsApp, "0") {
			p.WhatsApp = "62" + p.WhatsApp[1:]
		}
	}
	p.Wordmark = strings.TrimSpace(p.Wordmark)
	p.HeroTitle = strings.TrimSpace(p.HeroTitle)
	p.HeroEyebrow = strings.TrimSpace(p.HeroEyebrow)
	p.HeroDescription = strings.TrimSpace(p.HeroDescription)
	p.FooterTagline = strings.TrimSpace(p.FooterTagline)
	p.GalleryTitle = strings.TrimSpace(p.GalleryTitle)
	p.GalleryDescription = strings.TrimSpace(p.GalleryDescription)
	if utf8.RuneCountInString(p.Wordmark) < 2 || utf8.RuneCountInString(p.Wordmark) > 30 ||
		utf8.RuneCountInString(p.HeroTitle) < 2 || utf8.RuneCountInString(p.HeroTitle) > 100 || strings.Count(p.HeroTitle, "\n") > 2 ||
		utf8.RuneCountInString(p.HeroEyebrow) > 80 || utf8.RuneCountInString(p.HeroDescription) < 5 || utf8.RuneCountInString(p.HeroDescription) > 500 ||
		utf8.RuneCountInString(p.FooterTagline) > 120 || utf8.RuneCountInString(p.GalleryTitle) < 2 || utf8.RuneCountInString(p.GalleryTitle) > 100 || utf8.RuneCountInString(p.GalleryDescription) > 300 {
		return invalid("Periksa panjang nama logo, judul, dan deskripsi halaman publik.")
	}
	if p.HeroImageURL == "" {
		p.HeroImageURL = "/assets/gilir-chair.webp"
	}
	if err = validMediaURL(ctx, db, p.HeroImageURL, true); err != nil {
		return err
	}
	if p.LogoImageURL != "" {
		return validMediaURL(ctx, db, p.LogoImageURL, false)
	}
	return nil
}

func validMediaURL(ctx context.Context, db queryer, value string, allowDefault bool) error {
	if allowDefault && value == "/assets/gilir-chair.webp" {
		return nil
	}
	if !strings.HasPrefix(value, "/media/") || !mediaFilename.MatchString(strings.TrimPrefix(value, "/media/")) {
		return invalid("Pilih foto yang sudah diunggah melalui website.")
	}
	var exists bool
	if err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM media_assets WHERE filename=$1)", strings.TrimPrefix(value, "/media/")).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return invalid("Foto unggahan tidak ditemukan.")
	}
	return nil
}

func loadContent(ctx context.Context, db queryer, catalog *Catalog, public bool) error {
	gallery, err := db.Query(ctx, `SELECT id::text,image_url,title,caption,alt_text,sort_order,active,consent_confirmed,version
        FROM gallery_photos WHERE NOT $1 OR (active AND consent_confirmed) ORDER BY sort_order,created_at,id`, public)
	if err != nil {
		return err
	}
	for gallery.Next() {
		var photo GalleryPhoto
		if err = gallery.Scan(&photo.ID, &photo.ImageURL, &photo.Title, &photo.Caption, &photo.AltText, &photo.SortOrder, &photo.Active, &photo.ConsentConfirmed, &photo.Version); err != nil {
			gallery.Close()
			return err
		}
		catalog.Gallery = append(catalog.Gallery, photo)
	}
	err = gallery.Err()
	gallery.Close()
	if err != nil {
		return err
	}
	faqs, err := db.Query(ctx, "SELECT id::text,question,answer,sort_order,active,version FROM faqs WHERE NOT $1 OR active ORDER BY sort_order,id", public)
	if err != nil {
		return err
	}
	for faqs.Next() {
		var faq FAQ
		if err = faqs.Scan(&faq.ID, &faq.Question, &faq.Answer, &faq.SortOrder, &faq.Active, &faq.Version); err != nil {
			faqs.Close()
			return err
		}
		catalog.FAQs = append(catalog.FAQs, faq)
	}
	err = faqs.Err()
	faqs.Close()
	if err != nil {
		return err
	}
	catalog.Reviews, err = readReviews(ctx, db, 0, 12)
	if err != nil {
		return err
	}
	return db.QueryRow(ctx, "SELECT COALESCE(avg(rating),0)::float8,count(*) FROM reviews").Scan(&catalog.Rating.Average, &catalog.Rating.Count)
}

func (a *App) saveGallery(w http.ResponseWriter, r *http.Request) error {
	var input GalleryPhoto
	if err := decode(r, &input); err != nil {
		return err
	}
	input.Title, input.Caption, input.AltText = strings.TrimSpace(input.Title), strings.TrimSpace(input.Caption), strings.TrimSpace(input.AltText)
	if utf8.RuneCountInString(input.Title) < 2 || utf8.RuneCountInString(input.Title) > 100 || utf8.RuneCountInString(input.AltText) < 5 || utf8.RuneCountInString(input.AltText) > 200 || utf8.RuneCountInString(input.Caption) > 300 || input.SortOrder < 0 || input.SortOrder > 10000 || (input.Active && !input.ConsentConfirmed) {
		return invalid("Isi judul dan deskripsi foto. Konfirmasi hak penggunaan foto sebelum memublikasikannya.")
	}
	id := chi.URLParam(r, "id")
	creating := id == ""
	if creating {
		id = newID()
	} else if !validID(id) {
		return invalid("Referensi foto tidak valid.")
	}
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		if err := validMediaURL(r.Context(), tx, input.ImageURL, false); err != nil {
			return err
		}
		if creating {
			var count int
			if err := tx.QueryRow(r.Context(), "SELECT count(*) FROM gallery_photos").Scan(&count); err != nil {
				return err
			}
			if count >= 60 {
				return invalid("Galeri maksimal 60 foto. Hapus foto lama sebelum menambahkan.")
			}
			_, err := tx.Exec(r.Context(), `INSERT INTO gallery_photos(id,image_url,title,caption,alt_text,sort_order,active,consent_confirmed) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, id, input.ImageURL, input.Title, input.Caption, input.AltText, input.SortOrder, input.Active, input.ConsentConfirmed)
			return err
		}
		tag, err := tx.Exec(r.Context(), `UPDATE gallery_photos SET image_url=$1,title=$2,caption=$3,alt_text=$4,sort_order=$5,active=$6,consent_confirmed=$7,version=version+1 WHERE id=$8 AND version=$9`, input.ImageURL, input.Title, input.Caption, input.AltText, input.SortOrder, input.Active, input.ConsentConfirmed, id, input.Version)
		if err == nil && tag.RowsAffected() != 1 {
			return conflict("SETTINGS_CHANGED", "Foto sudah berubah atau dihapus. Muat ulang galeri.")
		}
		return err
	})
	if err != nil {
		return err
	}
	return a.adminCatalog(w, r)
}

func (a *App) saveFAQ(w http.ResponseWriter, r *http.Request) error {
	var input FAQ
	if err := decode(r, &input); err != nil {
		return err
	}
	input.Question, input.Answer = strings.TrimSpace(input.Question), strings.TrimSpace(input.Answer)
	if utf8.RuneCountInString(input.Question) < 5 || utf8.RuneCountInString(input.Question) > 150 || utf8.RuneCountInString(input.Answer) < 5 || utf8.RuneCountInString(input.Answer) > 1500 || input.SortOrder < 0 || input.SortOrder > 10000 {
		return invalid("Periksa pertanyaan, jawaban, dan urutan FAQ.")
	}
	id := chi.URLParam(r, "id")
	if id != "" && !validID(id) {
		return invalid("Referensi FAQ tidak valid.")
	}
	err := locked(r.Context(), a, func(tx pgx.Tx) error {
		if id == "" {
			var count int
			if err := tx.QueryRow(r.Context(), "SELECT count(*) FROM faqs").Scan(&count); err != nil {
				return err
			}
			if count >= 30 {
				return invalid("FAQ maksimal 30 pertanyaan.")
			}
			_, err := tx.Exec(r.Context(), "INSERT INTO faqs(id,question,answer,sort_order,active) VALUES($1,$2,$3,$4,$5)", newID(), input.Question, input.Answer, input.SortOrder, input.Active)
			if err != nil {
				return err
			}
		} else {
			tag, err := tx.Exec(r.Context(), "UPDATE faqs SET question=$1,answer=$2,sort_order=$3,active=$4,version=version+1 WHERE id=$5 AND version=$6", input.Question, input.Answer, input.SortOrder, input.Active, id, input.Version)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return conflict("SETTINGS_CHANGED", "FAQ sudah berubah atau dihapus. Muat ulang halaman.")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return a.adminCatalog(w, r)
}

func (a *App) deleteContent(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		return invalid("Referensi konten tidak valid.")
	}
	var input struct {
		Version int `json:"version"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	table := "faqs"
	if strings.Contains(r.URL.Path, "/gallery/") {
		table = "gallery_photos"
	}
	// The table is selected from two constants, never from request input.
	tag, err := a.db.Exec(r.Context(), "DELETE FROM "+table+" WHERE id=$1 AND version=$2", id, input.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return conflict("SETTINGS_CHANGED", "Konten sudah berubah atau dihapus. Muat ulang halaman.")
	}
	return a.adminCatalog(w, r)
}
