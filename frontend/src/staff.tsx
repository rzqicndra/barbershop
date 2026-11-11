import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import {
  ArrowRight,
  ArrowUpRight,
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  Clock3,
  LogOut,
  Plus,
  Settings2,
} from 'lucide-react'
import {
  addDays,
  canManage,
  longDate,
  money,
  parseTime,
  shortDate,
  statusActions,
  statuses,
  suitableBarbers,
  time,
  timeInput,
  today,
  validPhone,
  type Barber,
  type Booking,
  type BookingStatus,
  type Role,
  type Service,
  type User,
  type StudioProps,
} from './data'
import { BackLink, Brand, Empty, Kicker, Mark, Modal, Status } from './ui'
import { ManageDialog } from './booking'
import { PublicFooter } from './public'
import { api, APIError, availabilityPath, messageOf, setCSRF, useSlots } from './api'
import { StaffAccounts, TimeOffSettings } from './staff-settings'

export function StaffLogin({ signIn }: { signIn: (user: User) => void }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function submit(e: FormEvent) {
    e.preventDefault()
    if (pending) return
    setPending(true)
    setError('')
    try {
      const result = await api<{ user: User; csrfToken: string }>('/auth/login', {
        method: 'POST',
        body: { email, password },
      })
      setCSRF(result.csrfToken)
      setPassword('')
      signIn(result.user)
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <>
      <header className="staff-login-header">
        <div className="wrap masthead">
          <Brand />
          <Kicker>AREA STAF</Kicker>
        </div>
      </header>
      <main id="konten" tabIndex={-1} className="narrow-wrap staff-login">
        <BackLink />
        <div className="login-grid">
          <div className="login-cover">
            <Kicker>GILIR / AREA STAF</Kicker>
            <div>
              <h2>
                Jadwal
                <br />
                satu tempat.
              </h2>
              <p>Booking online dan walk-in berada di agenda yang sama.</p>
            </div>
            <Mark />
          </div>
          <form className="login-form" onSubmit={submit}>
            <Kicker>AKSES INTERNAL</Kicker>
            <h1>Masuk area staf.</h1>
            <p>Gunakan akun yang diberikan pemilik barbershop.</p>
            <label className="field">
              <span>Email staf</span>
              <input
                type="email"
                name="email"
                required
                value={email}
                autoComplete="username"
                onChange={(e) => setEmail(e.target.value)}
              />
            </label>
            <label className="field">
              <span>Kata sandi</span>
              <input
                type="password"
                name="password"
                required
                value={password}
                maxLength={128}
                autoComplete="current-password"
                onChange={(e) => setPassword(e.target.value)}
              />
            </label>
            {error && (
              <p className="notice notice-error" role="alert">
                {error}
              </p>
            )}
            <button type="submit" className="button button-primary" disabled={pending}>
              {pending ? 'Memeriksa akun…' : 'Masuk'}
              <ArrowRight size={17} />
            </button>
            <span className="muted">Hubungi pemilik barbershop jika akses akun bermasalah.</span>
          </form>
        </div>
      </main>
      <PublicFooter compact />
    </>
  )
}

export function StaffShell({
  user,
  route,
  signOut,
  children,
}: {
  user: User
  route: string
  signOut: () => void
  children: ReactNode
}) {
  const role = user.role
  return (
    <div className="staff-shell">
      <aside className="staff-sidebar">
        <div className="sidebar-brand">
          <Brand />
          <span>RUANG STAF</span>
        </div>
        <nav aria-label="Navigasi staf" className="staff-nav">
          <a
            href={role === 'admin' ? '/admin/agenda' : '/admin/jadwal'}
            aria-current={['admin/agenda', 'admin/jadwal'].includes(route) ? 'page' : undefined}
          >
            <CalendarDays size={18} />
            {role === 'admin' ? 'Agenda' : 'Jadwal saya'}
          </a>
          {role === 'admin' && (
            <a
              href="/admin/pengaturan"
              aria-current={route === 'admin/pengaturan' ? 'page' : undefined}
            >
              <Settings2 size={18} />
              Pengaturan
            </a>
          )}
          <a href="/admin/akun" aria-current={route === 'admin/akun' ? 'page' : undefined}>
            <Settings2 size={18} />
            Akun saya
          </a>
        </nav>
        <div className="staff-sidebar-bottom">
          <a href="/#beranda" className="text-link">
            Lihat website
            <ArrowUpRight size={15} />
          </a>
          <div className="staff-account">
            <span className="staff-avatar" aria-hidden="true">
              {user.name[0]}
            </span>
            <div>
              <strong>{user.name}</strong>
              <small>{role === 'admin' ? 'Admin' : 'Barber'}</small>
            </div>
            <button
              type="button"
              className="icon-button"
              aria-label="Keluar area staf"
              onClick={signOut}
            >
              <LogOut size={17} />
            </button>
          </div>
        </div>
      </aside>
      <main id="konten" tabIndex={-1} className="staff-content">
        {children}
      </main>
    </div>
  )
}

function WalkInDialog({
  studio,
  refresh,
  onClose,
  onBooked,
}: StudioProps & { onClose: () => void; onBooked: (date: string) => void }) {
  const active = studio.services.filter((s) => s.active)
  const [serviceId, setService] = useState(active[0]?.id ?? '')
  const [barberId, setBarber] = useState(suitableBarbers(studio, active[0]?.id ?? '')[0]?.id ?? '')
  const [start, setStart] = useState('')
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [error, setError] = useState('')
  const date = today()
  const service = active.find((s) => s.id === serviceId)
  const barbers = suitableBarbers(studio, serviceId)
  const availability = useSlots(
    service && barberId ? availabilityPath(service.id, barberId, date, 'walk-in') : null,
  )
  const times = availability.slots
  const [pending, setPending] = useState(false)
  const request = useRef({ fingerprint: '', key: '' })

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!service || pending) return
    if (name.trim().length < 2 || (phone && !validPhone(phone))) {
      setError('Periksa nama dan nomor WhatsApp. Nomor boleh dikosongkan.')
      return
    }
    const body = {
      serviceId,
      barberId,
      date,
      start: parseTime(start),
      customer: name.trim(),
      phone: phone.trim(),
      offerVersion: service.version,
    }
    const fingerprint = JSON.stringify(body)
    if (request.current.fingerprint !== fingerprint)
      request.current = { fingerprint, key: crypto.randomUUID() }
    setPending(true)
    setError('')
    try {
      await api<Booking>('/admin/bookings', {
        method: 'POST',
        body: { ...body, requestKey: request.current.key },
      })
      onBooked(date)
    } catch (err) {
      setError(messageOf(err))
      availability.reload()
      if (err instanceof APIError && err.code === 'OFFER_CHANGED') await refresh().catch(() => {})
    } finally {
      setPending(false)
    }
  }

  return (
    <Modal title="Catat walk-in" onClose={onClose}>
      <form className="dialog-form" onSubmit={submit}>
        <p className="dialog-intro">Kunjungan hari ini · {longDate(date)}</p>
        <div className="form-pair">
          <label className="field">
            <span>Layanan</span>
            <select
              value={serviceId}
              required
              onChange={(e) => {
                setService(e.target.value)
                setBarber(suitableBarbers(studio, e.target.value)[0]?.id ?? '')
                setStart('')
                setError('')
              }}
            >
              {active.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span>Barber</span>
            <select
              value={barberId}
              required
              onChange={(e) => {
                setBarber(e.target.value)
                setStart('')
                setError('')
              }}
            >
              {barbers.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>
          </label>
        </div>
        <label className="field">
          <span>Jam kunjungan (WIB)</span>
          <input
            type="time"
            value={start}
            required
            onInput={(e) => setStart(e.currentTarget.value)}
          />
        </label>
        {availability.error && (
          <p className="notice notice-error" role="alert">
            {availability.error}
          </p>
        )}
        {availability.loading ? (
          <p role="status">Memeriksa jadwal…</p>
        ) : times.length > 0 ? (
          <button
            type="button"
            className="text-link nearest-time"
            onClick={() => setStart(timeInput(times[0]))}
          >
            <Clock3 size={15} />
            Pakai jam terdekat · {time(times[0])}
          </button>
        ) : (
          <p className="notice">Belum ada jam tersisa untuk barber dan layanan ini hari ini.</p>
        )}
        <div className="form-pair">
          <label className="field">
            <span>Nama pelanggan</span>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoComplete="off"
              maxLength={80}
              required
            />
          </label>
          <label className="field">
            <span>WhatsApp (opsional)</span>
            <input
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              autoComplete="off"
              maxLength={24}
            />
          </label>
        </div>
        {service && (
          <div className="dialog-booking-note">
            <strong>{money(service.price)}</strong>
            <span>{service.duration} menit · bayar di tempat</span>
          </div>
        )}
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button type="button" className="button button-outline" onClick={onClose}>
            Kembali
          </button>
          <button
            type="submit"
            className="button button-primary"
            disabled={pending || !service || !barberId || !start}
          >
            {pending ? 'Menyimpan…' : 'Simpan walk-in'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

function DetailDialog({
  studio,
  booking,
  role,
  onClose,
  onManage,
  onUpdated,
}: StudioProps & {
  booking: Booking
  role: Role
  onClose: () => void
  onManage: (action: 'move' | 'cancel') => void
  onUpdated: () => void
}) {
  const [error, setError] = useState('')
  const actions = statusActions(booking)
  const barberName = studio.barbers.find((b) => b.id === booking.barberId)?.name ?? 'Barber'
  const actionName: Partial<Record<BookingStatus, string>> = {
    serving: 'Mulai layanan',
    completed: 'Tandai selesai',
    'no-show': 'Tandai tidak hadir',
  }
  const [pending, setPending] = useState(false)
  async function mark(status: BookingStatus) {
    if (pending) return
    setPending(true)
    setError('')
    try {
      await api<Booking>('/staff/bookings/' + booking.id + '/status', {
        method: 'POST',
        body: { status, version: booking.version },
      })
      onUpdated()
    } catch (err) {
      setError(messageOf(err))
      onUpdated()
    } finally {
      setPending(false)
    }
  }
  return (
    <Modal title="Detail kunjungan" onClose={onClose}>
      <div className="detail-body">
        <div className="detail-top">
          <Kicker>
            {booking.code} / {booking.source === 'online' ? 'ONLINE' : 'WALK-IN'}
          </Kicker>
          <Status status={booking.status} />
        </div>
        <h3>{booking.customer}</h3>
        <dl className="detail-data">
          <div>
            <dt>Jadwal</dt>
            <dd>
              {shortDate(booking.date)} · {time(booking.start)}–
              {time(booking.start + booking.duration)} WIB
            </dd>
          </div>
          <div>
            <dt>Layanan</dt>
            <dd>{booking.serviceName}</dd>
          </div>
          <div>
            <dt>Barber</dt>
            <dd>{barberName}</dd>
          </div>
          {role === 'admin' && (
            <div>
              <dt>WhatsApp</dt>
              <dd>{booking.phone || 'Belum dicatat'}</dd>
            </div>
          )}
          <div>
            <dt>Bayar di tempat</dt>
            <dd>{money(booking.price)}</dd>
          </div>
        </dl>
        {actions.length > 0 && (
          <div className="detail-actions">
            {actions.map((action, index) => (
              <button
                key={action}
                type="button"
                className={'button ' + (index === 0 ? 'button-primary' : 'button-outline')}
                disabled={pending}
                onClick={() => void mark(action)}
              >
                {actionName[action]}
              </button>
            ))}
          </div>
        )}
        {role === 'admin' && canManage(booking, 'admin') && (
          <div className="detail-actions">
            <button
              type="button"
              className="button button-outline"
              onClick={() => onManage('move')}
            >
              Ubah jadwal / barber
            </button>
            <button
              type="button"
              className="text-link danger-text"
              onClick={() => onManage('cancel')}
            >
              Batalkan booking
            </button>
          </div>
        )}
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        {role === 'admin' && (
          <details className="history">
            <summary>Riwayat perubahan</summary>
            <ol>
              {[...(booking.history ?? [])].reverse().map((event, index) => (
                <li key={index}>
                  <strong>{event.action}</strong>
                  <span>
                    {event.by} ·{' '}
                    {new Date(event.at).toLocaleString('id-ID', {
                      timeZone: 'Asia/Jakarta',
                      dateStyle: 'medium',
                      timeStyle: 'short',
                    })}
                  </span>
                  {event.detail && <p>{event.detail}</p>}
                </li>
              ))}
            </ol>
          </details>
        )}
      </div>
    </Modal>
  )
}

export function Agenda({
  studio,
  refresh,
  role,
  barberId,
}: StudioProps & { role: Role; barberId: string }) {
  const [date, setDate] = useState(today())
  const [filterBarber, setFilterBarber] = useState('')
  const [filterStatus, setFilterStatus] = useState('')
  const [filterSource, setFilterSource] = useState('')
  const [walkIn, setWalkIn] = useState(false)
  const [selected, setSelected] = useState('')
  const [manage, setManage] = useState<'move' | 'cancel' | null>(null)
  const [message, setMessage] = useState('')
  const [all, setAll] = useState<Booking[]>([])
  const [focused, setFocused] = useState<Booking | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [revision, setRevision] = useState(0)
  const loadDetail = useCallback(async (id: string) => {
    try {
      setFocused(await api<Booking>('/staff/bookings/' + id))
      setError('')
    } catch (err) {
      setFocused(null)
      setError(messageOf(err))
    }
  }, [])
  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    api<Booking[]>('/staff/bookings?' + new URLSearchParams({ date }), {
      signal: controller.signal,
    })
      .then((result) => {
        if (!controller.signal.aborted) {
          setAll(result)
          setError('')
          setLoading(false)
        }
      })
      .catch((err) => {
        if (!controller.signal.aborted) {
          setError(messageOf(err))
          setAll([])
          setLoading(false)
        }
      })
    return () => controller.abort()
  }, [date, revision, barberId])
  useEffect(() => {
    const reload = () => setRevision((value) => value + 1)
    const timer = window.setInterval(reload, 30_000)
    window.addEventListener('focus', reload)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('focus', reload)
    }
  }, [])
  const onUpdated = () => {
    setRevision((value) => value + 1)
    if (selected) void loadDetail(selected)
  }
  const filtered = all
    .filter(
      (b) =>
        (!filterBarber || filterBarber === b.barberId) &&
        (!filterStatus || filterStatus === b.status) &&
        (!filterSource || filterSource === b.source),
    )
    .sort((a, b) => a.start - b.start)
  const bars = [
    ['Total booking', all.length],
    ['Online', all.filter((b) => b.source === 'online').length],
    ['Walk-in', all.filter((b) => b.source === 'walk-in').length],
    ['Selesai', all.filter((b) => b.status === 'completed').length],
  ] as const
  return (
    <>
      <div className="staff-page-heading">
        <div>
          <Kicker>OPERASIONAL / {role === 'admin' ? 'AGENDA' : 'BARBER'}</Kicker>
          <h1>{role === 'admin' ? 'Agenda kunjungan' : 'Jadwal saya'}</h1>
          <p>
            {role === 'admin'
              ? 'Lihat booking online dan walk-in dalam satu agenda.'
              : 'Lihat kunjungan yang ditugaskan kepadamu dan catat status layanan.'}
          </p>
        </div>
        {role === 'admin' && (
          <button
            type="button"
            className="button button-primary"
            onClick={() => {
              setMessage('')
              setWalkIn(true)
            }}
          >
            <Plus size={18} />
            Catat walk-in
          </button>
        )}
      </div>
      <div className="agenda-datebar">
        <div className="agenda-date">
          <CalendarDays size={21} />
          <div>
            <small>TANGGAL AGENDA</small>
            <strong>{longDate(date)}</strong>
          </div>
        </div>
        <div className="day-controls">
          <button
            type="button"
            className="icon-button"
            aria-label="Hari sebelumnya"
            onClick={() => setDate(addDays(date, -1))}
          >
            <ChevronLeft size={20} />
          </button>
          <button type="button" className="button button-outline" onClick={() => setDate(today())}>
            Hari ini
          </button>
          <button
            type="button"
            className="icon-button"
            aria-label="Hari berikutnya"
            onClick={() => setDate(addDays(date, 1))}
          >
            <ChevronRight size={20} />
          </button>
          <input
            type="date"
            aria-label="Tanggal agenda"
            value={date}
            onInput={(e) => {
              if (e.currentTarget.value) setDate(e.currentTarget.value)
            }}
          />
        </div>
      </div>
      <div className="agenda-summary" aria-label="Ringkasan hari terpilih">
        {bars.map(([label, value]) => (
          <div key={label}>
            <strong>{value}</strong>
            <span>{label}</span>
          </div>
        ))}
      </div>
      {message && (
        <p className="notice notice-success" role="status">
          {message}
        </p>
      )}
      {error && (
        <p className="notice notice-error" role="alert">
          {error}
          <button
            type="button"
            className="text-link"
            onClick={() => setRevision((value) => value + 1)}
          >
            Coba kembali
          </button>
        </p>
      )}
      <section className="agenda-panel" aria-label="Daftar kunjungan">
        <div className="agenda-list-heading">
          <h2>Daftar kunjungan</h2>
          <span className="muted">{filtered.length} booking · WIB</span>
        </div>
        <div className="filter-row">
          {role === 'admin' && (
            <label className="filter-field">
              <span>Barber</span>
              <select value={filterBarber} onChange={(e) => setFilterBarber(e.target.value)}>
                <option value="">Semua barber</option>
                {studio.barbers.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </label>
          )}
          <label className="filter-field">
            <span>Status</span>
            <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)}>
              <option value="">Semua status</option>
              {Object.entries(statuses).map(([id, title]) => (
                <option key={id} value={id}>
                  {title}
                </option>
              ))}
            </select>
          </label>
          {role === 'admin' && (
            <label className="filter-field">
              <span>Sumber</span>
              <select value={filterSource} onChange={(e) => setFilterSource(e.target.value)}>
                <option value="">Semua sumber</option>
                <option value="online">Online</option>
                <option value="walk-in">Walk-in</option>
              </select>
            </label>
          )}
        </div>
        {loading ? (
          <p className="notice" role="status">
            Memuat agenda…
          </p>
        ) : filtered.length ? (
          <table className="agenda-table" aria-label="Booking pada tanggal terpilih">
            <thead>
              <tr>
                <th scope="col">JAM</th>
                <th scope="col">PELANGGAN</th>
                {role === 'admin' && (
                  <th scope="col" className="table-barber">
                    BARBER
                  </th>
                )}
                <th scope="col" className="table-source">
                  SUMBER
                </th>
                <th scope="col">STATUS</th>
                <th scope="col">
                  <span className="sr-only">Detail</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((b) => (
                <tr key={b.id}>
                  <td className="agenda-time">
                    <strong>{time(b.start)}</strong>
                    <small>{time(b.start + b.duration)}</small>
                  </td>
                  <td className="agenda-customer">
                    <strong>{b.customer}</strong>
                    <small>
                      {b.serviceName}
                      <span className="mobile-source">
                        {' '}
                        · {b.source === 'online' ? 'Online' : 'Walk-in'}
                        {role === 'admin'
                          ? ' · ' +
                            (studio.barbers.find((barber) => barber.id === b.barberId)?.name ?? '')
                          : ''}
                      </span>
                    </small>
                  </td>
                  {role === 'admin' && (
                    <td className="table-barber">
                      {studio.barbers.find((p) => p.id === b.barberId)?.name ?? 'Barber'}
                    </td>
                  )}
                  <td className="table-source">{b.source === 'online' ? 'Online' : 'Walk-in'}</td>
                  <td className="agenda-status">
                    <Status status={b.status} />
                  </td>
                  <td className="agenda-open">
                    <button
                      type="button"
                      className="icon-button"
                      aria-label={'Detail ' + b.customer}
                      onClick={() => {
                        setSelected(b.id)
                        setManage(null)
                        void loadDetail(b.id)
                      }}
                    >
                      <ArrowUpRight size={19} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <Empty title="Belum ada kunjungan">Pilih tanggal lain atau ubah filter yang aktif.</Empty>
        )}
      </section>
      {walkIn && role === 'admin' && (
        <WalkInDialog
          studio={studio}
          refresh={refresh}
          onClose={() => setWalkIn(false)}
          onBooked={(value) => {
            setDate(value)
            setWalkIn(false)
            setFilterSource('')
            setFilterStatus('')
            setFilterBarber('')
            setMessage('Walk-in masuk ke agenda hari ini.')
            setRevision((value) => value + 1)
          }}
        />
      )}
      {focused && !manage && (
        <DetailDialog
          studio={studio}
          refresh={refresh}
          booking={focused}
          role={role}
          onClose={() => {
            setSelected('')
            setFocused(null)
          }}
          onManage={setManage}
          onUpdated={onUpdated}
        />
      )}
      {focused && manage && role === 'admin' && (
        <ManageDialog
          studio={studio}
          refresh={refresh}
          booking={focused}
          actor="admin"
          action={manage}
          onUpdated={onUpdated}
          onClose={() => setManage(null)}
        />
      )}
    </>
  )
}

function ServiceEditor({
  service,
  onClose,
  onSave,
}: {
  service?: Service
  onClose: () => void
  onSave: (value: Service) => Promise<string>
}) {
  const [draft, setDraft] = useState<Service>(
    service ?? {
      id: crypto.randomUUID(),
      name: '',
      description: '',
      price: 0,
      duration: 45,
      active: true,
      version: 0,
    },
  )
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)
  async function submit(e: FormEvent) {
    e.preventDefault()
    if (pending) return
    if (!draft.name.trim()) {
      setError('Isi nama layanan.')
      return
    }
    setPending(true)
    setError(
      await onSave({ ...draft, name: draft.name.trim(), description: draft.description.trim() }),
    )
    setPending(false)
  }
  return (
    <Modal title={service ? 'Edit layanan' : 'Tambah layanan'} onClose={onClose}>
      <form className="dialog-form" onSubmit={submit}>
        <label className="field">
          <span>Nama layanan</span>
          <input
            required
            value={draft.name}
            maxLength={50}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </label>
        <label className="field">
          <span>Deskripsi</span>
          <textarea
            rows={2}
            value={draft.description}
            maxLength={150}
            onChange={(e) => setDraft({ ...draft, description: e.target.value })}
          />
        </label>
        <div className="form-pair">
          <label className="field">
            <span>Harga (Rp)</span>
            <input
              type="number"
              required
              min="0"
              max="10000000"
              value={draft.price}
              onChange={(e) => setDraft({ ...draft, price: Number(e.target.value) })}
            />
          </label>
          <label className="field">
            <span>Durasi (menit)</span>
            <input
              type="number"
              required
              min="15"
              max="180"
              value={draft.duration}
              onChange={(e) => setDraft({ ...draft, duration: Number(e.target.value) })}
            />
          </label>
        </div>
        <label className="check-field">
          <input
            type="checkbox"
            checked={draft.active}
            onChange={(e) => setDraft({ ...draft, active: e.target.checked })}
          />
          Tampilkan layanan di website
        </label>
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button type="button" className="button button-outline" onClick={onClose}>
            Kembali
          </button>
          <button type="submit" className="button button-primary" disabled={pending}>
            Simpan layanan
          </button>
        </div>
      </form>
    </Modal>
  )
}

function BarberEditor({
  barber,
  services,
  onClose,
  onSave,
}: {
  barber: Barber
  services: Service[]
  onClose: () => void
  onSave: (value: Barber) => Promise<string>
}) {
  const [draft, setDraft] = useState(barber)
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)
  const times = [
    ['start', 'Mulai kerja'],
    ['end', 'Selesai kerja'],
    ['breakStart', 'Mulai istirahat'],
    ['breakEnd', 'Selesai istirahat'],
  ] as const
  async function submit(e: FormEvent) {
    e.preventDefault()
    if (pending) return
    if (
      !draft.name.trim() ||
      draft.start >= draft.end ||
      draft.breakStart > draft.breakEnd ||
      (draft.breakStart !== draft.breakEnd &&
        (draft.breakStart < draft.start || draft.breakEnd > draft.end))
    ) {
      setError('Periksa nama, jam kerja, dan rentang istirahat.')
      return
    }
    setPending(true)
    setError(await onSave({ ...draft, name: draft.name.trim() }))
    setPending(false)
  }
  return (
    <Modal title={'Jadwal barber · ' + barber.name} onClose={onClose} wide>
      <form className="dialog-form" onSubmit={submit}>
        <div className="form-pair">
          <label className="field">
            <span>Nama</span>
            <input
              required
              maxLength={36}
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            />
          </label>
          <label className="field">
            <span>Keahlian</span>
            <input
              maxLength={75}
              value={draft.specialty}
              onChange={(e) => setDraft({ ...draft, specialty: e.target.value })}
            />
          </label>
        </div>
        <div className="form-pair">
          {times.map(([key, label]) => (
            <label className="field" key={key}>
              <span>{label} (WIB)</span>
              <input
                type="time"
                required
                value={timeInput(draft[key])}
                onInput={(e) => {
                  if (e.currentTarget.value)
                    setDraft({ ...draft, [key]: parseTime(e.currentTarget.value) })
                }}
              />
            </label>
          ))}
        </div>
        <fieldset>
          <legend>Hari libur rutin</legend>
          <div className="day-checkboxes">
            {['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'].map((day, i) => (
              <label className="check-field" key={day}>
                <input
                  type="checkbox"
                  checked={draft.daysOff.includes(i)}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      daysOff: e.target.checked
                        ? [...draft.daysOff, i]
                        : draft.daysOff.filter((d) => d !== i),
                    })
                  }
                />
                {day}
              </label>
            ))}
          </div>
        </fieldset>
        <fieldset>
          <legend>Layanan yang bisa dikerjakan</legend>
          <div className="check-list">
            {services.map((s) => (
              <label className="check-field" key={s.id}>
                <input
                  type="checkbox"
                  checked={draft.serviceIds.includes(s.id)}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      serviceIds: e.target.checked
                        ? [...draft.serviceIds, s.id]
                        : draft.serviceIds.filter((id) => id !== s.id),
                    })
                  }
                />
                {s.name}
              </label>
            ))}
          </div>
        </fieldset>
        <label className="check-field">
          <input
            type="checkbox"
            checked={draft.active}
            onChange={(e) => setDraft({ ...draft, active: e.target.checked })}
          />
          Aktif dan bisa dipesan
        </label>
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button type="button" className="button button-outline" onClick={onClose}>
            Kembali
          </button>
          <button type="submit" className="button button-primary" disabled={pending}>
            Simpan jadwal
          </button>
        </div>
      </form>
    </Modal>
  )
}

