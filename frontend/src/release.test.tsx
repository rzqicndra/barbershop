import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'
import { api } from './api'
import { DeleteDialog, Modal } from './ui'
import { BookingTicket } from './booking'
import type { Booking, Studio } from './data'

vi.mock('./api', async (original) => ({
  ...(await original<typeof import('./api')>()),
  api: vi.fn(),
}))
beforeEach(() => vi.resetAllMocks())

it('keeps a pending deletion open when Escape requests native dialog cancellation', async () => {
  let finish: (value: unknown) => void = () => {}
  vi.mocked(api).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  const close = vi.fn()
  const deleted = vi.fn()
  render(
    <DeleteDialog
      title="Hapus layanan?"
      description="Booking lama dipertahankan."
      path="/admin/services/cut"
      version={1}
      onClose={close}
      onDeleted={deleted}
    />,
  )
  fireEvent.click(screen.getByRole('button', { name: 'Ya, hapus' }))
  expect(screen.getByRole('button', { name: 'Menghapus…' })).toBeDisabled()
  const cancel = new Event('cancel', { cancelable: true })
  fireEvent(screen.getByRole('dialog'), cancel)
  expect(cancel.defaultPrevented).toBe(true)
  expect(close).not.toHaveBeenCalled()
  expect(screen.getByRole('dialog')).toHaveAttribute('open')
  await act(async () => finish({}))
  await waitFor(() => expect(deleted).toHaveBeenCalledOnce())
  expect(close).toHaveBeenCalledOnce()
})

it('gives stacked dialogs separate accessible titles', () => {
  render(
    <>
      <Modal title="Detail kunjungan" onClose={vi.fn()}>
        <p>Ringkasan</p>
      </Modal>
      <Modal title="Ubah jadwal" onClose={vi.fn()}>
        <p>Formulir</p>
      </Modal>
    </>,
  )
  const detail = screen.getByRole('dialog', { name: 'Detail kunjungan' })
  const form = screen.getByRole('dialog', { name: 'Ubah jadwal' })
  expect(detail.getAttribute('aria-labelledby')).not.toBe(form.getAttribute('aria-labelledby'))
})

it('refreshes an open customer ticket so a completed visit unlocks reviews without reopening it', async () => {
  vi.useFakeTimers()
  try {
    const booking: Booking = {
      id: 'visit',
      code: 'GL-POLL',
      date: '2026-11-12',
      start: 600,
      serviceId: 'cut',
      serviceName: 'Potong rambut',
      duration: 45,
      price: 65000,
      barberId: 'raka',
      barberName: 'Raka',
      customer: 'Pengunjung uji',
      source: 'online',
      status: 'confirmed',
      version: 1,
    }
    const studio: Studio = {
      profile: {
        name: 'Gilir Barbershop',
        location: 'Tangerang',
        open: 600,
        close: 1260,
        closedDates: [],
        version: 1,
      },
      services: [],
      barbers: [],
      timeOff: [],
    }
    vi.mocked(api)
      .mockResolvedValueOnce(booking)
      .mockResolvedValue({ ...booking, status: 'completed', version: 3 })
    await act(async () => {
      render(
        <BookingTicket
          studio={studio}
          refresh={vi.fn(async () => {})}
          token="private"
          created={false}
        />,
      )
    })
    expect(screen.getByRole('button', { name: 'Beri ulasan setelah selesai' })).toBeDisabled()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(30_000)
    })
    expect(screen.getByRole('radio', { name: '5 bintang' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Kirim ulasan' })).toBeEnabled()
  } finally {
    vi.useRealTimers()
  }
})
