import { useContext, useState } from 'react'
import {
  ArrowDown,
  ArrowRight,
  ArrowUpRight,
  Clock3,
  Instagram,
  MapPin,
  Menu,
  MessageCircle,
  ReceiptText,
  X,
} from 'lucide-react'
import { mapsLink, money, suitableBarbers, time, whatsappLink, type Studio } from './data'
import { BookingLink, Brand, BrandContext, Kicker, Mark } from './ui'
import { PhotoGallery } from './gallery'
import { PublicReviews } from './reviews'

const sections = [
  ['layanan', 'Layanan'],
  ['barber', 'Barber'],
  ['galeri', 'Galeri'],
  ['ulasan', 'Ulasan'],
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
  const profile = useContext(BrandContext)
  return (
    <footer className={'public-footer ' + (compact ? 'public-footer-compact' : '')}>
      <div className="wrap footer-inner">
        {!compact && <Brand />}
        <span>
          {profile?.name || 'Gilir Barbershop'}
          {(profile?.footerTagline ?? 'Pilih jam. Potong rapi.') && (
            <>
              <span className="footer-dot"> · </span>
              {profile?.footerTagline ?? 'Pilih jam. Potong rapi.'}
            </>
          )}
        </span>
      </div>
    </footer>
  )
}
export function Home({ studio }: { studio: Studio }) {
  const { profile } = studio
  const [category, setCategory] = useState('')
  const services = studio.services.filter((s) => s.active)
  const categories = [...new Set(services.map((s) => s.category || 'Potong & styling'))]
  const filtered = category
    ? services.filter((s) => (s.category || 'Potong & styling') === category)
    : services
  const barbers = studio.barbers.filter((b) => b.active)
  const hours = time(profile.open) + '–' + time(profile.close) + ' WIB'
  const whatsapp = whatsappLink(profile)
  return (
    <>
      <PublicHeader />
      <main id="konten" tabIndex={-1}>
        <div className="wrap" id="beranda">
          <section className="hero" aria-labelledby="hero-title">
            <div className="hero-copy">
              {(profile.heroEyebrow ?? 'BARBERSHOP DI TANGERANG') && (
                <div className="hero-kicker">
                  <span className="hero-rule" />
                  <Kicker>{profile.heroEyebrow ?? 'BARBERSHOP DI TANGERANG'}</Kicker>
                </div>
              )}
              <h1 id="hero-title">
                {(profile.heroTitle || 'Pilih jam.\nPotong rapi.')
                  .split('\n')
                  .map((line, index) => (
                    <span className="hero-title-line" key={index}>
                      {line.endsWith('.') ? (
                        <>
                          {line.slice(0, -1)}
                          <span>.</span>
                        </>
                      ) : (
                        line
                      )}
                    </span>
                  ))}
              </h1>
              <p>
                {profile.heroDescription ||
                  'Pilih layanan, barber, dan jam yang pas. Datang sesuai jadwal, lalu bayar langsung di barbershop.'}
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
                  src={profile.heroImageUrl || '/assets/gilir-chair.webp'}
                  alt={'Suasana ' + profile.name}
                  width="1440"
                  height="1080"
                  fetchPriority="high"
                />
              </div>
            </figure>
          </section>
          <div className="visit-strip" aria-label="Informasi kunjungan">
            <div>
              <Clock3 size={20} />
              <span>
                <small>JAM BUKA</small>
                <strong>{hours}</strong>
              </span>
            </div>
            <div>
              <MapPin size={20} />
              <span>
                <small>LOKASI</small>
                <strong>{profile.location}</strong>
              </span>
            </div>
            <div>
              <ReceiptText size={20} />
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
              <Kicker>01 / LAYANAN</Kicker>
              <h2 id="layanan-title">Layanan &amp; harga</h2>
              <p>Durasi setiap layanan ikut menentukan pilihan jam yang tersedia.</p>
            </div>
            <div>
              {categories.length > 1 && (
                <div className="category-filters" aria-label="Kategori layanan">
                  <button type="button" aria-pressed={!category} onClick={() => setCategory('')}>
                    Semua
                  </button>
                  {categories.map((name) => (
                    <button
                      type="button"
                      key={name}
                      aria-pressed={category === name}
                      onClick={() => setCategory(name)}
                    >
                      {name}
                    </button>
                  ))}
                </div>
              )}
              <div className="price-list">
                {!services.length && <p>Daftar layanan sedang diperbarui.</p>}
                {filtered.map((service, index) => {
                  const bookable = suitableBarbers(studio, service.id).length > 0
                  return (
                    <a
                      className="price-row"
                      key={service.id}
                      href={
                        bookable
                          ? '#booking?layanan=' + encodeURIComponent(service.id)
                          : whatsapp || '#lokasi'
                      }
                      aria-label={
                        (bookable ? 'Booking ' : 'Hubungi kami untuk ') +
                        service.name +
                        ', ' +
                        money(service.price)
                      }
                    >
                      <span className="row-index">{String(index + 1).padStart(2, '0')}</span>
                      <span className="price-desc">
                        <strong>{service.name}</strong>
                        <span>{service.description}</span>
                      </span>
                      <span className="price-meta">
                        <strong>{money(service.price)}</strong>
                        <small>{service.duration} menit</small>
                        {!bookable && <small>Konsultasi ketersediaan</small>}
                      </span>
                      <span className="price-arrow">
                        <ArrowUpRight size={21} />
                      </span>
                    </a>
                  )
                })}
              </div>
            </div>
          </section>
          <section id="barber" className="team-section section-grid" aria-labelledby="barber-title">
            <div className="section-intro">
              <Kicker>02 / TIM KAMI</Kicker>
              <h2 id="barber-title">Di balik kursi.</h2>
              <p>Kenali gaya potong tiap barber sebelum memilih jadwal.</p>
            </div>
            <div className="crew-list">
              {barbers.map((barber, index) => (
                <div className="crew-row" key={barber.id}>
                  <span className="crew-number">{String(index + 1).padStart(2, '0')}</span>
                  <strong>{barber.name}</strong>
                  <span>{barber.specialty}</span>
                </div>
              ))}
              {!barbers.length && <p>Jadwal barber sedang diperbarui.</p>}
            </div>
          </section>
          <section id="galeri" className="gallery-section" aria-labelledby="gallery-title">
            <div className="gallery-intro">
              <div>
                <Kicker>03 / HASIL KARYA</Kicker>
                <h2 id="gallery-title">{profile.galleryTitle || 'Hasil dari kursi kami.'}</h2>
              </div>
              <p>
                {profile.galleryDescription ??
                  'Pilihan hasil potongan dan perawatan dari tim barber kami.'}
              </p>
            </div>
            <PhotoGallery
              photos={studio.gallery || []}
              title={profile.galleryTitle || 'Galeri hasil karya'}
            />
          </section>
          <section
            id="ulasan"
            className="reviews-section section-grid"
            aria-labelledby="reviews-title"
          >
            <div className="section-intro">
              <Kicker>04 / DARI PELANGGAN</Kicker>
              <h2 id="reviews-title">Cerita setelah berkunjung.</h2>
              <p>Rating dan ulasan hanya dapat diberikan dari kunjungan yang sudah selesai.</p>
            </div>
            <PublicReviews studio={studio} />
          </section>
          <section
            id="lokasi"
            className="location-section section-grid"
            aria-labelledby="lokasi-title"
          >
            <div className="section-intro">
              <Kicker>05 / KUNJUNGAN</Kicker>
              <h2 id="lokasi-title">Sampai ketemu.</h2>
              <p>
                {profile.name}
                <br />
                {profile.location}
                <br />
                Jam buka {hours}
              </p>
              <div className="contact-links">
                <a
                  className="button button-outline"
                  href={mapsLink(profile)}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <MapPin size={17} />
                  Buka Google Maps
                </a>
                {profile.instagramUrl && (
                  <a
                    className="text-link"
                    href={profile.instagramUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Instagram size={18} />
                    Instagram
                    <ArrowUpRight size={15} />
                  </a>
                )}
                {whatsapp && (
                  <a
                    className="text-link"
                    href={whatsapp}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <MessageCircle size={18} />
                    Kritik &amp; saran via WhatsApp
                    <ArrowUpRight size={15} />
                  </a>
                )}
              </div>
            </div>
            <div className="location-content">
              {profile.mapsEmbedUrl && (
                <iframe
                  className="location-map"
                  src={profile.mapsEmbedUrl}
                  title={'Peta lokasi ' + profile.name}
                  loading="lazy"
                  referrerPolicy="no-referrer"
                  allowFullScreen
                />
              )}
              <div className="visit-faq">
                {(studio.faqs || [])
                  .filter((item) => item.active)
                  .map((item) => (
                    <details key={item.id}>
                      <summary>{item.question}</summary>
                      <p>{item.answer}</p>
                    </details>
                  ))}
              </div>
            </div>
            <Mark className="location-watermark" />
          </section>
        </div>
      </main>
      <PublicFooter />
    </>
  )
}