export function Settings({ studio, refresh }: StudioProps) {
  const [tab, setTab] = useState('profil')
  const [profile, setProfile] = useState(studio.profile)
  const [closure, setClosure] = useState('')
  const [serviceId, setService] = useState<string | null>(null)
  const [barberId, setBarber] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [pending, setPending] = useState(false)
  const barber =
    barberId === 'new'
      ? ({
          id: '',
          name: '',
          specialty: '',
          serviceIds: [],
          start: 600,
          end: 1260,
          breakStart: 840,
          breakEnd: 885,
          daysOff: [],
          active: true,
          version: 0,
        } as Barber)
      : studio.barbers.find((b) => b.id === barberId)
  useEffect(() => setProfile(studio.profile), [studio.profile.version])
  async function commit(path: string, body: unknown, method = 'PUT') {
    try {
      const catalog = await api<typeof studio>(path, { method, body })
      setProfile(catalog.profile)
      await refresh()
      setMessage('Pengaturan tersimpan.')
      setError('')
      return ''
    } catch (err) {
      return messageOf(err)
    }
  }
  async function profileSubmit(e: FormEvent) {
    e.preventDefault()
    if (pending) return
    setMessage('')
    setPending(true)
    setError(await commit('/admin/profile', profile))
    setPending(false)
  }

  return (
    <>
      <div className="staff-page-heading">
        <div>
          <Kicker>OPERASIONAL / ADMIN</Kicker>
          <h1>Pengaturan usaha</h1>
          <p>Kelola informasi, jam buka, layanan, dan barber.</p>
        </div>
      </div>
      <div className="settings-tabs" aria-label="Kategori pengaturan">
        {[
          ['profil', 'Profil & jam buka'],
          ['layanan', 'Layanan'],
          ['barber', 'Barber'],
          ['izin', 'Izin barber'],
          ['akun', 'Akun staf'],
        ].map(([key, label]) => (
          <button
            type="button"
            aria-pressed={tab === key}
            key={key}
            onClick={() => {
              setTab(key)
              setError('')
              setMessage('')
            }}
          >
            {label}
          </button>
        ))}
      </div>
      {message && (
        <p className="notice notice-success" role="status">
          {message}
        </p>
      )}
      {tab === 'profil' && (
        <form className="settings-panel" onSubmit={profileSubmit}>
          <h2>Informasi barbershop</h2>
          <div className="form-pair">
            <label className="field">
              <span>Nama usaha</span>
              <input
                required
                value={profile.name}
                maxLength={60}
                onChange={(e) => setProfile({ ...profile, name: e.target.value })}
              />
            </label>
            <label className="field">
              <span>Lokasi</span>
              <input
                required
                value={profile.location}
                maxLength={75}
                onChange={(e) => setProfile({ ...profile, location: e.target.value })}
              />
            </label>
          </div>
          <div className="form-pair">
            <label className="field">
              <span>Jam buka (WIB)</span>
              <input
                type="time"
                required
                value={timeInput(profile.open)}
                onInput={(e) => {
                  if (e.currentTarget.value)
                    setProfile({ ...profile, open: parseTime(e.currentTarget.value) })
                }}
              />
            </label>
            <label className="field">
              <span>Jam tutup (WIB)</span>
              <input
                type="time"
                required
                value={timeInput(profile.close)}
                onInput={(e) => {
                  if (e.currentTarget.value)
                    setProfile({ ...profile, close: parseTime(e.currentTarget.value) })
                }}
              />
            </label>
          </div>
          <h3>Tanggal tutup</h3>
          <div className="closure-row">
            <input
              type="date"
              aria-label="Tambah tanggal tutup"
              min={today()}
              value={closure}
              onInput={(e) => setClosure(e.currentTarget.value)}
            />
            <button
              type="button"
              className="button button-outline"
              disabled={!closure || profile.closedDates.includes(closure)}
              onClick={() => {
                setProfile({ ...profile, closedDates: [...profile.closedDates, closure] })
                setClosure('')
              }}
            >
              Tambahkan
            </button>
          </div>
          {profile.closedDates.length > 0 && (
            <ul className="closures">
              {profile.closedDates.map((date) => (
                <li key={date}>
                  <span>{shortDate(date)}</span>
                  <button
                    type="button"
                    className="text-link"
                    onClick={() =>
                      setProfile({
                        ...profile,
                        closedDates: profile.closedDates.filter((item) => item !== date),
                      })
                    }
                  >
                    Hapus
                  </button>
                </li>
              ))}
            </ul>
          )}
          {error && (
            <p className="notice notice-error" role="alert">
              {error}
            </p>
          )}
          <button type="submit" className="button button-primary" disabled={pending}>
            Simpan pengaturan
            <ArrowRight size={17} />
          </button>
        </form>
      )}
      {tab === 'layanan' && (
        <section className="settings-panel">
          <div className="settings-section-head">
            <div>
              <h2>Katalog layanan</h2>
              <p>Harga baru berlaku untuk booking berikutnya.</p>
            </div>
            <button
              type="button"
              className="button button-primary"
              onClick={() => {
                setMessage('')
                setService('new')
              }}
            >
              <Plus size={17} />
              Tambah layanan
            </button>
          </div>
          <div className="settings-list">
            {studio.services.map((service) => (
              <div key={service.id}>
                <span>
                  <strong>{service.name}</strong>
                  <small>
                    {money(service.price)} · {service.duration} menit ·{' '}
                    {service.active ? 'Aktif' : 'Nonaktif'}
                  </small>
                </span>
                <button
                  type="button"
                  className="button button-outline"
                  aria-label={'Edit ' + service.name}
                  onClick={() => {
                    setMessage('')
                    setService(service.id)
                  }}
                >
                  Edit
                </button>
              </div>
            ))}
          </div>
        </section>
      )}
      {tab === 'barber' && (
        <section className="settings-panel">
          <div className="settings-section-head">
            <div>
              <h2>Tim barber</h2>
              <p>Atur jadwal, istirahat, dan layanan per barber.</p>
            </div>
            <button
              className="button button-primary"
              type="button"
              onClick={() => setBarber('new')}
            >
              Tambah barber
            </button>
          </div>
          <div className="settings-list">
            {studio.barbers.map((b) => (
              <div key={b.id}>
                <span>
                  <strong>{b.name}</strong>
                  <small>
                    {time(b.start)}–{time(b.end)} WIB · {b.active ? 'Aktif' : 'Nonaktif'}
                  </small>
                </span>
                <button
                  type="button"
                  className="button button-outline"
                  aria-label={'Atur ' + b.name}
                  onClick={() => {
                    setMessage('')
                    setBarber(b.id)
                  }}
                >
                  Atur jadwal
                </button>
              </div>
            ))}
          </div>
        </section>
      )}
      {tab === 'izin' && <TimeOffSettings studio={studio} refresh={refresh} />}
      {tab === 'akun' && <StaffAccounts studio={studio} />}
      {serviceId !== null && (
        <ServiceEditor
          service={studio.services.find((s) => s.id === serviceId)}
          onClose={() => setService(null)}
          onSave={async (service) => {
            const result = await commit(
              serviceId === 'new' ? '/admin/services' : '/admin/services/' + service.id,
              service,
              serviceId === 'new' ? 'POST' : 'PUT',
            )
            if (!result) setService(null)
            return result
          }}
        />
      )}
      {barber && (
        <BarberEditor
          barber={barber}
          services={studio.services}
          onClose={() => setBarber(null)}
          onSave={async (updated) => {
            const result = await commit(
              barberId === 'new' ? '/admin/barbers' : '/admin/barbers/' + updated.id,
              updated,
              barberId === 'new' ? 'POST' : 'PUT',
            )
            if (!result) setBarber(null)
            return result
          }}
        />
      )}
    </>
  )
}
