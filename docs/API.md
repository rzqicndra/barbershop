# API Gilir v1

Prefix endpoint JSON adalah `/api/v1`. Body berupa satu objek JSON, kecuali unggahan foto memakai multipart. Respons kegagalan memakai bentuk `{"error":{"code":"...","message":"..."}}`. `version` diperlukan pada perubahan booking, pembaruan katalog, dan hapus layanan/barber/foto/FAQ.

## Pembatasan akses

| Kelompok | Identitas yang diperlukan |
| --- | --- |
| Public | Tanpa sesi, tanpa data booking pelanggan lain |
| Customer booking | `Authorization: Bearer <token dari fragment tautan privat>` |
| Staff | Cookie `gilir_session` dari login |
| Admin | Sesi dengan role `admin` di database |
| Staff write | Sesi sah dan `X-CSRF-Token` dari respons login atau session |

Cookie tidak dapat dibaca JavaScript. Frontend menyimpan CSRF di memori. Origin penulisan yang mengirim header Origin harus sesuai `PUBLIC_ORIGIN` atau daftar development `EXTRA_ORIGINS`.

## Endpoint publik dan pelanggan

| Metode | Path | Hasil |
| --- | --- | --- |
| GET | `/health` | Pemeriksaan koneksi database |
| GET | `/public` | Profil/kontak/tampilan, layanan dan barber aktif, galeri publik, FAQ aktif, 12 ulasan terbaru, serta ringkasan rating |
| GET | `/reviews?offset=0` | Maksimal 12 ulasan publik berikutnya, offset 0–100000 |
| GET | `/availability?serviceId=...&barberId=...&date=YYYY-MM-DD` | Pilihan jam mulai dalam menit sejak 00.00 WIB |
| POST | `/bookings` | Membuat booking online |
| GET | `/bookings/current` | Satu booking sesuai token dan ulasannya jika sudah ada |
| GET | `/bookings/current/availability?date=...` | Pilihan pengganti dengan durasi booking yang sudah disimpan |
| POST | `/bookings/current/reschedule` | Mengubah jadwal sendiri |
| POST | `/bookings/current/cancel` | Membatalkan sendiri |
| POST | `/bookings/current/review` | `{rating, displayName, comment}`, hanya booking selesai |

Contoh body pembuatan booking. Semua ID dan requestKey berupa UUID. Gunakan version layanan yang ditampilkan terakhir sebagai offerVersion.

```json
{
  "requestKey": "UUID baru yang dipertahankan untuk retry isian yang sama",
  "serviceId": "UUID layanan",
  "barberId": "UUID barber",
  "date": "2026-11-12",
  "start": 660,
  "customer": "Nama pelanggan",
  "phone": "081200000000",
  "offerVersion": 1
}
```

Booking baru menghasilkan 201, replay idempoten menghasilkan 200. Respons berisi `id`, `code`, `token`, snapshot layanan, jadwal, status, dan version. Token tidak perlu dikirim pada query URL API.

Reschedule memakai `{date, start, barberId, version}`. Pelanggan harus mempertahankan barber yang sama. Pembatalan cukup memakai `{version}`. Nama layanan, harga, durasi, dan status tidak diterima dari pelanggan sebagai keputusan server.

Ulasan menerima bintang integer 1–5, nama tampilan 2–40 karakter, dan komentar opsional maksimal 1.000 karakter. Satu booking hanya mendapat satu ulasan. Pengiriman pertama menghasilkan 201, retry dengan isian sama menghasilkan 200, dan isian berbeda setelah ada ulasan menghasilkan 409 `REVIEW_EXISTS`. Status selain `completed` menghasilkan 409 `REVIEW_NOT_READY`. Tautan tetap harus sah dan belum kedaluwarsa. Respons ulasan tidak memuat nomor, kode booking, atau token.

## Autentikasi dan staf

| Metode | Path | Isi body atau parameter |
| --- | --- | --- |
| POST | `/auth/login` | `{email, password}` |
| GET | `/auth/session` | Mendapatkan identitas dan csrfToken sesi |
| POST | `/auth/logout` | Menghapus sesi aktif |
| POST | `/auth/password` | `{currentPassword, newPassword}`, mengakhiri seluruh sesi akun |
| GET | `/staff/bookings?date=YYYY-MM-DD` | Agenda, disaring menurut barber di server |
| GET | `/staff/bookings/{id}` | Detail, hanya Admin atau barber yang ditugaskan |
| POST | `/staff/bookings/{id}/status` | `{status, version}` |

## Operasi khusus Admin

