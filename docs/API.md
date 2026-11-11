# API Gilir v1

Prefix semua endpoint adalah `/api/v1`. Body berupa satu objek JSON. Respons kegagalan memakai bentuk `{"error":{"code":"...","message":"..."}}`. `version` diperlukan pada perubahan booking dan pembaruan katalog.

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
| GET | `/public` | Profil, layanan aktif, dan barber aktif |
| GET | `/availability?serviceId=...&barberId=...&date=YYYY-MM-DD` | Pilihan jam mulai dalam menit sejak 00.00 WIB |
| POST | `/bookings` | Membuat booking online |
| GET | `/bookings/current` | Satu booking sesuai token |
| GET | `/bookings/current/availability?date=...` | Pilihan pengganti dengan durasi booking yang sudah disimpan |
| POST | `/bookings/current/reschedule` | Mengubah jadwal sendiri |
| POST | `/bookings/current/cancel` | Membatalkan sendiri |

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

Akun Admin dibuat lewat CLI `admin:create`, bukan melalui pendaftaran publik. Tidak ada endpoint untuk mengangkat role dari browser.

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
| 409 `REQUEST_KEY_REUSED` | Kunci sudah dipakai untuk isian berbeda |
| 410 `BOOKING_LINK_EXPIRED` | Hubungi barbershop untuk informasi kunjungan lama |
| 415 `JSON_REQUIRED` | Kirim body dengan Content-Type application/json |
| 422 `INVALID_INPUT` | Perbaiki isian atau periksa batas waktu bisnis |
| 429 `RATE_LIMITED` | Tunggu satu menit sebelum mencoba lagi |

Daftar slot adalah informasi pada saat dibaca. Pemeriksaan transaksi pada saat menyimpan tetap menjadi keputusan akhir.
