import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api, APIError } from './api'
import type { Booking, Review, Studio, User } from './data'
import { Home } from './public'
import { ImagePicker } from './public-settings'
import { BookingReview } from './reviews'
import { StaffAccounts } from './staff-settings'
import { BrandContext, DeleteDialog } from './ui'

vi.mock('./api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./api')>()),
  api: vi.fn(),
}))

const booking: Booking = {
  id: 'booking',
  code: 'GL-REVIEW',
  date: '2026-11-11',
  start: 660,
  serviceId: 'cut',
  serviceName: 'Potong rambut',
  duration: 45,
  price: 65000,
  barberId: 'raka',
  barberName: 'Raka',
  customer: 'Rizqi Pratama',
  source: 'online',
  status: 'completed',
  version: 3,
}
const review: Review = {
  id: 'review',
  rating: 5,
  displayName: 'Rizqi',
  comment: 'Hasilnya rapi.',
  serviceName: 'Potong rambut',
  barberName: 'Raka',
  createdAt: '2026-11-11T19:00:00Z',
}
const studio: Studio = {
  profile: {
    name: 'Kursi Sore Barbershop',
    wordmark: 'kursi sore',
    location: 'Alamat milik client',
    open: 600,
    close: 1260,
    closedDates: [],
    version: 2,
    heroTitle: 'Rapi setiap hari.',
    heroDescription: 'Deskripsi yang ditulis pemilik usaha.',
    heroEyebrow: 'BARBERSHOP PILIHANMU',
    heroImageUrl: '/media/hero.jpg',
    galleryTitle: 'Dari pelanggan kami',
    galleryDescription: 'Potongan dari tim kami.',
    instagramUrl: 'https://www.instagram.com/kursisore/',
    whatsapp: '6281200000000',
    mapsUrl: 'https://maps.app.goo.gl/client',
    mapsEmbedUrl: 'https://www.google.com/maps/embed?pb=client',
  },
  services: [
    {
      id: 'cut',
      name: 'Potong rambut',
      description: 'Potong dan styling.',
      category: 'Potong',
      price: 65000,
      duration: 45,
      active: true,
      version: 1,
    },
    {
      id: 'color',
      name: 'Hair coloring',
      description: 'Warna sesuai konsultasi.',
      category: 'Warna khusus',
      price: 250000,
      duration: 150,
      active: true,
      version: 1,
    },
    {
      id: 'perm',
      name: 'Perming',
      description: 'Ikal sesuai karakter rambut.',
      category: 'Tekstur',
      price: 350000,
      duration: 180,
      active: true,
      version: 1,
    },
  ],
  barbers: [
    {
      id: 'raka',
      name: 'Raka',
      specialty: 'Potong & warna',
      serviceIds: ['cut', 'color'],
      start: 600,
      end: 1260,
      breakStart: 840,
      breakEnd: 885,
      daysOff: [],
      active: true,
      version: 1,
    },
  ],
  timeOff: [],
  gallery: [
    {
      id: 'photo',
      imageUrl: '/media/customer.jpg',
      title: 'Classic cut',
      caption: 'Potong oleh Raka',
      altText: 'Hasil classic cut pelanggan',
      sortOrder: 1,
      active: true,
      consentConfirmed: true,
      version: 1,
    },
  ],
  faqs: [
    {
      id: 'faq',
      question: 'Apakah menerima coloring?',
      answer: 'Tersedia setelah konsultasi dengan barber.',
      sortOrder: 1,
      active: true,
      version: 1,
    },
  ],
  reviews: [review],
  rating: { average: 5, count: 1 },
}

beforeEach(() => vi.resetAllMocks())

