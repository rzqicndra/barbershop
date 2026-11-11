import { useState, type FormEvent } from 'react'
import { Check, Star } from 'lucide-react'
import { api, APIError, messageOf } from './api'
import { shortDate, today, type Booking, type Review, type Studio } from './data'

export function Stars({ rating, label }: { rating: number; label?: string }) {
  return (
    <span className="stars" role="img" aria-label={label || rating + ' dari 5 bintang'}>
      {Array.from({ length: 5 }, (_, index) => (
        <Star
          key={index}
          size={16}
          fill={index < Math.round(rating) ? 'currentColor' : 'none'}
          aria-hidden="true"
        />
      ))}
    </span>
  )
}

export function BookingReview({
  booking,
  token,
  onUpdated,
}: {
  booking: Booking
  token: string
  onUpdated: () => Promise<void>
}) {
  const [rating, setRating] = useState(0)
  const [name, setName] = useState(booking.customer.trim().split(/\s+/)[0] || '')
  const [comment, setComment] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  if (booking.review)
    return (
      <section className="customer-review-panel" aria-label="Ulasan kunjungan">
        <div className="review-saved">
          <Check size={20} />
          <h2>Terima kasih untuk ulasanmu.</h2>
        </div>
        <Stars rating={booking.review.rating} />
        <p>{booking.review.comment || 'Penilaianmu sudah tercatat.'}</p>
        <small>
          Ditampilkan sebagai {booking.review.displayName} · satu ulasan untuk kunjungan ini.
        </small>
      </section>
    )
  if (booking.status !== 'completed')
    return (
      <section className="customer-review-panel review-waiting" aria-label="Ulasan kunjungan">
        <h2>Bagaimana hasil kunjunganmu?</h2>
        <p>
          Rating dan ulasan dapat diberikan setelah status kunjungan menjadi{' '}
          <strong>Selesai</strong>.
        </p>
        <button className="button button-outline" type="button" disabled>
          Beri ulasan setelah selesai
        </button>
      </section>
    )
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending) return
    if (rating < 1) {
      setError('Pilih jumlah bintang untuk kunjungan ini.')
      return
    }
    setPending(true)
    setError('')
    try {
      await api<Review>('/bookings/current/review', {
        method: 'POST',
        token,
        body: { rating, displayName: name.trim(), comment: comment.trim() },
      })
      await onUpdated()
    } catch (err) {
      setError(messageOf(err))
      if (err instanceof APIError && err.code === 'REVIEW_EXISTS') await onUpdated()
    } finally {
      setPending(false)
    }
  }
  return (
    <section className="customer-review-panel" aria-label="Beri ulasan kunjungan">
      <h2>Bagaimana hasil kunjunganmu?</h2>
      <p>
        Bagikan pengalamanmu. Nama tampilan, rating, dan ulasanmu akan tampil di halaman publik.
      </p>
      <form className="review-form" onSubmit={submit}>
        <fieldset className="rating-field">
          <legend>Penilaian</legend>
          <div className="rating-options">
            {[1, 2, 3, 4, 5].map((value) => (
              <label key={value} className={value <= rating ? 'selected' : ''}>
                <input
                  type="radio"
                  name="rating"
                  value={value}
                  checked={rating === value}
                  onChange={() => setRating(value)}
                  required
                  aria-label={value + ' bintang'}
                />
                <Star
                  size={30}
                  fill={value <= rating ? 'currentColor' : 'none'}
                  aria-hidden="true"
                />
              </label>
            ))}
          </div>
          <span className="muted">{rating ? rating + ' dari 5 bintang' : 'Pilih 1–5 bintang'}</span>
        </fieldset>
        <label className="field">
          <span>Nama tampilan</span>
          <input
            required
            minLength={2}
            maxLength={40}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <small>Nama depan atau nama panggilan yang boleh ditampilkan di halaman publik.</small>
        </label>
        <label className="field">
          <span>Ulasan (opsional)</span>
          <textarea
            rows={3}
            maxLength={1000}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            placeholder="Ceritakan pengalaman potong atau perawatanmu."
          />
        </label>
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <div>
          <button className="button button-primary" type="submit" disabled={pending}>
            {pending ? 'Menyimpan…' : 'Kirim ulasan'}
          </button>
        </div>
      </form>
    </section>
  )
}

export function PublicReviews({ studio }: { studio: Studio }) {
  const [more, setMore] = useState<Review[]>([])
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const initial = studio.reviews || []
  const reviews = [
    ...initial,
    ...more.filter((item) => !initial.some((first) => first.id === item.id)),
  ]
  const summary = studio.rating || { average: 0, count: 0 }
  async function loadMore() {
    setPending(true)
    try {
      const values = await api<Review[]>('/reviews?offset=' + reviews.length)
      setMore((current) => [
        ...current,
        ...values.filter((value) => !current.some((item) => item.id === value.id)),
      ])
      setError('')
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <div className="public-reviews">
      <div className="rating-summary">
        <strong>
          {summary.count
            ? summary.average.toLocaleString('id-ID', {
                minimumFractionDigits: 1,
                maximumFractionDigits: 1,
              })
            : '—'}
        </strong>
        <div>
          <Stars
            rating={summary.average}
            label={
              summary.count ? summary.average.toFixed(1) + ' dari 5 bintang' : 'Belum ada penilaian'
            }
          />
          <p>
            {summary.count
              ? summary.count + ' penilaian dari kunjungan selesai'
              : 'Belum ada penilaian. Jadilah yang pertama setelah berkunjung.'}
          </p>
        </div>
      </div>
      <div className="review-list">
        {reviews.map((review) => (
          <article className="review-card" key={review.id}>
            <header>
              <strong>{review.displayName}</strong>
              <Stars rating={review.rating} />
            </header>
            <small>
              {review.serviceName} · {review.barberName} ·{' '}
              {shortDate(today(Date.parse(review.createdAt)))}
            </small>
            {review.comment && <p>{review.comment}</p>}
            <span className="verified-visit">
              <Check size={13} />
              Kunjungan selesai terverifikasi
            </span>
          </article>
        ))}
      </div>
      {error && (
        <p className="notice notice-error" role="alert">
          {error}
        </p>
      )}
      {reviews.length < summary.count && (
        <button
          className="button button-outline"
          type="button"
          disabled={pending}
          onClick={() => void loadMore()}
        >
          {pending ? 'Memuat…' : 'Lihat ulasan lainnya'}
        </button>
      )}
    </div>
  )
}
