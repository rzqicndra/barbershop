import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import {
  ArrowLeft,
  ArrowRight,
  CalendarDays,
  Check,
  Clock3,
  Copy,
  Link2,
  Scissors,
  UserRound,
} from 'lucide-react'
import {
  addDays,
  canManage,
  dateFormat,
  longDate,
  money,
  shortDate,
  suitableBarbers,
  time,
  today,
  validPhone,
  type Booking,
  type Service,
  type StudioProps,
} from './data'
import { BackLink, BookingLink, Empty, Kicker, Modal, Status } from './ui'
import { PublicFooter, PublicHeader } from './public'
import { BookingReview } from './reviews'
import { api, APIError, availabilityPath, messageOf, useSlots } from './api'
import { randomId } from './random-id'

export function DateStrip({ date, onChange }: { date: string; onChange: (value: string) => void }) {
  const first = today()
  const end = addDays(first, 30)
  const weekStart =
    date >= first && date <= addDays(first, 6)
      ? first
      : date > addDays(end, -6)
        ? addDays(end, -6)
        : date
  return (
    <div className="date-picker">
      <div className="field-heading">
        <h3>Pilih tanggal</h3>
        <label className="date-jump">
          <span>Tanggal lain</span>
          <input
            type="date"
            aria-label="Pilih tanggal lain"
            min={first}
            max={end}
            value={date}
            onInput={(e) => {
              if (e.currentTarget.value >= first && e.currentTarget.value <= end)
                onChange(e.currentTarget.value)
            }}
          />
        </label>
      </div>
      <div className="date-strip" aria-label="Tanggal kunjungan">
        {Array.from({ length: 7 }, (_, index) => {
          const day = addDays(weekStart, index)
          return (
            <button
              type="button"
              key={day}
              className={'day-choice ' + (date === day ? 'selected' : '')}
              aria-pressed={date === day}
              aria-label={longDate(day)}
              onClick={() => onChange(day)}
            >
              <span>{day === first ? 'Hari ini' : dateFormat(day, { weekday: 'short' })}</span>
              <strong>{dateFormat(day, { day: 'numeric' })}</strong>
              <small>{dateFormat(day, { month: 'short' })}</small>
            </button>
          )
        })}
      </div>
    </div>
  )
}

const periods = [
  { name: 'Pagi', from: 0, to: 720 },
  { name: 'Siang', from: 720, to: 960 },
  { name: 'Sore', from: 960, to: 1440 },
]

export function SlotGrid({
  times,
  start,
  onChange,
}: {
  times: number[]
  start: number | null
  onChange: (value: number) => void
}) {
  const groups = periods.map((p) => times.filter((t) => t >= p.from && t < p.to))
  const [period, setPeriod] = useState(() => {
    const selected = start === null ? -1 : groups.findIndex((group) => group.includes(start))
    return selected >= 0
      ? selected
      : Math.max(
          0,
          groups.findIndex((group) => group.length > 0),
        )
  })
  if (!times.length)
    return (
      <Empty title="Belum ada jam tersedia">
        Coba pilih tanggal atau barber lain untuk layanan ini.
      </Empty>
    )
  const active = groups[period]?.length ? period : groups.findIndex((g) => g.length > 0)
  return (
    <section className="slot-picker" aria-label="Jam kunjungan">
      <div className="field-heading">
        <h3>Pilih jam</h3>
        <span className="muted">{times.length} pilihan · WIB</span>
      </div>
      <div className="periods" aria-label="Waktu dalam sehari">
        {periods.map((p, i) => (
          <button
            type="button"
            key={p.name}
            className={i === active ? 'active' : ''}
            aria-pressed={i === active}
            disabled={groups[i].length === 0}
            onClick={() => setPeriod(i)}
          >
            {p.name}
            <small>{groups[i].length}</small>
          </button>
        ))}
      </div>
      <div className="slot-grid">
        {groups[active].map((slot) => (
          <button
            type="button"
            className={start === slot ? 'selected' : ''}
            aria-pressed={start === slot}
            key={slot}
            onClick={() => onChange(slot)}
          >
            {time(slot)}
          </button>
        ))}
      </div>
    </section>
  )
}

