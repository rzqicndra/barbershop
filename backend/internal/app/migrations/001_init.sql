CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA public;

CREATE TABLE shop (
    id integer PRIMARY KEY CHECK (id = 1),
    name text NOT NULL,
    location text NOT NULL,
    opens_at integer NOT NULL CHECK (opens_at >= 0 AND opens_at < 1440),
    closes_at integer NOT NULL CHECK (closes_at > opens_at AND closes_at <= 1440),
    closed_dates text[] NOT NULL DEFAULT '{}',
    version integer NOT NULL DEFAULT 1
);
INSERT INTO shop (id, name, location, opens_at, closes_at)
VALUES (1, 'Gilir Barbershop', 'Gading Serpong, Tangerang', 600, 1260);

CREATE TABLE services (
    id uuid PRIMARY KEY,
    created_order bigint GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL CHECK (length(name) BETWEEN 2 AND 80),
    description text NOT NULL DEFAULT '',
    price integer NOT NULL CHECK (price >= 0),
    duration integer NOT NULL CHECK (duration BETWEEN 15 AND 180),
    active boolean NOT NULL DEFAULT true,
    version integer NOT NULL DEFAULT 1
);

CREATE TABLE barbers (
    id uuid PRIMARY KEY,
    created_order bigint GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL CHECK (length(name) BETWEEN 2 AND 80),
    specialty text NOT NULL DEFAULT '',
    starts_at integer NOT NULL CHECK (starts_at >= 0 AND starts_at < 1440),
    ends_at integer NOT NULL CHECK (ends_at > starts_at AND ends_at <= 1440),
    break_start integer NOT NULL,
    break_end integer NOT NULL CHECK (break_end >= break_start),
    days_off integer[] NOT NULL DEFAULT '{}',
    active boolean NOT NULL DEFAULT true,
    version integer NOT NULL DEFAULT 1,
    CHECK (days_off <@ ARRAY[0,1,2,3,4,5,6]),
    CHECK (break_start = break_end OR (break_start >= starts_at AND break_end <= ends_at))
);

CREATE TABLE barber_services (
    barber_id uuid REFERENCES barbers(id),
    service_id uuid REFERENCES services(id),
    PRIMARY KEY (barber_id, service_id)
);

CREATE TABLE staff (
    id uuid PRIMARY KEY,
    email text NOT NULL UNIQUE CHECK (email = lower(email)),
    name text NOT NULL,
    password_hash text NOT NULL,
    role text NOT NULL CHECK (role IN ('admin', 'barber')),
    barber_id uuid UNIQUE REFERENCES barbers(id),
    active boolean NOT NULL DEFAULT true,
    CHECK ((role = 'barber' AND barber_id IS NOT NULL) OR (role = 'admin' AND barber_id IS NULL))
);

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    staff_id uuid NOT NULL REFERENCES staff(id),
    csrf_token text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_staff_idx ON sessions(staff_id);
CREATE INDEX sessions_expiry_idx ON sessions(expires_at);

CREATE TABLE time_off (
    id uuid PRIMARY KEY,
    barber_id uuid NOT NULL REFERENCES barbers(id),
    date text NOT NULL CHECK (date ~ '^\d{4}-\d{2}-\d{2}$'),
    starts_at integer NOT NULL CHECK (starts_at >= 0),
    ends_at integer NOT NULL CHECK (ends_at > starts_at AND ends_at <= 1440),
    reason text NOT NULL DEFAULT ''
);
CREATE INDEX time_off_barber_date_idx ON time_off(barber_id, date);

CREATE TABLE bookings (
    id uuid PRIMARY KEY,
    code text NOT NULL UNIQUE,
    service_id uuid NOT NULL REFERENCES services(id),
    service_name text NOT NULL,
    duration integer NOT NULL CHECK (duration > 0),
    price integer NOT NULL CHECK (price >= 0),
    barber_id uuid NOT NULL REFERENCES barbers(id),
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL CHECK (ends_at > starts_at),
    customer text NOT NULL,
    phone text NOT NULL DEFAULT '',
    source text NOT NULL CHECK (source IN ('online', 'walk-in')),
    status text NOT NULL CHECK (status IN ('confirmed', 'serving', 'completed', 'cancelled', 'no-show')),
    token_nonce text NOT NULL,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT barber_slot_exclusion EXCLUDE USING gist
      (barber_id WITH =, tstzrange(starts_at, ends_at, '[)') WITH &&)
      WHERE (status IN ('confirmed', 'serving', 'completed'))
);
CREATE INDEX bookings_date_idx ON bookings(starts_at);

CREATE TABLE booking_history (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    booking_id uuid NOT NULL REFERENCES bookings(id),
    action text NOT NULL,
    actor_id uuid REFERENCES staff(id),
    actor_name text NOT NULL,
    detail text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE booking_requests (
    key uuid NOT NULL,
    actor_scope text NOT NULL,
    payload_hash bytea NOT NULL,
    response jsonb NOT NULL,
    booking_id uuid NOT NULL REFERENCES bookings(id),
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (key, actor_scope)
);
CREATE INDEX booking_requests_expiry_idx ON booking_requests(expires_at);
