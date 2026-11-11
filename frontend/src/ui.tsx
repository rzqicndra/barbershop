import { useEffect, useRef, type ReactNode } from 'react'
import { ArrowLeft, ArrowRight, X } from 'lucide-react'
import { statuses, type BookingStatus } from './data'

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

export function Brand({ inverse = false }: { inverse?: boolean }) {
  return (
    <a
      className={inverse ? 'brand brand-inverse' : 'brand'}
      href="/#beranda"
      aria-label="Gilir Barbershop, beranda"
    >
      <span className="brand-mark">
        <Mark />
      </span>
      <span className="brand-type">
        gilir<span>.</span>
        <small>BARBERSHOP</small>
      </span>
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
  useEffect(() => {
    const element = dialog.current
    element?.showModal()
    return () => element?.close()
  }, [])
  return (
    <dialog
      ref={dialog}
      className={'modal ' + (wide ? 'modal-wide' : '')}
      aria-labelledby="modal-title"
      onCancel={onClose}
      onClick={(e) => {
        if (e.target === dialog.current) onClose()
      }}
    >
      <div className="modal-body">
        <header className="modal-head">
          <h2 id="modal-title">{title}</h2>
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