function BookingSteps({ step, goTo }: { step: number; goTo: (value: number) => void }) {
  return (
    <ol className="booking-steps" aria-label="Tahapan booking">
      {['Layanan', 'Barber & waktu', 'Konfirmasi'].map((name, index) => (
        <li
          key={name}
          className={index === step ? 'current' : index < step ? 'done' : ''}
          aria-current={index === step ? 'step' : undefined}
        >
          <button type="button" disabled={index >= step} onClick={() => goTo(index)}>
            <span className="step-number">
              {index < step ? <Check size={13} aria-hidden="true" /> : index + 1}
            </span>
            <span>{name}</span>
          </button>
        </li>
      ))}
    </ol>
  )
}

function BookingSummary({
  service,
  barber,
  date,
  start,
}: {
  service?: Service
  barber?: string
  date: string
  start: number | null
}) {
  return (
    <aside className="booking-summary" aria-label="Ringkasan booking">
      <div className="summary-head">
        <Kicker>RENCANA KUNJUNGAN</Kicker>
        <Scissors size={21} aria-hidden="true" />
      </div>
      <h2>{service?.name ?? 'Pilih layanan'}</h2>
      <dl>
        <div>
          <dt>
            <UserRound size={15} aria-hidden="true" />
            Barber
          </dt>
          <dd>{barber ?? 'Belum dipilih'}</dd>
        </div>
        <div>
          <dt>
            <CalendarDays size={15} aria-hidden="true" />
            Tanggal
          </dt>
          <dd>{barber ? shortDate(date) : 'Belum dipilih'}</dd>
        </div>
        <div>
          <dt>
            <Clock3 size={15} aria-hidden="true" />
            Jam
          </dt>
          <dd>
            {start !== null && service
              ? time(start) + '–' + time(start + service.duration) + ' WIB'
              : 'Belum dipilih'}
          </dd>
        </div>
      </dl>
      <div className="summary-total">
        <span>Bayar di tempat</span>
        <strong>{service ? money(service.price) : '—'}</strong>
      </div>
    </aside>
  )
}