| Metode | Path | Fungsi |
| --- | --- | --- |
| GET | `/admin/catalog` | Katalog lengkap dan izin barber |
| GET | `/admin/availability?serviceId=...&barberId=...&date=...&mode=walk-in` | Pilihan walk-in, mode admin juga tersedia |
| POST | `/admin/bookings` | Walk-in, body sama dengan pembuatan online, nomor opsional |
| GET | `/admin/bookings/{id}/availability?date=...&barberId=...` | Pengganti sesuai snapshot durasi |
| POST | `/admin/bookings/{id}/reschedule` | `{date, start, barberId, version, reason}` |
| POST | `/admin/bookings/{id}/cancel` | `{version, reason}` |
| PUT | `/admin/profile` | Profil lengkap dengan version |
| POST / PUT | `/admin/services` atau `/admin/services/{id}` | Membuat atau memperbarui layanan |
| POST / PUT | `/admin/barbers` atau `/admin/barbers/{id}` | Membuat atau memperbarui barber dan penugasan layanannya |
| POST | `/admin/time-off` | `{barberId, date, start, end, reason}` |
| DELETE | `/admin/time-off/{id}` | Menghapus satu izin |
| GET | `/admin/staff` | Daftar akun tanpa hash kata sandi |
| POST | `/admin/staff` | `{email, name, password, barberId}`, membuat akun Barber |
| PUT | `/admin/staff/{id}` | `{active}`, hanya akun Barber |
| POST | `/admin/staff/{id}/password` | `{password}`, hanya akun Barber |
| DELETE | `/admin/staff/{id}` | Hapus akun, cabut semua sesi, lindungi akun sendiri dan Admin aktif terakhir |
| DELETE | `/admin/services/{id}` | `{version}`, hapus layanan jika tidak ada booking confirmed/serving |
| DELETE | `/admin/barbers/{id}` | `{version}`, hapus barber dan akses akun terkait jika tidak ada booking confirmed/serving |
| POST | `/admin/media` | Multipart satu part bernama `file`, menghasilkan `{id,url,width,height,size}` |
| POST / PUT | `/admin/gallery` atau `/admin/gallery/{id}` | `{imageUrl,title,caption,altText,sortOrder,active,consentConfirmed,version}` |
| DELETE | `/admin/gallery/{id}` | `{version}` |
| POST / PUT | `/admin/faqs` atau `/admin/faqs/{id}` | `{question,answer,sortOrder,active,version}` |
| DELETE | `/admin/faqs/{id}` | `{version}` |

Akun Admin dibuat lewat CLI `admin:create`, bukan melalui pendaftaran publik. Tidak ada endpoint untuk mengangkat role dari browser.

Layanan menerima kategori bebas maksimal 50 karakter dan durasi 15–360 menit. Penghapusan akun, layanan, dan barber memakai soft delete; tidak menghapus snapshot atau relasi historis booking. Email dan penugasan barber dari akun terhapus dapat digunakan akun pengganti dengan identitas baru.

Profil lengkap mencakup `mapsUrl`, `mapsEmbedUrl`, `instagramUrl`, `whatsapp`, `wordmark`, `logoImageUrl`, `heroEyebrow`, `heroTitle`, `heroDescription`, `heroImageUrl`, `footerTagline`, `galleryTitle`, dan `galleryDescription`. Kirim profil lengkap beserta `version`, bukan patch parsial. Editor kontak dan tampilan publik memakai endpoint profil yang sama, dengan pemeriksaan version untuk mencegah saling menimpa.

Galeri maksimal 60 foto dan FAQ maksimal 30 pertanyaan. Konten mempunyai `version`; publikasi foto memerlukan `consentConfirmed=true`. URL gambar harus berasal dari unggahan website. Upload menerima JPG/PNG/WebP maksimal 8 MB dan 24 megapiksel; file di-decode dan di-encode ulang sebagai JPEG. Unggahan memerlukan sesi Admin dan CSRF seperti operasi Admin lainnya.

Di luar prefix API, `GET /media/{filename}` menyajikan gambar JPEG yang dipakai sebagai logo/foto utama atau galeri publik. Admin juga dapat melihat unggahan privat. Nama file acak bukan kredensial akses. Gambar privat menghasilkan 404 bagi pengunjung/Barber; respons gambar memakai `Cache-Control: no-store` dan `X-Content-Type-Options: nosniff`.

Admin mendapat token tautan pelanggan pada respons pembuatan walk-in dan pembacaan detail booking melalui `/staff/bookings/{id}`. Respons untuk akun Barber tidak memuat token tersebut, nomor pelanggan, atau riwayat internal.