describe('customer reviews', () => {
  it.each(['confirmed', 'serving', 'cancelled', 'no-show'] as const)(
    'does not offer a review form for %s bookings',
    (status) => {
      render(<BookingReview booking={{ ...booking, status }} token="private" onUpdated={vi.fn()} />)
      expect(screen.getByRole('button', { name: 'Beri ulasan setelah selesai' })).toBeDisabled()
      expect(screen.queryByRole('radio')).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Kirim ulasan' })).not.toBeInTheDocument()
      expect(api).not.toHaveBeenCalled()
    },
  )
  it('submits a completed visit using its private token and the selected public name', async () => {
    vi.mocked(api).mockResolvedValue(review)
    const updated = vi.fn(async () => {})
    render(<BookingReview booking={booking} token="private-link-token" onUpdated={updated} />)
    fireEvent.click(screen.getByRole('radio', { name: '5 bintang' }))
    fireEvent.change(screen.getByLabelText(/^Nama tampilan/), { target: { value: '  Rizqi  ' } })
    fireEvent.change(screen.getByLabelText('Ulasan (opsional)'), {
      target: { value: ' Hasilnya rapi. ' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Kirim ulasan' }))
    await waitFor(() => expect(updated).toHaveBeenCalledOnce())
    expect(api).toHaveBeenCalledExactlyOnceWith('/bookings/current/review', {
      method: 'POST',
      token: 'private-link-token',
      body: { rating: 5, displayName: 'Rizqi', comment: 'Hasilnya rapi.' },
    })
  })
  it('shows the saved review without allowing a second rating', () => {
    render(<BookingReview booking={{ ...booking, review }} token="private" onUpdated={vi.fn()} />)
    expect(screen.getByText('Terima kasih untuk ulasanmu.')).toBeInTheDocument()
    expect(screen.getByText(review.comment)).toBeInTheDocument()
    expect(screen.queryByRole('radio')).not.toBeInTheDocument()
  })
  it('reloads the visit if another tab has already submitted its review', async () => {
    vi.mocked(api).mockRejectedValue(
      new APIError(409, 'REVIEW_EXISTS', 'Kunjungan ini sudah memiliki ulasan.'),
    )
    const updated = vi.fn(async () => {})
    render(<BookingReview booking={booking} token="private" onUpdated={updated} />)
    fireEvent.click(screen.getByRole('radio', { name: '4 bintang' }))
    fireEvent.click(screen.getByRole('button', { name: 'Kirim ulasan' }))
    await waitFor(() => expect(updated).toHaveBeenCalledOnce())
    expect(screen.getByRole('alert')).toHaveTextContent('sudah memiliki ulasan')
  })
})

describe('dynamic public page', () => {
  function home() {
    return render(
      <BrandContext.Provider value={studio.profile}>
        <Home studio={studio} />
      </BrandContext.Provider>,
    )
  }
  it('renders owner content, real gallery data, map and contact links without the removed caption', () => {
    home()
    expect(screen.getByRole('heading', { name: 'Rapi setiap hari.' })).toBeInTheDocument()
    expect(screen.getByText(studio.profile.heroDescription!)).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /Kursi Sore Barbershop, beranda/ })).toHaveLength(2)
    expect(screen.getByRole('img', { name: 'Hasil classic cut pelanggan' })).toHaveAttribute(
      'src',
      '/media/customer.jpg',
    )
    expect(screen.getByRole('link', { name: 'Buka Google Maps' })).toHaveAttribute(
      'href',
      studio.profile.mapsUrl,
    )
    expect(screen.getByTitle(/Peta/)).toHaveAttribute('src', studio.profile.mapsEmbedUrl)
    expect(screen.getByRole('link', { name: /Instagram/ })).toHaveAttribute(
      'href',
      studio.profile.instagramUrl,
    )
    expect(screen.getByRole('link', { name: /Kritik/ })).toHaveAttribute(
      'href',
      expect.stringContaining('https://wa.me/6281200000000'),
    )
    expect(screen.getByText('Apakah menerima coloring?')).toBeInTheDocument()
    expect(screen.queryByText('DI KURSI CUKUR')).not.toBeInTheDocument()
    expect(screen.queryByText('01 / GADING SERPONG')).not.toBeInTheDocument()
    expect(screen.getByText(/Potong rambut · Raka · 12 November/)).toBeInTheDocument()
  })
  it('derives category filters from the catalog and sends unassigned treatments to consultation', () => {
    home()
    fireEvent.click(screen.getByRole('button', { name: 'Warna khusus' }))
    expect(screen.getByRole('link', { name: 'Booking Hair coloring, Rp250.000' })).toHaveAttribute(
      'href',
      '#booking?layanan=color',
    )
    expect(
      screen.queryByRole('link', { name: 'Booking Potong rambut, Rp65.000' }),
    ).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Tekstur' }))
    expect(
      screen.getByRole('link', { name: 'Hubungi kami untuk Perming, Rp350.000' }),
    ).toHaveAttribute('href', expect.stringContaining('https://wa.me/'))
    expect(screen.getByText('Konsultasi ketersediaan')).toBeInTheDocument()
  })
})

