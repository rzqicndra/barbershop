DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM bookings WHERE status='serving' GROUP BY barber_id HAVING count(*) > 1) THEN
        RAISE EXCEPTION 'Ada barber dengan lebih dari satu kunjungan Sedang dilayani. Periksa status kunjungan pada versi lama sebelum menerapkan migrasi ini.';
    END IF;
END;
$$;

CREATE UNIQUE INDEX one_serving_per_barber ON bookings(barber_id) WHERE status='serving';
