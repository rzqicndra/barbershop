import { useState } from 'react'
import {
  ArrowDown,
  ArrowRight,
  ArrowUpRight,
  Clock3,
  MapPin,
  Menu,
  ReceiptText,
  X,
} from 'lucide-react'
import { money, time, type Studio } from './data'
import { BookingLink, Brand, Kicker, Mark } from './ui'

const sections = [
  ['layanan', 'Layanan'],
  ['barber', 'Barber'],
  ['lokasi', 'Lokasi'],
]

export function PublicHeader({ compact = false }: { compact?: boolean }) {
  const [open, setOpen] = useState(false)
  return (
    <header className={'public-header ' + (compact ? 'public-header-compact' : '')}>
      <div className="wrap masthead">
        <Brand />
        {!compact && (
          <nav className="public-nav" aria-label="Navigasi utama">
            {sections.map(([id, label]) => (
              <a key={id} href={'#' + id}>
                {label}
              </a>
            ))}
          </nav>
        )}
        <div className="masthead-right">
          {!compact && <BookingLink>Booking</BookingLink>}
          {!compact && (
            <button
              type="button"
              className="icon-button mobile-menu"
              aria-label={open ? 'Tutup menu' : 'Buka menu'}
              aria-expanded={open}
              aria-controls="mobile-nav"
              onClick={() => setOpen(!open)}
            >
              {open ? <X size={22} /> : <Menu size={22} />}
            </button>
          )}
        </div>
      </div>
      {!compact && open && (
        <nav id="mobile-nav" className="mobile-nav wrap" aria-label="Navigasi ponsel">
          {sections.map(([id, label]) => (
            <a key={id} href={'#' + id} onClick={() => setOpen(false)}>
              {label}
              <ArrowRight size={16} />
            </a>
          ))}
        </nav>
      )}
    </header>
  )
}

export function PublicFooter({ compact = false }: { compact?: boolean }) {
  return (
    <footer className={'public-footer ' + (compact ? 'public-footer-compact' : '')}>
      <div className="wrap footer-inner">
        {!compact && <Brand />}
        <span>
          Gilir Barbershop <span className="footer-dot">·</span> Pilih jam. Potong rapi.
        </span>
      </div>
    </footer>
  )
}

