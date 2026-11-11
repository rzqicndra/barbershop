# Gilir Barbershop

Website booking untuk satu barbershop dengan beberapa barber. Frontend React mempertahankan rancangan Gilir yang sudah dibuat. Backend Go menyimpan booking, sesi staf, dan jadwal di PostgreSQL.

Versi pertama ini mencakup booking tanpa akun pelanggan, tautan privat per booking, perubahan dan pembatalan jadwal, agenda staf, walk-in, pengaturan usaha, serta akun Admin dan Barber. Pembayaran dilakukan di tempat. Integrasi WhatsApp otomatis, pembayaran online, multi-cabang, dan Super Admin belum termasuk.

## Mulai dari sini di Windows

Untuk menjalankan website, siapkan:

| Aplikasi | Kegunaan | Kebutuhan |
| --- | --- | --- |
| [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) | Menjalankan Go, PostgreSQL, dan proses build frontend di container Linux | Wajib untuk langkah di bawah |
| [VS Code](https://code.visualstudio.com/) | Membaca dan mengubah proyek | Disarankan |
| [Git](https://git-scm.com/downloads) | Mencatat perubahan kode | Disarankan |
| [Go](https://go.dev/doc/install), versi 1.27.x | Mengembangkan backend langsung di laptop | Opsional pada tahap pertama |
| [Node.js](https://nodejs.org/en/download), versi 24 LTS | Mengembangkan frontend langsung di laptop | Opsional pada tahap pertama |

Go dan Node sudah tersedia di tahap build Docker. Kamu belum perlu memasang keduanya di Windows untuk mencoba website. Tidak perlu memasang PostgreSQL secara terpisah atau menggunakan XAMPP untuk proyek ini.

1. Buka Docker Desktop dan tunggu sampai engine berjalan. Gunakan Linux containers. Untuk backend WSL 2, ikuti [panduan resmi Docker](https://docs.docker.com/desktop/features/wsl/).
2. Ekstrak ZIP proyek ke folder kerja, misalnya `C:\Projects\gilir-fullstack`.
3. Buka folder itu di VS Code, lalu buka terminal PowerShell. Pastikan terminal berada di folder yang berisi `compose.yaml`.
4. Periksa Docker sebelum melanjutkan.

```powershell
docker version
docker compose version
```

`docker version` harus menampilkan bagian Client dan Server. Jika hanya Client yang tampil atau muncul pesan tidak dapat terhubung ke engine, selesaikan masalah Docker Desktop terlebih dahulu.

### Buat konfigurasi lokal

Jalankan dari folder utama proyek.

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\setup.ps1
```

Perintah tersebut berlaku untuk proses skrip itu saja. Skrip membuat `.env` dengan dua kata sandi database dan kunci token acak. File yang sudah ada dipertahankan. Alternatif jika Node sudah terpasang adalah `node scripts/setup.mjs`.

Jangan mengirim atau commit `.env`. Simpan file ini karena kunci token dibutuhkan agar tautan booking lama tetap bisa digunakan setelah server dipindah atau dipulihkan.

### Build, database, dan akun pemilik

Jalankan satu per satu. Tunggu setiap perintah selesai sebelum melanjutkan. Build pertama memerlukan internet untuk mengunduh image dan dependensi.

```powershell
docker compose build app
docker compose up -d db
docker compose run --rm tools migrate
docker compose run --rm tools seed
docker compose run --rm tools admin:create
docker compose up -d app
```

Saat `admin:create`, isi email, nama, kata sandi 12–128 karakter, dan konfirmasinya. Kata sandi tidak terlihat ketika diketik. Tidak ada akun atau kata sandi bawaan.

`migrate` membuat struktur database. `seed` menambahkan tiga layanan dan tiga profil barber sesuai rancangan awal, tanpa membuat pelanggan, booking, atau akun barber. Menjalankan ulang `seed` tidak menimpa katalog yang sudah diedit.

Buka:

- Website publik di [http://localhost:8080](http://localhost:8080).
- Login staf di [http://localhost:8080/admin](http://localhost:8080/admin).
- Status backend di [http://localhost:8080/api/v1/health](http://localhost:8080/api/v1/health).

Setelah login sebagai Admin, buka Pengaturan. Sesuaikan profil usaha, harga, jadwal barber, dan hari tutup. Pada tab Akun staf, buat akun untuk masing-masing profil barber. Barber masuk melalui `/admin` yang sama dan mendapat halaman Jadwal saya berdasarkan akun yang terdaftar.

Nama barber, alamat, harga, dan foto awal merupakan materi contoh dari rancangan. Sesuaikan sebelum dipakai oleh usaha sebenarnya.

### Operasi sehari-hari

```powershell
# Melihat status container
docker compose ps

# Melihat log aplikasi
docker compose logs --tail=100 app

# Menghentikan tanpa menghapus data
docker compose down

# Menjalankan kembali setelah pernah melakukan instalasi
docker compose up -d

# Memperbarui kode aplikasi setelah mengubah frontend atau backend
docker compose build app
docker compose run --rm tools migrate
docker compose up -d app
```

Data berada di volume `pgdata`, bukan di container aplikasi. `docker compose down -v` menghapus volume database. Jangan gunakan opsi `-v` pada data yang ingin disimpan. Mengganti nilai kata sandi di `.env` tidak mengganti kata sandi pada database yang sudah diinisialisasi.

Jika port 8080 sudah dipakai, ubah `APP_PORT=8081` dan `PUBLIC_ORIGIN=http://localhost:8081` di `.env`, lalu jalankan `docker compose up -d app`.

## Struktur kode

| Lokasi | Isi |
| --- | --- |
| `backend/cmd/gilir/main.go` | Titik masuk server dan perintah administrasi |
| `backend/internal/app/http.go` | Route, respons JSON, middleware, dan penyajian frontend |
| `backend/internal/app/auth.go` | Login, cookie sesi, CSRF, logout, dan ganti kata sandi |
| `backend/internal/app/booking.go` | Transaksi booking, idempotensi, perubahan jadwal, dan status |
| `backend/internal/app/calendar.go` | Aturan waktu, jam kerja, istirahat, dan benturan jadwal |
| `backend/internal/app/availability.go` | Pilihan jam dari data server |
| `backend/internal/app/settings.go` | Pengaturan usaha, layanan, barber, izin, dan akun staf |
| `backend/internal/app/migrations/` | Migrasi SQL yang ditanamkan ke binary Go |
| `frontend/src/public.tsx` | Beranda dan halaman informasi publik |
| `frontend/src/booking.tsx` | Pemilihan jadwal dan tiket privat pelanggan |
| `frontend/src/staff.tsx` | Login, agenda, detail booking, dan editor katalog |
| `frontend/src/staff-settings.tsx` | Akun staf, kata sandi, dan izin barber |
| `frontend/src/api.ts` | Komunikasi API dan pembaruan ketersediaan |
| `frontend/src/styles.css` | Warna, tipografi, layout, dan responsivitas Gilir |
| `deploy/postgres/init.sh` | Membuat akun database aplikasi dengan izin terbatas |
| `scripts/` | Persiapan lokal, pengembangan, dan backup |

Backend memakai Chi, pgx, dan Argon2id. Kode dipisah menurut fungsi bisnis dalam satu paket aplikasi. Batas paket bisa dikembangkan saat kebutuhan bertambah.

Baca [panduan Go untuk proyek ini](docs/BELAJAR_GO.md), [kontrak API](docs/API.md), dan [catatan pengujian](docs/VALIDASI.md) setelah website berhasil berjalan.

## Aturan bisnis yang sudah berjalan

- Seluruh kalender memakai WIB. Waktu server menjadi acuan pemeriksaan akhir.
- Booking online paling cepat 30 menit dari sekarang, pada interval mulai 15 menit, untuk hari ini sampai 30 hari kalender berikutnya.
- Seluruh durasi harus masuk jam buka, jam kerja barber, serta di luar istirahat, libur rutin, tanggal tutup, dan izin barber.
- Booking online dan walk-in memakai kalender yang sama. Walk-in hanya dicatat untuk hari ini dan dapat dimulai pada menit berjalan.
- Pelanggan dapat mengubah jadwal dengan layanan dan barber yang sama atau membatalkan booking terkonfirmasi minimal dua jam sebelum mulai.
- Admin dapat mengubah jadwal atau barber dan membatalkan booking terkonfirmasi sebelum mulai, dengan alasan yang dicatat.
- Harga, nama layanan, dan durasi disalin saat booking dikonfirmasi. Perubahan katalog berikutnya tidak mengubah booking tersebut.
- Perubahan jadwal dilakukan dalam satu transaksi. Jika jadwal baru bentrok, jadwal sebelumnya tetap berlaku.
- Setiap perubahan booking membawa `version`. Permintaan dari detail lama ditolak agar tidak menimpa perubahan orang lain.
- Kunci idempotensi berlaku 24 jam. Mengirim ulang isian yang sama dengan kunci yang sama mengembalikan booking awal. Kunci yang sama dengan isian berbeda ditolak.
- Status bergerak dari `confirmed` ke `serving`, `cancelled`, atau `no-show`. `serving` dilanjutkan ke `completed`. Layanan mulai pada atau setelah waktu terjadwal. Tidak hadir dapat ditandai pada atau setelah waktu selesai terjadwal.
- Slot untuk `confirmed`, `serving`, dan `completed` tetap dicadangkan pada rentang rencana awal. Pembatalan melepas slot. Sistem belum melakukan perpanjangan otomatis jika pelayanan melewati durasi rencana.
- Perubahan katalog atau kalender yang mengganggu booking mendatang atau sedang dilayani ditolak dengan kode booking yang harus ditangani.
- Tautan privat hanya membuka satu booking dan berakhir 30 hari setelah waktu selesai jadwal terakhir. Tautan yang hilang perlu ditangani melalui Admin.

Sistem memakai kunci pada satu baris usaha untuk mengurutkan perubahan kalender, ditambah exclusion constraint PostgreSQL sebagai perlindungan benturan slot. Desain ini ditujukan untuk satu lokasi, bukan throughput multi-cabang dalam skala besar.

## Akses dan data

Admin mengelola seluruh operasional. Barber hanya dapat membaca booking yang ditugaskan kepadanya dan mengubah status pelayanannya. Nomor pelanggan dan riwayat internal tidak dikirim dalam respons akun barber. Tidak ada pemilih peran pada login atau role yang dipercaya dari browser.

Sesi menggunakan cookie HttpOnly, SameSite Lax, serta batas 12 jam di server. Semua perubahan staf memerlukan token CSRF. Menonaktifkan akun, menonaktifkan profil barber, reset kata sandi, atau ganti kata sandi mengakhiri sesi terkait. Hash kata sandi memakai Argon2id.

Pada Compose, akun `gilir_app` dipakai saat website berjalan dan tidak mendapat hak membuat skema. Akun pemilik database digunakan hanya oleh container `tools` untuk migrasi dan pembuatan Admin. Database tidak membuka port ke host pada konfigurasi utama.

Tautan privat memakai fragment URL dan dikirim ke API melalui header Authorization. Data booking dan sesi tidak disimpan di localStorage. Formulir menyimpan isian sementara di memori tab. Setelah tab ditutup atau dimuat ulang, isian yang belum terkirim tidak dipulihkan.

## Pengujian

Tes integrasi memakai database terpisah, tanpa mengakses data operasional.

```powershell
docker compose -f compose.test.yaml up --build --abort-on-container-exit --exit-code-from test
docker compose -f compose.test.yaml down
```

Perintah pertama harus berakhir dengan exit code 0. Tes menguji benturan slot bersamaan, idempotensi, snapshot harga dan durasi, perubahan jadwal yang gagal, pembatasan peran, CSRF, status layanan, dan izin barber.

## Backup dan pemulihan

Dari folder proyek pada PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\backup.ps1
```

Salinan berformat PostgreSQL custom dump akan berada di folder `backups`. Simpan salinan terenkripsi di lokasi terpisah bersama `.env` dan versi kode. Dump memuat data pelanggan dan hash akun, sehingga aksesnya harus terbatas.

Uji pemulihan ke database baru, bukan langsung ke database operasional. Ganti nama file contoh berikut dengan hasil backup yang benar.

```powershell
docker compose cp .\backups\gilir-20261111-120000.dump db:/tmp/restore-check.dump
docker compose exec db createdb -U gilir gilir_restore_check
docker compose exec db pg_restore -U gilir -d gilir_restore_check --exit-on-error /tmp/restore-check.dump
docker compose exec db psql -U gilir -d gilir_restore_check -c "SELECT count(*) FROM bookings;"
```

Periksa booking dan riwayat pada hasil pemulihan sebelum menetapkan prosedur produksi. Database `gilir_restore_check` berdiri sendiri dan tidak mengubah koneksi website. Jangan menggunakan `--clean` pada database operasional tanpa rencana pemulihan yang sudah disepakati.

## Sebelum digunakan publik

Proyek ini siap dijalankan lokal dan menjadi dasar pengujian penerimaan. Deployment domain publik belum dilakukan.

Gunakan server yang dapat menjalankan container Go dan PostgreSQL. Tempatkan reverse proxy HTTPS di depan aplikasi. Atur `PUBLIC_ORIGIN` ke origin domain sebenarnya dan `COOKIE_SECURE=true`. Ikat port aplikasi ke loopback jika reverse proxy berada di host yang sama. Pertahankan database di jaringan privat.

Pembatasan percobaan masuk dan pembuatan booking saat ini berada di memori satu proses dan memakai alamat koneksi langsung. Jika memakai reverse proxy, tambahkan pembatasan per IP pengunjung pada proxy tersebut. Untuk beberapa instance aplikasi, gunakan pembatasan bersama. Tidak ada kepercayaan otomatis terhadap header X-Forwarded-For dari pengunjung.

Sebelum rilis, lengkapi data usaha dan kontak bantuan, uji operasional bersama pemilik dan barber, pastikan backup bisa dipulihkan, serta tetapkan kebijakan penyimpanan data pelanggan. Tidak ada pengiriman WhatsApp otomatis pada versi ini. Nama dan nomor yang diisikan dipakai sebagai data booking internal.

Preview prototipe sebelumnya berjalan pada layanan berbeda. Paket Go/PostgreSQL ini tidak otomatis menggantikan backend atau data di preview tersebut.