describe('Admin deletion and media', () => {
  it('requires confirmation and retains the dialog when active bookings block deletion', async () => {
    vi.mocked(api).mockRejectedValue(
      new APIError(409, 'BOOKINGS_AFFECTED', 'Tangani booking GL-ACTIVE dahulu.'),
    )
    const deleted = vi.fn()
    const close = vi.fn()
    render(
      <DeleteDialog
        title="Hapus layanan?"
        description="Riwayat tetap tersimpan."
        path="/admin/services/cut"
        version={4}
        onClose={close}
        onDeleted={deleted}
      />,
    )
    expect(api).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Ya, hapus' }))
    await screen.findByRole('alert')
    expect(api).toHaveBeenCalledExactlyOnceWith('/admin/services/cut', {
      method: 'DELETE',
      body: { version: 4 },
    })
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(deleted).not.toHaveBeenCalled()
    expect(close).not.toHaveBeenCalled()
  })
  it('protects the current account and deletes another account after confirmation', async () => {
    const staff: User[] = [
      { id: 'owner', email: 'owner@example.test', name: 'Pemilik', role: 'admin', active: true },
      {
        id: 'barber',
        email: 'raka@example.test',
        name: 'Raka',
        role: 'barber',
        barberId: 'raka',
        active: true,
      },
    ]
    vi.mocked(api).mockResolvedValue(staff)
    render(<StaffAccounts studio={studio} userId="owner" />)
    expect(await screen.findByRole('button', { name: 'Akun ini' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Hapus akun' }))
    expect(api).not.toHaveBeenCalledWith('/admin/staff/barber', expect.anything())
    vi.mocked(api).mockResolvedValue([staff[0]])
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Ya, hapus' }))
    await screen.findByText('Akun berhasil dihapus.')
    expect(api).toHaveBeenCalledWith('/admin/staff/barber', { method: 'DELETE', body: undefined })
    expect(screen.queryByText('Raka · Barber')).not.toBeInTheDocument()
  })
  it('uploads a selected image as multipart and returns its server URL to the editor', async () => {
    vi.mocked(api).mockResolvedValue({
      id: 'asset',
      url: '/media/new.jpg',
      width: 100,
      height: 100,
      size: 42,
    })
    const changed = vi.fn()
    render(<ImagePicker label="Foto galeri" value="" onChange={changed} />)
    const file = new File(['example image'], 'customer.png', { type: 'image/png' })
    fireEvent.change(screen.getByLabelText('Foto galeri — unggah foto'), {
      target: { files: [file] },
    })
    await waitFor(() => expect(changed).toHaveBeenCalledExactlyOnceWith('/media/new.jpg'))
    const call = vi.mocked(api).mock.calls[0]
    expect(call[0]).toBe('/admin/media')
    expect(call[1]?.method).toBe('POST')
    expect(call[1]?.body).toBeInstanceOf(FormData)
    expect((call[1]?.body as FormData).get('file')).toBe(file)
  })
})