export function Home({ studio }: { studio: Studio }) {
  const { profile } = studio
  const activeServices = studio.services.filter((s) => s.active)
  const activeBarbers = studio.barbers.filter((b) => b.active)
  const hours = time(profile.open) + '–' + time(profile.close) + ' WIB'
  return (
    <>
      <PublicHeader />
      <main id="konten" tabIndex={-1}>
        <div className="wrap" id="beranda">
          <section className="hero" aria-labelledby="hero-title">
            <div className="hero-copy">
              <div className="hero-kicker">
                <span className="hero-rule" />
                <Kicker>BARBERSHOP DI TANGERANG</Kicker>
              </div>
              <h1 id="hero-title">
                Pilih jam<span>.</span>
                <br />
                Potong rapi<span>.</span>
              </h1>
              <p>
                Pilih layanan, barber, dan jam yang pas. Datang sesuai jadwal, lalu bayar langsung
                di barbershop.
              </p>
              <div className="hero-actions">
                <BookingLink>Pilih jadwal</BookingLink>
                <a href="#layanan" className="text-link">
                  Lihat layanan
                  <ArrowDown size={16} aria-hidden="true" />
                </a>
              </div>
            </div>
            <figure className="hero-figure">
              <div className="hero-image">
                <img
                  src="/assets/gilir-chair.webp"
                  alt="Barber sedang merapikan rambut pelanggan di kursi cukur"
                  width="1440"
                  height="1080"
                  fetchPriority="high"
                />
              </div>
              <figcaption>
                <span>
                  <i aria-hidden="true" /> DI KURSI CUKUR
                </span>
                <span>01 / GADING SERPONG</span>
              </figcaption>
            </figure>
          </section>

          <div className="visit-strip" aria-label="Informasi kunjungan">
            <div>
              <Clock3 size={20} aria-hidden="true" />
              <span>
                <small>JAM BUKA</small>
                <strong>{hours}</strong>
              </span>
            </div>
            <div>
              <MapPin size={20} aria-hidden="true" />
              <span>
                <small>LOKASI</small>
                <strong>{profile.location}</strong>
              </span>
            </div>
            <div>
              <ReceiptText size={20} aria-hidden="true" />
              <span>
                <small>PEMBAYARAN</small>
                <strong>Langsung di tempat</strong>
              </span>
            </div>
          </div>

          <section
            id="layanan"
            className="service-section section-grid"
            aria-labelledby="layanan-title"
          >
            <div className="section-intro">
              <Kicker>01 / MENU POTONG</Kicker>
              <h2 id="layanan-title">Layanan &amp; harga</h2>
              <p>Durasi setiap layanan ikut menentukan pilihan jam yang tersedia.</p>
            </div>
            <div className="price-list">
              {activeServices.length === 0 && <p>Daftar layanan sedang diperbarui.</p>}
              {activeServices.map((service, index) => (
                <a
                  className="price-row"
                  key={service.id}
                  href={'#booking?layanan=' + encodeURIComponent(service.id)}
                  aria-label={'Booking ' + service.name + ', ' + money(service.price)}
                >
                  <span className="row-index">{String(index + 1).padStart(2, '0')}</span>
                  <span className="price-desc">
                    <strong>{service.name}</strong>
                    <span>{service.description}</span>
                  </span>
                  <span className="price-meta">
                    <strong>{money(service.price)}</strong>
                    <small>{service.duration} menit</small>
                  </span>
                  <span className="price-arrow">
                    <ArrowUpRight size={21} aria-hidden="true" />
                  </span>
                </a>
              ))}
            </div>
          </section>

          <section id="barber" className="team-section section-grid" aria-labelledby="barber-title">
            <div className="section-intro">
              <Kicker>02 / TIM KAMI</Kicker>
              <h2 id="barber-title">Di balik kursi.</h2>
              <p>Kenali gaya potong tiap barber sebelum memilih jadwal.</p>
            </div>
            <div className="crew-list">
              {activeBarbers.map((barber, index) => (
                <div className="crew-row" key={barber.id}>
                  <span className="crew-number">{String(index + 1).padStart(2, '0')}</span>
                  <strong>{barber.name}</strong>
                  <span>{barber.specialty}</span>
                </div>
              ))}
              {activeBarbers.length === 0 && <p>Jadwal barber sedang diperbarui.</p>}
            </div>
          </section>

          <section
            id="lokasi"
            className="location-section section-grid"
            aria-labelledby="lokasi-title"
          >
            <div className="section-intro">
              <Kicker>03 / KUNJUNGAN</Kicker>
              <h2 id="lokasi-title">Sampai ketemu.</h2>
              <p>
                {profile.name}
                <br />
                {profile.location}
                <br />
                Jam buka {hours}
              </p>
              <BookingLink>Pilih jadwal</BookingLink>
            </div>
            <div className="visit-faq">
              <details>
                <summary>Perlu buat akun untuk booking?</summary>
                <p>
                  Cukup isi nama dan nomor WhatsApp. Setelah konfirmasi, simpan tautan privat yang
                  muncul untuk membuka kembali booking kamu.
                </p>
              </details>
              <details>
                <summary>Kalau jadwal berubah?</summary>
                <p>
                  Buka tautan privat dari halaman konfirmasi. Kamu bisa mengubah jadwal atau
                  membatalkan booking paling lambat dua jam sebelum jam kunjungan.
                </p>
              </details>
              <details>
                <summary>Kapan saya membayar?</summary>
                <p>Pembayaran dilakukan di barbershop setelah kunjungan.</p>
              </details>
            </div>
            <Mark className="location-watermark" />
          </section>
        </div>
      </main>
      <PublicFooter />
    </>
  )
}
