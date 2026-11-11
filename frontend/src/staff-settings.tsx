import { useEffect, useState, type FormEvent } from 'react'
import { api, messageOf } from './api'
import { longDate, parseTime, time, today, type Studio, type StudioProps, type User } from './data'
import { DeleteDialog, Kicker, Modal } from './ui'

export function AccountPage({ user, onChanged }: { user: User; onChanged: () => void }) {
  const [currentPassword, setCurrent] = useState('')
  const [newPassword, setNew] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending) return
    if (newPassword !== confirmation) {
      setError('Konfirmasi kata sandi belum cocok.')
      return
    }
    setPending(true)
    try {
      await api('/auth/password', { method: 'POST', body: { currentPassword, newPassword } })
      onChanged()
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <>
      <div className="staff-page-heading">
        <div>
          <Kicker>AKSES / AKUN SAYA</Kicker>
          <h1>Akun saya</h1>
          <p>
            {user.name} · {user.email}
          </p>
        </div>
      </div>
      <form className="settings-panel account-panel" onSubmit={submit}>
        <h2>Ganti kata sandi</h2>
        <p>
          Setelah berhasil, masuk kembali dengan kata sandi baru. Sesi lain milik akun ini juga akan
          berakhir.
        </p>
        <label className="field">
          <span>Kata sandi saat ini</span>
          <input
            type="password"
            autoComplete="current-password"
            required
            maxLength={128}
            value={currentPassword}
            onChange={(e) => setCurrent(e.target.value)}
          />
        </label>
        <div className="form-pair">
          <label className="field">
            <span>Kata sandi baru</span>
            <input
              type="password"
              autoComplete="new-password"
              required
              minLength={12}
              maxLength={128}
              value={newPassword}
              onChange={(e) => setNew(e.target.value)}
            />
          </label>
          <label className="field">
            <span>Ulangi kata sandi baru</span>
            <input
              type="password"
              autoComplete="new-password"
              required
              minLength={12}
              maxLength={128}
              value={confirmation}
              onChange={(e) => setConfirmation(e.target.value)}
            />
          </label>
        </div>
        <small>Gunakan 12–128 karakter.</small>
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <button type="submit" className="button button-primary" disabled={pending}>
          {pending ? 'Menyimpan…' : 'Ganti kata sandi'}
        </button>
      </form>
    </>
  )
}
export function TimeOffSettings({ studio, refresh }: StudioProps) {
  const [barberId, setBarber] = useState(studio.barbers[0]?.id ?? '')
  const [date, setDate] = useState(today())
  const [start, setStart] = useState('10:00')
  const [end, setEnd] = useState('21:00')
  const [reason, setReason] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending) return
    setPending(true)
    setError('')
    try {
      await api('/admin/time-off', {
        method: 'POST',
        body: { barberId, date, start: parseTime(start), end: parseTime(end), reason },
      })
      await refresh()
      setReason('')
      setMessage('Izin tersimpan. Jam tersebut tidak ditawarkan untuk booking baru.')
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  async function remove(id: string) {
    if (pending) return
    setPending(true)
    try {
      await api('/admin/time-off/' + id, { method: 'DELETE' })
      await refresh()
      setMessage('Izin dihapus.')
      setError('')
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <section className="settings-panel">
      <h2>Izin pada tanggal tertentu</h2>
      <p>
        Gunakan untuk cuti, izin sebagian hari, atau kebutuhan mendadak. Jika ada booking yang
        terdampak, tangani di agenda terlebih dahulu.
      </p>
      <form className="dialog-form" onSubmit={submit}>
        <div className="form-pair">
          <label className="field">
            <span>Barber</span>
            <select required value={barberId} onChange={(e) => setBarber(e.target.value)}>
              {studio.barbers.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span>Tanggal</span>
            <input
              type="date"
              required
              min={today()}
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </label>
        </div>
        <div className="form-pair">
          <label className="field">
            <span>Mulai (WIB)</span>
            <input type="time" required value={start} onChange={(e) => setStart(e.target.value)} />
          </label>
          <label className="field">
            <span>Selesai (WIB)</span>
            <input type="time" required value={end} onChange={(e) => setEnd(e.target.value)} />
          </label>
        </div>
        <label className="field">
          <span>Catatan internal</span>
          <input maxLength={200} value={reason} onChange={(e) => setReason(e.target.value)} />
        </label>
        <div>
          <button className="button button-primary" type="submit" disabled={pending || !barberId}>
            Simpan izin
          </button>
        </div>
      </form>
      {error && (
        <p className="notice notice-error" role="alert">
          {error}
        </p>
      )}
      {message && (
        <p className="notice notice-success" role="status">
          {message}
        </p>
      )}
      <div className="settings-list">
        {studio.timeOff
          .filter((block) => block.date >= today())
          .map((block) => (
            <div key={block.id}>
              <span>
                <strong>
                  {studio.barbers.find((b) => b.id === block.barberId)?.name} ·{' '}
                  {longDate(block.date)}
                </strong>
                <small>
                  {time(block.start)}–{time(block.end)} WIB
                  {block.reason ? ' · ' + block.reason : ''}
                </small>
              </span>
              <button
                className="button button-outline"
                type="button"
                disabled={pending}
                onClick={() => void remove(block.id)}
              >
                Hapus izin
              </button>
            </div>
          ))}
      </div>
    </section>
  )
}
export function StaffAccounts({ studio, userId }: { studio: Studio; userId: string }) {
  const [staff, setStaff] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [editor, setEditor] = useState<User | 'new' | null>(null)
  const [deleting, setDeleting] = useState<User | null>(null)
  const [pending, setPending] = useState(false)
  useEffect(() => {
    const controller = new AbortController()
    api<User[]>('/admin/staff', { signal: controller.signal })
      .then(setStaff)
      .catch((err) => {
        if (!controller.signal.aborted) setError(messageOf(err))
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
  }, [])
  async function toggle(user: User) {
    setPending(true)
    setError('')
    try {
      setStaff(
        await api<User[]>('/admin/staff/' + user.id, {
          method: 'PUT',
          body: { active: !user.active },
        }),
      )
      setMessage(user.active ? 'Akun dinonaktifkan dan sesinya diakhiri.' : 'Akun diaktifkan.')
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <section className="settings-panel">
      <div className="settings-section-head">
        <div>
          <h2>Akun staf</h2>
          <p>Setiap akun barber terhubung ke satu profil barber.</p>
        </div>
        <button className="button button-primary" type="button" onClick={() => setEditor('new')}>
          Tambah akun barber
        </button>
      </div>
      {loading && <p role="status">Memuat akun…</p>}
      {error && (
        <p className="notice notice-error" role="alert">
          {error}
        </p>
      )}
      {message && (
        <p className="notice notice-success" role="status">
          {message}
        </p>
      )}
      <div className="settings-list">
        {staff.map((user) => (
          <div key={user.id}>
            <span>
              <strong>
                {user.name} · {user.role === 'admin' ? 'Admin' : 'Barber'}
              </strong>
              <small>
                {user.email} · {user.active ? 'Aktif' : 'Nonaktif'}
              </small>
            </span>
            <div className="account-actions">
              {user.role === 'barber' && (
                <>
                  <button
                    type="button"
                    className="button button-outline"
                    disabled={pending}
                    onClick={() => setEditor(user)}
                  >
                    Reset sandi
                  </button>
                  <button
                    type="button"
                    className="text-link"
                    disabled={pending}
                    onClick={() => void toggle(user)}
                  >
                    {user.active ? 'Nonaktifkan' : 'Aktifkan'}
                  </button>
                </>
              )}
              <button
                type="button"
                className="text-link danger-text"
                disabled={pending || user.id === userId}
                title={
                  user.id === userId ? 'Akun yang sedang dipakai tidak dapat dihapus' : undefined
                }
                onClick={() => setDeleting(user)}
              >
                {user.id === userId ? 'Akun ini' : 'Hapus akun'}
              </button>
            </div>
          </div>
        ))}
      </div>
      {deleting && (
        <DeleteDialog
          title={'Hapus akun ' + deleting.name + '?'}
          description="Akun akan dihapus dari daftar staf dan seluruh sesinya diakhiri. Profil barber dan riwayat booking tetap tersimpan."
          path={'/admin/staff/' + deleting.id}
          onClose={() => setDeleting(null)}
          onDeleted={async () => {
            setStaff(await api<User[]>('/admin/staff'))
            setMessage('Akun berhasil dihapus.')
          }}
        />
      )}
      {editor && (
        <StaffAccountEditor
          user={editor === 'new' ? undefined : editor}
          availableBarbers={studio.barbers.filter((b) => !staff.some((u) => u.barberId === b.id))}
          onClose={() => setEditor(null)}
          onSaved={(values) => {
            if (values) setStaff(values)
            setEditor(null)
            setMessage('Akses akun berhasil disimpan.')
          }}
        />
      )}
    </section>
  )
}
function StaffAccountEditor({
  user,
  availableBarbers,
  onClose,
  onSaved,
}: {
  user?: User
  availableBarbers: Studio['barbers']
  onClose: () => void
  onSaved: (values?: User[]) => void
}) {
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [barberId, setBarber] = useState(availableBarbers[0]?.id ?? '')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending) return
    setPending(true)
    try {
      if (user) {
        await api('/admin/staff/' + user.id + '/password', { method: 'POST', body: { password } })
        onSaved()
      } else {
        onSaved(
          await api<User[]>('/admin/staff', {
            method: 'POST',
            body: { email, name, barberId, password },
          }),
        )
      }
      setPassword('')
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <Modal title={user ? 'Reset sandi · ' + user.name : 'Tambah akun barber'} onClose={onClose}>
      <form className="dialog-form" onSubmit={submit}>
        {!user && (
          <>
            <label className="field">
              <span>Profil barber</span>
              <select required value={barberId} onChange={(e) => setBarber(e.target.value)}>
                {availableBarbers.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              <span>Nama akun</span>
              <input
                required
                minLength={2}
                maxLength={80}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            <label className="field">
              <span>Email</span>
              <input
                type="email"
                required
                autoComplete="off"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </label>
          </>
        )}
        <label className="field">
          <span>Kata sandi {user ? 'baru' : 'awal'}</span>
          <input
            type="password"
            required
            autoComplete="new-password"
            minLength={12}
            maxLength={128}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <small>12–128 karakter. Sampaikan langsung kepada barber yang bersangkutan.</small>
        </label>
        {user && <p className="notice">Sesi akun ini akan diakhiri setelah kata sandi direset.</p>}
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
            disabled={pending || (!user && !barberId)}
          >
            {pending ? 'Menyimpan…' : 'Simpan akses'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
