import { useEffect, useState } from 'react'
import { Clock3 } from 'lucide-react'
import { serverNow } from './clock'
import { longDate, statusActions, time, ZONE, type Booking, type BookingStatus } from './data'

const serverTimeFormat = new Intl.DateTimeFormat('id-ID', {
  timeZone: ZONE,
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hourCycle: 'h23',
})

export function ServiceActions({
  booking,
  pending,
  onChange,
}: {
  booking: Booking
  pending: boolean
  onChange: (status: BookingStatus) => Promise<void>
}) {
  const [now, setNow] = useState(serverNow)
  useEffect(() => {
    const tick = () => setNow(serverNow())
    const timer = window.setInterval(tick, 1000)
    window.addEventListener('focus', tick)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('focus', tick)
    }
  }, [])
  const actions = statusActions(booking, now)
  const waiting = booking.status === 'confirmed' && !actions.includes('serving')
  if (!actions.length && booking.status !== 'confirmed') return null
  return (
    <section className="service-actions" aria-label="Pelayanan kunjungan">
      {waiting && (
        <div className="service-start-help" id="service-start-rule">
          <Clock3 size={18} aria-hidden="true" />
          <div>
            <strong>Menunggu waktu kunjungan</strong>
            <p>
              Mulai layanan tersedia pada {longDate(booking.date)}, pukul {time(booking.start)} WIB.
            </p>
            <small>
              Waktu server saat ini{' '}
              <time dateTime={new Date(now).toISOString()}>{serverTimeFormat.format(now)} WIB</time>
              .
            </small>
          </div>
        </div>
      )}
      <div className="detail-actions">
        {booking.status === 'confirmed' && (
          <button
            type="button"
            className="button button-primary"
            disabled={pending || waiting}
            aria-describedby={waiting ? 'service-start-rule' : undefined}
            onClick={() => void onChange('serving')}
          >
            Mulai layanan
          </button>
        )}
        {actions.includes('completed') && (
          <button
            type="button"
            className="button button-primary"
            disabled={pending}
            onClick={() => void onChange('completed')}
          >
            Tandai selesai
          </button>
        )}
        {actions.includes('no-show') && (
          <button
            type="button"
            className="button button-outline"
            disabled={pending}
            onClick={() => void onChange('no-show')}
          >
            Tandai tidak hadir
          </button>
        )}
      </div>
    </section>
  )
}
