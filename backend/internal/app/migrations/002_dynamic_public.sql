ALTER TABLE services ADD COLUMN deleted_at timestamptz;
ALTER TABLE services ADD COLUMN category text NOT NULL DEFAULT 'Potong & styling';
ALTER TABLE services DROP CONSTRAINT services_duration_check;
ALTER TABLE services ADD CONSTRAINT services_duration_check CHECK (duration BETWEEN 15 AND 360);
ALTER TABLE barbers ADD COLUMN deleted_at timestamptz;
ALTER TABLE staff ADD COLUMN deleted_at timestamptz;
ALTER TABLE staff DROP CONSTRAINT staff_email_key;
ALTER TABLE staff DROP CONSTRAINT staff_barber_id_key;
CREATE UNIQUE INDEX staff_email_live_idx ON staff(email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX staff_barber_live_idx ON staff(barber_id) WHERE deleted_at IS NULL;

ALTER TABLE shop ADD COLUMN maps_url text NOT NULL DEFAULT '';
ALTER TABLE shop ADD COLUMN maps_embed_url text NOT NULL DEFAULT '';
ALTER TABLE shop ADD COLUMN instagram_url text NOT NULL DEFAULT '';
ALTER TABLE shop ADD COLUMN whatsapp text NOT NULL DEFAULT '';
ALTER TABLE shop ADD COLUMN wordmark text NOT NULL DEFAULT 'gilir';
ALTER TABLE shop ADD COLUMN logo_image_url text NOT NULL DEFAULT '';
ALTER TABLE shop ADD COLUMN hero_eyebrow text NOT NULL DEFAULT 'BARBERSHOP DI TANGERANG';
ALTER TABLE shop ADD COLUMN hero_title text NOT NULL DEFAULT E'Pilih jam.\nPotong rapi.';
ALTER TABLE shop ADD COLUMN hero_description text NOT NULL DEFAULT 'Pilih layanan, barber, dan jam yang pas. Datang sesuai jadwal, lalu bayar langsung di barbershop.';
ALTER TABLE shop ADD COLUMN hero_image_url text NOT NULL DEFAULT '/assets/gilir-chair.webp';
ALTER TABLE shop ADD COLUMN footer_tagline text NOT NULL DEFAULT 'Pilih jam. Potong rapi.';
ALTER TABLE shop ADD COLUMN gallery_title text NOT NULL DEFAULT 'Hasil dari kursi kami.';
ALTER TABLE shop ADD COLUMN gallery_description text NOT NULL DEFAULT 'Pilihan hasil potongan dan perawatan dari tim barber kami.';

ALTER TABLE bookings ADD COLUMN barber_name text NOT NULL DEFAULT '';
UPDATE bookings b SET barber_name = p.name FROM barbers p WHERE p.id = b.barber_id;

CREATE TABLE media_assets (
    id uuid PRIMARY KEY,
    filename text NOT NULL UNIQUE,
    width integer NOT NULL CHECK (width > 0),
    height integer NOT NULL CHECK (height > 0),
    size_bytes integer NOT NULL CHECK (size_bytes > 0),
    uploaded_by uuid NOT NULL REFERENCES staff(id),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE gallery_photos (
    id uuid PRIMARY KEY,
    image_url text NOT NULL,
    title text NOT NULL CHECK (length(title) BETWEEN 2 AND 100),
    caption text NOT NULL DEFAULT '',
    alt_text text NOT NULL CHECK (length(alt_text) BETWEEN 5 AND 200),
    sort_order integer NOT NULL DEFAULT 0,
    active boolean NOT NULL DEFAULT false,
    consent_confirmed boolean NOT NULL DEFAULT false,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (NOT active OR consent_confirmed)
);
CREATE TABLE faqs (
    id uuid PRIMARY KEY,
    question text NOT NULL CHECK (length(question) BETWEEN 5 AND 150),
    answer text NOT NULL CHECK (length(answer) BETWEEN 5 AND 1500),
    sort_order integer NOT NULL DEFAULT 0,
    active boolean NOT NULL DEFAULT true,
    version integer NOT NULL DEFAULT 1
);
INSERT INTO faqs(id, question, answer, sort_order) VALUES
('10000000-0000-4000-8000-000000000001', 'Perlu buat akun untuk booking?', 'Cukup isi nama dan nomor WhatsApp. Setelah konfirmasi, simpan tautan privat yang muncul untuk membuka kembali booking kamu.', 1),
('10000000-0000-4000-8000-000000000002', 'Kalau jadwal berubah?', 'Buka tautan privat dari halaman konfirmasi. Kamu bisa mengubah jadwal atau membatalkan booking paling lambat dua jam sebelum jam kunjungan.', 2),
('10000000-0000-4000-8000-000000000003', 'Kapan saya membayar?', 'Pembayaran dilakukan di barbershop setelah kunjungan.', 3);

CREATE TABLE reviews (
    id uuid PRIMARY KEY,
    booking_id uuid NOT NULL UNIQUE REFERENCES bookings(id),
    rating integer NOT NULL CHECK (rating BETWEEN 1 AND 5),
    display_name text NOT NULL CHECK (length(display_name) BETWEEN 2 AND 40),
    comment text NOT NULL DEFAULT '' CHECK (length(comment) <= 1000),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE FUNCTION require_completed_review() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM id FROM bookings WHERE id = NEW.booking_id AND status = 'completed' FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Reviews require a completed appointment' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER reviews_completed_only BEFORE INSERT OR UPDATE ON reviews FOR EACH ROW EXECUTE FUNCTION require_completed_review();
