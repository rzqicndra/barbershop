import {
  createContext,
  useContext,
  useEffect,
  useId,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { ArrowLeft, ArrowRight, X } from 'lucide-react'
import { statuses, type BookingStatus, type Studio } from './data'
import { api, messageOf } from './api'

export function Mark({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      aria-hidden="true"
      width="36"
      height="36"
      viewBox="0 0 40 40"
      fill="currentColor"
    >
      <path d="M8 24 17 9h5l-9 15zM14 30 26 9h5L19 30zM23 30l6-10h5l-6 10z" />
    </svg>
  )
}

export const BrandContext = createContext<Studio['profile'] | null>(null)

export function Brand({
  inverse = false,
  href = '/#beranda',
  label,
}: {
  inverse?: boolean
  href?: string
  label?: string
}) {
  const profile = useContext(BrandContext)
  const name = profile?.name || 'Gilir Barbershop'
  const wordmark = profile?.wordmark || name.replace(/\s*barbershop$/i, '').toLowerCase()
  return (
    <a
      className={inverse ? 'brand brand-inverse' : 'brand'}
      href={href}
      aria-label={label || name + ', beranda'}
    >
      {profile?.logoImageUrl ? (
        <img className="brand-image" src={profile.logoImageUrl} alt={name} />
      ) : (
        <span className="brand-mark">
          <Mark />
        </span>
      )}
      {!profile?.logoImageUrl && (
        <span className="brand-type">
          {wordmark}
          <span>.</span>
          <small>BARBERSHOP</small>
        </span>
      )}
    </a>
  )
}

export function Kicker({ children }: { children: ReactNode }) {
  return <span className="kicker">{children}</span>
}

export function BookingLink({
  children = 'Booking sekarang',
  service,
}: {
  children?: ReactNode
  service?: string
}) {
  return (
    <a
      href={'/#booking' + (service ? '?layanan=' + encodeURIComponent(service) : '')}
      className="button button-primary"
    >
      {children}
      <ArrowRight size={17} aria-hidden="true" />
    </a>
  )
}

export function BackLink({
  href = '/#beranda',
  children = 'Kembali ke beranda',
}: {
  href?: string
  children?: ReactNode
}) {
  return (
    <a href={href} className="text-link back-link">
      <ArrowLeft size={16} aria-hidden="true" />
      {children}
    </a>
  )
}

export function Status({ status }: { status: BookingStatus }) {
  return (
    <span className={'status status-' + status}>
      <i aria-hidden="true" />
      {statuses[status]}
    </span>
  )
}

export function Modal({
  title,
  children,
  onClose,
  wide = false,
}: {
  title: string
  children: ReactNode
  onClose: () => void
  wide?: boolean
}) {
  const dialog = useRef<HTMLDialogElement>(null)
  const titleId = useId()
  useEffect(() => {
    const element = dialog.current
    element?.showModal()
    return () => element?.close()
  }, [])
  return (
    <dialog
      ref={dialog}
      className={'modal ' + (wide ? 'modal-wide' : '')}
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault()
        onClose()
      }}
      onClick={(e) => {
        if (e.target === dialog.current) onClose()
      }}
    >
      <div className="modal-body">
        <header className="modal-head">
          <h2 id={titleId}>{title}</h2>
          <button className="icon-button" type="button" aria-label="Tutup" onClick={onClose}>
            <X size={20} />
          </button>
        </header>
        {children}
      </div>
    </dialog>
  )
}

export function Empty({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="empty">
      <strong>{title}</strong>
      <p>{children}</p>
    </div>
  )
}

export function DeleteDialog({
  title,
  description,
  path,
  version,
  onClose,
  onDeleted,
}: {
  title: string
  description: string
  path: string
  version?: number
  onClose: () => void
  onDeleted: () => void | Promise<void>
}) {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function remove() {
    if (pending) return
    setPending(true)
    try {
      await api(path, { method: 'DELETE', body: version === undefined ? undefined : { version } })
      await onDeleted()
      onClose()
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <Modal
      title={title}
      onClose={() => {
        if (!pending) onClose()
      }}
    >
      <div className="dialog-form">
        <p>{description}</p>
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button
            type="button"
            className="button button-outline"
            disabled={pending}
            onClick={onClose}
          >
            Kembali
          </button>
          <button
            type="button"
            className="button button-danger"
            disabled={pending}
            onClick={() => void remove()}
          >
            {pending ? 'Menghapus…' : 'Ya, hapus'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