export function BookingWizard({
  studio,
  refresh,
  initialService,
}: StudioProps & { initialService: string }) {
  const initial = studio.services.find((s) => s.id === initialService && s.active)
  const [step, setStep] = useState(initial ? 1 : 0)
  const [serviceId, setService] = useState(initial?.id ?? '')
  const [barberId, setBarber] = useState('')
  const [date, setDate] = useState(today())
  const [start, setStart] = useState<number | null>(null)
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [errors, setErrors] = useState<{ name?: string; phone?: string; form?: string }>({})
  const lock = useRef(false)
  const [pending, setPending] = useState(false)
  const request = useRef({ fingerprint: '', key: '' })
  const heading = useRef<HTMLHeadingElement>(null)
  const service = studio.services.find((s) => s.id === serviceId && s.active)
  const barbers = suitableBarbers(studio, serviceId)
  const barber = barbers.find((b) => b.id === barberId)
  const availability = useSlots(
    service && barber ? availabilityPath(service.id, barber.id, date) : null,
  )
  const times = availability.slots
  const ready = start !== null && times.includes(start)

  function goTo(value: number) {
    setStep(value)
    requestAnimationFrame(() => {
      heading.current?.focus({ preventScroll: true })
      window.scrollTo({ top: 0, behavior: 'instant' })
    })
  }

  function selectService(id: string) {
    if (id !== serviceId) {
      setService(id)
      setBarber('')
      setStart(null)
    }
    setErrors({})
  }

  function chooseBarber(id: string) {
    if (id !== barberId) {
      setBarber(id)
      setStart(null)
    }
    setErrors({})
  }

  function selectDate(value: string) {
    if (value !== date) {
      setDate(value)
      setStart(null)
    }
    setErrors({})
  }

  function continueToTime() {
    const firstBarber = barber ?? barbers[0]
    if (!firstBarber || !service) return
    setBarber(firstBarber.id)
    goTo(1)
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!service || !barber || start === null || lock.current) return
    const invalid = {
      name: name.trim().length < 2 ? 'Masukkan nama minimal 2 karakter.' : undefined,
      phone: !validPhone(phone)
        ? 'Gunakan nomor WhatsApp Indonesia, misalnya 081200000000.'
        : undefined,
    }
    if (invalid.name || invalid.phone) {
      setErrors(invalid)
      return
    }
    const body = {
      serviceId: service.id,
      barberId: barber.id,
      date,
      start,
      customer: name.trim(),
      phone: phone.trim(),
      offerVersion: service.version,
    }
    const fingerprint = JSON.stringify(body)
    if (request.current.fingerprint !== fingerprint)
      request.current = { fingerprint, key: randomId() }
    lock.current = true
    setPending(true)
    setErrors({})
    try {
      const booking = await api<Booking>('/bookings', {
        method: 'POST',
        body: { ...body, requestKey: request.current.key },
      })
      window.location.hash = 'booking/' + booking.token + '?baru=1'
    } catch (err) {
      setErrors({ form: messageOf(err) })
      if (err instanceof APIError && err.code === 'OFFER_CHANGED') {
        await refresh().catch(() => {})
        setStart(null)
        goTo(0)
      } else if (err instanceof APIError && (err.code === 'SLOT_TAKEN' || err.status === 422)) {
        setStart(null)
        availability.reload()
        goTo(1)
      }
    } finally {
      lock.current = false
      setPending(false)
    }
  }

  const titles = ['Pilih layanan.', 'Pilih barber & waktu.', 'Konfirmasi booking.']
  return (
    <>
      <PublicHeader compact />
      <main id="konten" tabIndex={-1} className="booking-page narrow-wrap">
        <div className="booking-top">
          <BackLink />
          <span className="muted">Waktu Indonesia Barat</span>
        </div>
        <BookingSteps step={step} goTo={goTo} />
        <div className="booking-layout">
          <div className="booking-content">
            <h1 ref={heading} tabIndex={-1}>
              {titles[step]}
            </h1>
            <p className="page-lead">
              {
                [
                  'Pilih satu layanan untuk kunjungan ini.',
                  'Jam yang muncul mengikuti durasi layanan dan jadwal barber.',
                  'Periksa jadwal, lalu isi nama dan nomor WhatsApp.',
                ][step]
              }
            </p>
            {errors.form && (
              <p className="notice notice-error" role="alert">
                {errors.form}
              </p>
            )}
            {step === 0 && (
              <>
                <div className="service-choices" aria-label="Layanan">
                  {studio.services
                    .filter((s) => s.active)
                    .map((s, index) => (
                      <button
                        type="button"
                        key={s.id}
                        aria-pressed={serviceId === s.id}
                        className={'service-choice ' + (serviceId === s.id ? 'selected' : '')}
                        onClick={() => selectService(s.id)}
                      >
                        <span className="choice-index">{String(index + 1).padStart(2, '0')}</span>
                        <span className="choice-text">
                          <strong>{s.name}</strong>
                          <span>{s.description}</span>
                        </span>
                        <span className="choice-price">
                          <strong>{money(s.price)}</strong>
                          <small>{s.duration} menit</small>
                        </span>
                        <span className="select-mark">
                          {serviceId === s.id && <Check size={13} aria-hidden="true" />}
                        </span>
                      </button>
                    ))}
                </div>
                {!studio.services.some((s) => s.active) && (
                  <Empty title="Belum ada layanan aktif">
                    Coba kembali setelah admin membuka layanan.
                  </Empty>
                )}
                {service && barbers.length === 0 && (
                  <p className="notice">Belum ada barber untuk layanan ini. Pilih layanan lain.</p>
                )}
                <div className="wizard-actions">
                  <span>{service ? service.duration + ' menit' : 'Langkah 1 dari 3'}</span>
                  <button
                    type="button"
                    className="button button-primary"
                    disabled={!service || barbers.length === 0}
                    onClick={continueToTime}
                  >
                    Lanjut pilih jam
                    <ArrowRight size={17} />
                  </button>
                </div>
              </>
            )}
            {step === 1 && (
              <>
                <section className="barber-picker">
                  <h3>Pilih barber</h3>
                  <div className="barber-choices">
                    {barbers.map((b) => (
                      <button
                        type="button"
                        key={b.id}
                        className={'barber-choice ' + (barberId === b.id ? 'selected' : '')}
                        aria-pressed={barberId === b.id}
                        onClick={() => chooseBarber(b.id)}
                      >
                        <span className="barber-initial" aria-hidden="true">
                          {b.name[0]}
                        </span>
                        <span>
                          <strong>{b.name}</strong>
                          <small>{b.specialty}</small>
                        </span>
                        {barberId === b.id && (
                          <Check className="barber-check" size={16} aria-hidden="true" />
                        )}
                      </button>
                    ))}
                  </div>
                  {!barbers.length && (
                    <Empty title="Barber belum tersedia">
                      Kembali ke pilihan layanan untuk memilih yang lain.
                    </Empty>
                  )}
                </section>
                <DateStrip date={date} onChange={selectDate} />
                {barber &&
                  (availability.loading ? (
                    <p role="status" className="notice">
                      Memeriksa jadwal…
                    </p>
                  ) : availability.error ? (
                    <p role="alert" className="notice notice-error">
                      {availability.error}
                      <button type="button" className="text-link" onClick={availability.reload}>
                        Coba kembali
                      </button>
                    </p>
                  ) : (
                    <SlotGrid
                      key={date + barber.id}
                      times={times}
                      start={start}
                      onChange={(s) => {
                        setStart(s)
                        setErrors({})
                      }}
                    />
                  ))}
                {!barber && barbers.length > 0 && (
                  <Empty title="Pilih barber dulu">
                    Pilihan jam muncul setelah kamu memilih barber.
                  </Empty>
                )}
                <div className="wizard-actions">
                  <button type="button" className="text-link" onClick={() => goTo(0)}>
                    <ArrowLeft size={16} />
                    Kembali
                  </button>
                  <button
                    type="button"
                    className="button button-primary"
                    disabled={!ready}
                    onClick={() => goTo(2)}
                  >
                    Lanjut konfirmasi
                    <ArrowRight size={17} />
                  </button>
                </div>
              </>
            )}
            {step === 2 && (
              <form noValidate onSubmit={submit} className="checkout-form">
                <div className="form-pair">
                  <label className="field" htmlFor="customer-name">
                    <span>Nama</span>
                    <input
                      id="customer-name"
                      autoComplete="off"
                      required
                      value={name}
                      maxLength={80}
                      placeholder="Nama untuk booking"
                      aria-invalid={!!errors.name}
                      aria-describedby={errors.name ? 'name-error' : undefined}
                      onChange={(e) => setName(e.target.value)}
                    />
                    {errors.name && (
                      <small id="name-error" className="field-error">
                        {errors.name}
                      </small>
                    )}
                  </label>
                  <label className="field" htmlFor="customer-phone">
                    <span>Nomor WhatsApp</span>
                    <input
                      id="customer-phone"
                      type="tel"
                      required
                      inputMode="tel"
                      autoComplete="off"
                      value={phone}
                      maxLength={24}
                      placeholder="08…"
                      aria-invalid={!!errors.phone}
                      aria-describedby={errors.phone ? 'phone-error' : undefined}
                      onChange={(e) => setPhone(e.target.value)}
                    />
                    {errors.phone && (
                      <small id="phone-error" className="field-error">
                        {errors.phone}
                      </small>
                    )}
                  </label>
                </div>
                {service && barber && start !== null && (
                  <div className="mobile-review">
                    <strong>
                      {service.name} · {money(service.price)}
                    </strong>
                    <span>
                      {barber.name} · {longDate(date)} · {time(start)} WIB
                    </span>
                  </div>
                )}
                <div className="notice policy-note">
                  <Clock3 size={18} aria-hidden="true" />
                  <span>
                    Kamu bisa ubah jadwal atau batalkan paling lambat{' '}
                    <strong>2 jam sebelum kunjungan</strong>. Pembayaran dilakukan di barbershop.
                  </span>
                </div>
                <div className="wizard-actions">
                  <button type="button" className="text-link" onClick={() => goTo(1)}>
                    <ArrowLeft size={16} />
                    Kembali
                  </button>
                  <button
                    type="submit"
                    className="button button-primary"
                    disabled={pending || start === null || !service || !barber}
                  >
                    {pending ? 'Menyimpan…' : 'Konfirmasi booking'}
                    <ArrowRight size={17} />
                  </button>
                </div>
              </form>
            )}
          </div>
          <BookingSummary service={service} barber={barber?.name} date={date} start={start} />
        </div>
      </main>
      <PublicFooter compact />
    </>
  )
}