## Kode penting

| HTTP / code | Arti dan kelanjutan |
| --- | --- |
| 401 `LOGIN_REQUIRED` | Sesi berakhir, masuk kembali |
| 403 `FORBIDDEN` | Akun tidak mempunyai izin operasi |
| 403 `CSRF_INVALID` | Ambil sesi terbaru sebelum mengirim formulir lagi |
| 404 `NOT_FOUND` | Objek tidak tersedia pada cakupan akses akun |
| 409 `SLOT_TAKEN` | Pilih slot baru, booking lama tetap berlaku jika reschedule gagal |
| 409 `OFFER_CHANGED` | Muat ulang katalog dan konfirmasi penawaran terbaru |
| 409 `BOOKING_CHANGED` | Muat ulang detail sebelum mengubahnya |
| 409 `SETTINGS_CHANGED` | Muat ulang pengaturan sebelum menyimpan |
| 409 `BOOKINGS_AFFECTED` | Tangani kode booking yang disebut melalui agenda |
| 409 `BARBER_BUSY` | Selesaikan kunjungan lain yang masih Sedang dilayani oleh barber tersebut |
| 409 `REQUEST_KEY_REUSED` | Kunci sudah dipakai untuk isian berbeda |
| 409 `REVIEW_NOT_READY` | Tunggu sampai pelayanan benar-benar selesai dan status dicatat |
| 409 `REVIEW_EXISTS` | Muat ulang booking untuk melihat ulasan yang sudah tercatat |
| 410 `BOOKING_LINK_EXPIRED` | Hubungi barbershop untuk informasi kunjungan lama |
| 415 `JSON_REQUIRED` | Kirim body dengan Content-Type application/json |
| 422 `INVALID_INPUT` | Perbaiki isian atau periksa batas waktu bisnis |
| 429 `RATE_LIMITED` | Tunggu satu menit sebelum mencoba lagi |
| 503 `LOGIN_BUSY` / `UPLOAD_BUSY` | Pemrosesan sedang penuh; coba ulang mengikuti header Retry-After |

Daftar slot adalah informasi pada saat dibaca. Pemeriksaan transaksi pada saat menyimpan tetap menjadi keputusan akhir.

## Pengalihan halaman staf

Middleware Go memeriksa sesi yang masih berlaku di database untuk halaman masuk dan halaman staf. Permintaan API tetap memberikan JSON 401 atau 403 sesuai hak akses, tanpa mengalihkan fetch API ke halaman login. Aset statis dapat dimuat tanpa pemeriksaan sesi.

| Pengunjung | Halaman yang dibuka | Tujuan |
| --- | --- | --- |
| Admin dengan sesi aktif | `/` atau `/admin` | `/admin/agenda` |
| Barber dengan sesi aktif | `/` atau `/admin` | `/admin/jadwal` |
| Barber dengan sesi aktif | Halaman khusus Admin | `/admin/jadwal` |
| Pengunjung tanpa sesi aktif | Halaman staf selain login | `/admin` |
| Pengunjung tanpa sesi aktif | `/` | Beranda pelanggan |

Frontend menerapkan pengalihan yang sama untuk perpindahan hash atau history di dalam aplikasi. Cookie palsu, sesi berakhir, atau akun nonaktif tidak dianggap sebagai sesi staf yang sah.

## Proxy dan pemeriksaan operasional

`TRUSTED_PROXY_CIDRS` berisi CIDR proxy terpercaya yang dipisahkan koma. `X-Real-IP` hanya dibaca jika peer TCP termasuk di daftar tersebut dan nilainya satu alamat IP yang valid. CIDR seluruh internet ditolak. Caddy pada paket produksi mengganti nilai header pengunjung, sedangkan app hanya mempercayai IP Caddy yang tetap. Tanpa daftar tersebut, app memakai IP koneksi langsung.

Maksimal dua pemeriksaan login dan dua pemrosesan foto berjalan bersamaan per app. Kelebihan sementara mendapat 503 dengan Retry-After. Ini berbeda dari batas frekuensi per IP yang menghasilkan 429. Limiter tetap berada di memori satu instance.

CLI `gilir healthcheck` memeriksa endpoint kesehatan lokal dengan timeout tiga detik dan exit code nonzero bila tidak sehat. Image app menggunakannya sebagai Docker healthcheck. Healthcheck memeriksa HTTP dan koneksi database; bukan pengganti pengujian penerimaan atau pemeriksaan sertifikat domain publik.
