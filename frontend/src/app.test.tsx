import { act, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './app'
import { api, APIError, getSession } from './api'
import type { Studio, User } from './data'

vi.mock('./api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./api')>()),
  api: vi.fn(),
  getSession: vi.fn(),
  setCSRF: vi.fn(),
}))
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
const admin: User = {
  id: 'owner',
  name: 'Pemilik',
  email: 'owner@gilir.test',
  role: 'admin',
  active: true,
}
const barber: User = {
  id: 'barber',
  name: 'Raka',
  email: 'raka@gilir.test',
  role: 'barber',
  barberId: 'raka',
  active: true,
}

beforeEach(() => {
  vi.clearAllMocks()
  window.history.replaceState(null, '', '/')
  vi.mocked(api).mockImplementation(async (path) => {
    if (path === '/public' || path === '/admin/catalog') return studio
    if (path.startsWith('/staff/bookings?')) return []
    throw new Error('Unexpected API in navigation test: ' + path)
  })
})

describe('staff session navigation', () => {
  it('keeps an Admin who removes /admin in the Admin dashboard', async () => {
    vi.mocked(getSession).mockResolvedValue(admin)
    render(<App />)
    await screen.findByRole('heading', { name: 'Agenda kunjungan' })
    expect(window.location.pathname).toBe('/admin/agenda')
    expect(screen.queryByRole('heading', { name: /Pilih jam/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Lihat website' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Gilir Barbershop, dashboard staf' })).toHaveAttribute(
      'href',
      '/admin/agenda',
    )
  })
  it('keeps a Barber in their own schedule even when opening a public hash route', async () => {
    window.history.replaceState(null, '', '/#booking')
    vi.mocked(getSession).mockResolvedValue(barber)
    render(<App />)
    await screen.findByRole('heading', { name: 'Jadwal saya' })
    expect(window.location.pathname).toBe('/admin/jadwal')
    expect(screen.queryByRole('link', { name: 'Pengaturan' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: /Pilih layanan/ })).not.toBeInTheDocument()
  })
  it('shows the public homepage to visitors without a staff session', async () => {
    vi.mocked(getSession).mockRejectedValue(new APIError(401, 'LOGIN_REQUIRED', 'Silakan masuk.'))
    render(<App />)
    await screen.findByRole('heading', { name: /Pilih jam.*Potong rapi/ })
    expect(window.location.pathname).toBe('/')
  })
  it('does not render the public page while a staff session is still being checked', async () => {
    let finish: (user: User) => void = () => {}
    vi.mocked(getSession).mockImplementation(
      () =>
        new Promise<User>((resolve) => {
          finish = resolve
        }),
    )
    render(<App />)
    await waitFor(() => expect(getSession).toHaveBeenCalled())
    expect(screen.queryByRole('heading', { name: /Pilih jam/ })).not.toBeInTheDocument()
    await act(async () => finish(barber))
    await screen.findByRole('heading', { name: 'Jadwal saya' })
    expect(window.location.pathname).toBe('/admin/jadwal')
  })
  it('returns an expired session to the login screen instead of leaving a staff page open', async () => {
    vi.mocked(getSession).mockResolvedValue(barber)
    render(<App />)
    await screen.findByRole('heading', { name: 'Jadwal saya' })
    act(() => window.dispatchEvent(new Event('gilir:session-ended')))
    await screen.findByRole('heading', { name: 'Masuk area staf.' })
    expect(window.location.pathname).toBe('/admin')
  })
})