export function ManageDialog({
  studio,
  booking,
  actor,
  action,
  onClose,
  onUpdated,
  token,
}: StudioProps & {
  booking: Booking
  actor: 'customer' | 'admin'
  action: 'move' | 'cancel'
  onClose: () => void
  onUpdated: () => void
  token?: string
}) {
  const [date, setDate] = useState(booking.date)
  const [barberId, setBarber] = useState(booking.barberId)
  const [start, setStart] = useState<number | null>(null)
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const moving = action === 'move'
  const barberOptions = suitableBarbers(studio, booking.serviceId)
  const [pending, setPending] = useState(false)
  const base = actor === 'admin' ? '/admin/bookings/' + booking.id : '/bookings/current'
  const availability = useSlots(
    moving ? base + '/availability?' + new URLSearchParams({ date, barberId }) : null,
    token,
  )
  const times = availability.slots.filter(
    (t) => date !== booking.date || barberId !== booking.barberId || t !== booking.start,
  )

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending) return
    if (actor === 'admin' && !reason.trim()) {
      setError('Isi alasan untuk mencatat perubahan ini.')
      return
    }
    setPending(true)
    setError('')
    try {
      await api<Booking>(base + (moving ? '/reschedule' : '/cancel'), {
        method: 'POST',
        token,
        body: {
          date,
          start: start ?? booking.start,
          barberId,
          version: booking.version,
          reason: reason.trim(),
        },
      })
      onUpdated()
      onClose()
    } catch (err) {
      setError(messageOf(err))
      availability.reload()
      if (err instanceof APIError && err.code === 'BOOKING_CHANGED') onUpdated()
    } finally {
      setPending(false)
    }
  }

  return (
    <Modal title={moving ? 'Ubah jadwal' : 'Batalkan booking?'} onClose={onClose} wide={moving}>
      <form className="dialog-form" onSubmit={submit}>
        <p className="dialog-intro">
          {moving
            ? 'Jadwal lama berlaku sampai jadwal baru berhasil disimpan.'
            : 'Jam kunjungan akan dilepas agar bisa dipilih pelanggan lain.'}
        </p>
        <div className="dialog-booking-note">
          <strong>
            {booking.serviceName} · {booking.code}
          </strong>
          <span>
            {longDate(booking.date)} · {time(booking.start)} WIB
          </span>
        </div>
        {moving && (
          <>
            {actor === 'admin' && (
              <label className="field">
                <span>Barber</span>
                <select
                  value={barberId}
                  onChange={(e) => {
                    setBarber(e.target.value)
                    setStart(null)
                    setError('')
                  }}
                >
                  {barberOptions.map((b) => (
                    <option value={b.id} key={b.id}>
                      {b.name}
                    </option>
                  ))}
                </select>
              </label>
            )}
            <DateStrip
              date={date}
              onChange={(d) => {
                setDate(d)
                setStart(null)
                setError('')
              }}
            />
            {availability.loading ? (
              <p role="status">Memeriksa jadwal…</p>
            ) : availability.error ? (
              <p className="notice notice-error" role="alert">
                {availability.error}
              </p>
            ) : (
              <SlotGrid
                key={date + barberId}
                times={times}
                start={start}
                onChange={(value) => {
                  setStart(value)
                  setError('')
                }}
              />
            )}
          </>
        )}
        {actor === 'admin' && (
          <label className="field">
            <span>Alasan {moving ? 'perubahan' : 'pembatalan'}</span>
            <textarea
              required
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={2}
              maxLength={200}
            />
          </label>
        )}
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button className="button button-outline" type="button" onClick={onClose}>
            Kembali
          </button>
          <button
            className="button button-primary"
            type="submit"
            disabled={pending || (moving && start === null)}
          >
            {pending ? 'Menyimpan…' : moving ? 'Simpan jadwal baru' : 'Ya, batalkan booking'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export function BookingTicket({
  studio,
  refresh,
  token,
  created,
}: StudioProps & { token: string; created: boolean }) {
  const [action, setAction] = useState<'move' | 'cancel' | null>(null)
  const [copy, setCopy] = useState('')
  const copyField = useRef<HTMLInputElement>(null)
  const [booking, setBooking] = useState<Booking | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const ticketRequest = useRef(0)
  const reload = useCallback(async () => {
    const request = ++ticketRequest.current
    try {
      const value = await api<Booking>('/bookings/current', { token })
      if (request !== ticketRequest.current) return
      setBooking(value)
      setError('')
    } catch (err) {
      if (request !== ticketRequest.current) return
      setError(messageOf(err))
      if (err instanceof APIError && [404, 410].includes(err.status)) setBooking(null)
    } finally {
      if (request === ticketRequest.current) setLoading(false)
    }
  }, [token])
  useEffect(() => {
    void reload()
    const visible = () => {
      if (!document.hidden) void reload()
    }
    const timer = window.setInterval(visible, 30_000)
    window.addEventListener('focus', reload)
    document.addEventListener('visibilitychange', visible)
    return () => {
      ticketRequest.current++
      window.clearInterval(timer)
      window.removeEventListener('focus', reload)
      document.removeEventListener('visibilitychange', visible)
    }
  }, [reload])
  const link = window.location.origin + window.location.pathname + '#booking/' + token

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(link)
      setCopy('Tautan disalin.')
    } catch {
      copyField.current?.focus()
      copyField.current?.select()
      setCopy('Salin tautan yang sudah dipilih.')
    }
  }

  return (
    <>
      <PublicHeader compact />
      <main id="konten" tabIndex={-1} className="ticket-page narrow-wrap">
        <BackLink />
        {booking && error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        {loading ? (
          <p className="notice" role="status">
            Membuka booking…
          </p>
        ) : booking ? (
          <>
            <div className="ticket-title">
              {booking.status === 'confirmed' && (
                <span className="confirmation-icon">
                  <Check size={26} aria-hidden="true" />
                </span>
              )}
              <div>
                <Kicker>
                  {booking.source === 'online' ? 'BOOKING ONLINE' : 'KUNJUNGAN DI TEMPAT'}
                </Kicker>
                <h1>
                  {created && booking.status === 'confirmed'
                    ? 'Jadwalmu sudah dicatat.'
                    : 'Detail booking.'}
                </h1>
                <p>
                  {booking.status === 'cancelled'
                    ? 'Kunjungan ini telah dibatalkan.'
                    : 'Simpan tautan privat agar bisa membuka jadwal ini kembali.'}
                </p>
              </div>
            </div>
            <div className="ticket-grid">
              <article className="ticket">
                <div className="ticket-main">
                  <div className="ticket-top">
                    <Kicker>TIKET KUNJUNGAN</Kicker>
                    <Status status={booking.status} />
                  </div>
                  <h2>{booking.serviceName}</h2>
                  <div className="ticket-date">
                    <strong>{longDate(booking.date)}</strong>
                    <span>
                      {time(booking.start)}–{time(booking.start + booking.duration)} WIB
                    </span>
                  </div>
                  <dl className="ticket-details">
                    <div>
                      <dt>Nama</dt>
                      <dd>{booking.customer}</dd>
                    </div>
                    <div>
                      <dt>Barber</dt>
                      <dd>
                        {booking.barberName ||
                          studio.barbers.find((b) => b.id === booking.barberId)?.name ||
                          'Barber'}
                      </dd>
                    </div>
                    <div>
                      <dt>Durasi</dt>
                      <dd>{booking.duration} menit</dd>
                    </div>
                    <div>
                      <dt>Bayar di tempat</dt>
                      <dd>{money(booking.price)}</dd>
                    </div>
                  </dl>
                </div>
                <div className="ticket-stub">
                  <span>KODE BOOKING</span>
                  <strong>{booking.code}</strong>
                  <small>{studio.profile.location}</small>
                </div>
              </article>
              <aside className="ticket-tools">
                <div className="ticket-link-heading">
                  <Link2 size={21} aria-hidden="true" />
                  <h2>Tautan privat</h2>
                </div>
                <p>
                  Tautan ini memberikan akses ke satu booking. Simpan secara pribadi dan buka dari
                  perangkat mana pun. Berlaku sampai 30 hari setelah jadwal terakhir.
                </p>
                <div className="copy-row">
                  <input ref={copyField} readOnly value={link} aria-label="Tautan privat booking" />
                  <button className="button button-outline" type="button" onClick={copyLink}>
                    <Copy size={16} aria-hidden="true" />
                    Salin
                  </button>
                </div>
                <span className="copy-result" role="status">
                  {copy}
                </span>
                <hr />
                <h3>Perlu ganti jam?</h3>
                <p>
                  {canManage(booking, 'customer')
                    ? 'Kamu bisa ubah jadwal atau batalkan paling lambat 2 jam sebelum kunjungan.'
                    : booking.status === 'confirmed'
                      ? 'Batas perubahan mandiri sudah lewat. Hubungi barbershop untuk bantuan.'
                      : 'Detail ini bisa kamu simpan sebagai referensi kunjungan.'}
                </p>
                {canManage(booking, 'customer') && (
                  <div className="ticket-actions">
                    <button
                      type="button"
                      className="button button-outline"
                      onClick={() => setAction('move')}
                    >
                      Ubah jadwal
                    </button>
                    <button
                      type="button"
                      className="text-link danger-text"
                      onClick={() => setAction('cancel')}
                    >
                      Batalkan booking
                    </button>
                  </div>
                )}
              </aside>
            </div>
            <BookingReview
              booking={booking}
              token={token}
              onUpdated={async () => {
                await reload()
                await refresh()
              }}
            />
            {action && (
              <ManageDialog
                studio={studio}
                refresh={refresh}
                booking={booking}
                actor="customer"
                token={token}
                action={action}
                onUpdated={() => void reload()}
                onClose={() => setAction(null)}
              />
            )}
          </>
        ) : (
          <div className="missing-ticket">
            <Kicker>TAUTAN BOOKING</Kicker>
            <h1>Tautan tidak tersedia.</h1>
            <p>{error || 'Buka tautan privat yang kamu simpan dari halaman konfirmasi.'}</p>
            <button type="button" className="button button-outline" onClick={() => void reload()}>
              Coba kembali
            </button>
            <BookingLink>Buat booking</BookingLink>
          </div>
        )}
      </main>
      <PublicFooter compact />
    </>
  )
}
