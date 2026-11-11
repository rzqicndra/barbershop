import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ServiceActions } from './service-actions'
import type { Booking } from './data'

const booking: Booking = {
  id: 'booking',
  code: 'GL-TEST',
  date: '2026-11-11',
  start: 750,
  serviceId: 'service',
  serviceName: 'Potong rambut',
  duration: 45,
  price: 65000,
  barberId: 'raka',
  customer: 'Pelanggan uji',
  source: 'online',
  status: 'confirmed',
  version: 1,
}

describe('service actions and scheduled start', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())
  it('explains the disabled action before the appointment instead of hiding it', () => {
    vi.setSystemTime(new Date('2026-11-11T11:47:00+07:00'))
    const onChange = vi.fn(async () => {})
    render(<ServiceActions booking={booking} pending={false} onChange={onChange} />)
    const button = screen.getByRole('button', { name: 'Mulai layanan' })
    expect(button).toBeDisabled()
    expect(screen.getByText(/pukul 12.30 WIB/)).toBeInTheDocument()
    expect(screen.getByText('11.47.00 WIB')).toBeInTheDocument()
    fireEvent.click(button)
    expect(onChange).not.toHaveBeenCalled()
  })
  it('enables the start button at the scheduled server time without reopening the dialog', () => {
    vi.setSystemTime(new Date('2026-11-11T12:29:59+07:00'))
    const onChange = vi.fn(async () => {})
    render(<ServiceActions booking={booking} pending={false} onChange={onChange} />)
    expect(screen.getByRole('button', { name: 'Mulai layanan' })).toBeDisabled()
    act(() => vi.advanceTimersByTime(1000))
    const button = screen.getByRole('button', { name: 'Mulai layanan' })
    expect(button).toBeEnabled()
    fireEvent.click(button)
    expect(onChange).toHaveBeenCalledExactlyOnceWith('serving')
    expect(screen.queryByText('Menunggu waktu kunjungan')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Tandai tidak hadir' })).not.toBeInTheDocument()
  })
  it('offers completion for an active service and disables writes while saving', () => {
    vi.setSystemTime(new Date('2026-11-11T12:40:00+07:00'))
    const onChange = vi.fn(async () => {})
    const view = render(
      <ServiceActions
        booking={{ ...booking, status: 'serving' }}
        pending={false}
        onChange={onChange}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Tandai selesai' }))
    expect(onChange).toHaveBeenCalledExactlyOnceWith('completed')
    view.rerender(
      <ServiceActions
        booking={{ ...booking, status: 'serving' }}
        pending={true}
        onChange={onChange}
      />,
    )
    expect(screen.getByRole('button', { name: 'Tandai selesai' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Mulai layanan' })).not.toBeInTheDocument()
  })
  it('shows no-show only once the scheduled end has passed, and no service action on terminal bookings', () => {
    vi.setSystemTime(new Date('2026-11-11T13:15:00+07:00'))
    const onChange = vi.fn(async () => {})
    const view = render(<ServiceActions booking={booking} pending={false} onChange={onChange} />)
    expect(screen.getByRole('button', { name: 'Tandai tidak hadir' })).toBeEnabled()
    for (const status of ['cancelled', 'completed', 'no-show'] as const) {
      view.rerender(
        <ServiceActions booking={{ ...booking, status }} pending={false} onChange={onChange} />,
      )
      expect(screen.queryByRole('button')).not.toBeInTheDocument()
    }
  })
})
