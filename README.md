# Gilir Barbershop

Versi 0.2.2. Untuk proyek yang sudah berjalan, ikuti [panduan pembaruan](docs/PEMBARUAN_0_2_2.md). Pembaruan dari 0.2.1 tidak membutuhkan migrasi baru. Jika masih memakai versi lebih lama, ikuti migrasi 0.2.1 terlebih dahulu. Baca [hasil pemeriksaan kesiapan deploy](docs/AUDIT_DEPLOY.md) dan [panduan server](docs/DEPLOY_PRODUKSI.md) sebelum rilis publik.

Website booking untuk satu barbershop dengan beberapa barber. Frontend React mempertahankan rancangan Gilir yang sudah dibuat. Backend Go menyimpan booking, sesi staf, dan jadwal di PostgreSQL.

Website mencakup booking tanpa akun pelanggan, tautan privat per booking, perubahan dan pembatalan jadwal, agenda staf, walk-in, pengaturan usaha, serta akun Admin dan Barber. Versi ini menambahkan hapus akun/barber/layanan, konten publik dinamis, galeri foto, Maps, Instagram, WhatsApp, dan ulasan dari kunjungan selesai. Pembayaran dilakukan di tempat. Pengiriman WhatsApp otomatis, pembayaran online, multi-cabang, dan Super Admin belum termasuk.

## Mulai dari sini di Windows

Untuk menjalankan website, siapkan:

| Aplikasi | Kegunaan | Kebutuhan |
| --- | --- | --- |
| [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) | Menjalankan Go, PostgreSQL, dan proses build frontend di container Linux | Wajib untuk langkah di bawah |
| [VS Code](https://code.visualstudio.com/) | Membaca dan mengubah proyek | Disarankan |
| [Git](https://git-scm.com/downloads) | Mencatat perubahan kode | Disarankan |
| [Go](https://go.dev/doc/install), minimal versi 1.27.4 | Mengembangkan backend langsung di laptop | Opsional pada tahap pertama |
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
docker compose run --rm --interactive=false tools migrate
docker compose run --rm --interactive=false tools seed
docker compose run --rm tools admin:create
docker compose up -d app
```

Saat `admin:create`, isi email, nama, kata sandi 12–128 karakter, dan konfirmasinya. Kata sandi tidak terlihat ketika diketik. Tidak ada akun atau kata sandi bawaan.

`migrate` membuat atau memperbarui struktur database. `seed` menambahkan tiga layanan awal, tiga profil barber, dan empat contoh layanan tambahan: coloring, perming, keratin, serta hair & scalp treatment. Empat layanan tambahan berstatus nonaktif. Sesuaikan harga dan durasinya, pilih barber yang mampu melayani, lalu aktifkan. Tidak ada pelanggan, booking, ulasan, atau akun barber contoh yang dibuat. Menjalankan ulang `seed` tidak menimpa katalog yang sudah diedit atau mengembalikan data yang sudah dihapus.

Buka:

- Website publik di [http://localhost:8080](http://localhost:8080).
- Login staf di [http://localhost:8080/admin](http://localhost:8080/admin).
- Status backend di [http://localhost:8080/api/v1/health](http://localhost:8080/api/v1/health).

Setelah login sebagai Admin, buka Pengaturan. Sesuaikan profil usaha, harga, jadwal barber, dan hari tutup. Masukkan kontak usaha dan sematan Google Maps pada Profil & jam buka. Ubah logo, foto pembuka, dan teks pada Tampilan publik. Unggah foto pelanggan asli melalui Galeri, lalu konfirmasi izin dan publikasikan. Pada tab Akun staf, buat akun untuk masing-masing profil barber. Barber masuk melalui `/admin` yang sama dan mendapat halaman Jadwal saya berdasarkan akun yang terdaftar.

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
docker compose stop app
docker compose build app
docker compose run --rm --interactive=false tools migrate
docker compose run --rm --interactive=false tools seed
docker compose up -d app
```

Data database berada di volume `pgdata` dan foto unggahan berada di volume `uploads`. Pada proyek Compose bernama `gilir`, nama volumenya `gilir_pgdata` dan `gilir_uploads`. `docker compose down -v` menghapus keduanya. Jangan gunakan opsi `-v` pada data yang ingin disimpan. Mengganti nilai kata sandi di `.env` tidak mengganti kata sandi pada database yang sudah diinisialisasi.

Jika port 8080 sudah dipakai, ubah `APP_PORT=8081` dan `PUBLIC_ORIGIN=http://localhost:8081` di `.env`, lalu jalankan `docker compose up -d app`.

Untuk mencoba dari HP tanpa deploy, ikuti [akses HP melalui Wi-Fi lokal](docs/AKSES_HP_LOKAL.md). Laptop dan HP memakai jaringan yang sama; container app tetap berjalan di laptop.

## Struktur kode

| Lokasi | Isi |
| --- | --- |
| `backend/cmd/gilir/main.go` | Titik masuk server dan perintah administrasi |
| `backend/internal/app/http.go` | Route, respons JSON, middleware, dan penyajian frontend |
| `backend/internal/app/navigation.go` | Middleware pengalihan halaman berdasarkan sesi dan peran |
| `backend/internal/app/auth.go` | Login, cookie sesi, CSRF, logout, dan ganti kata sandi |
| `backend/internal/app/booking.go` | Transaksi booking, idempotensi, perubahan jadwal, dan status |
| `backend/internal/app/calendar.go` | Aturan waktu, jam kerja, istirahat, dan benturan jadwal |
| `backend/internal/app/availability.go` | Pilihan jam dari data server |
| `backend/internal/app/settings.go` | Pengaturan usaha, layanan, barber, izin, dan akun staf |
| `backend/internal/app/deletion.go` | Penghapusan dengan perlindungan booking dan pencabutan sesi |
| `backend/internal/app/content.go` | Kontak, tampilan publik, galeri, dan FAQ |
| `backend/internal/app/media.go` | Validasi unggahan, pemrosesan foto, dan akses media |
| `backend/internal/app/reviews.go` | Ulasan satu per kunjungan selesai dan ringkasan rating |
| `backend/internal/app/migrations/` | Migrasi SQL yang ditanamkan ke binary Go |
| `frontend/src/app.tsx` | Pemeriksaan sesi dan navigasi aplikasi |
| `frontend/src/service-actions.tsx` | Tombol mulai, selesai, dan tidak hadir beserta batas waktunya |
| `frontend/src/public.tsx` | Beranda dan halaman informasi publik |
| `frontend/src/gallery.tsx`, `reviews.tsx` | Slider foto dan ulasan pelanggan |
| `frontend/src/public-settings.tsx` | Editor konten publik dan unggahan foto |
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
- Satu barber hanya boleh memiliki satu kunjungan Sedang dilayani, termasuk ketika pelayanan melewati jam selesai rencana. Catat selesai pada kunjungan sebelumnya sebelum memulai berikutnya.
- Slot untuk `confirmed`, `serving`, dan `completed` tetap dicadangkan pada rentang rencana awal. Pembatalan melepas slot. Sistem belum melakukan perpanjangan otomatis jika pelayanan melewati durasi rencana.
- Perubahan katalog atau kalender yang mengganggu booking mendatang atau sedang dilayani ditolak dengan kode booking yang harus ditangani.
- Tautan privat hanya membuka satu booking dan berakhir 30 hari setelah waktu selesai jadwal terakhir. Tautan yang hilang perlu ditangani melalui Admin.
- Pelanggan hanya dapat memberi satu ulasan setelah status `completed`. Pemeriksaan berlaku di API dan trigger database. Pengiriman ulang isian ulasan yang sama mengembalikan ulasan awal.
- Tiket pelanggan memperbarui status setiap 30 detik saat tab terlihat, serta ketika tab kembali aktif. Formulir ulasan ikut terbuka setelah status Selesai terbaca.
- Admin dapat menyalin tautan privat pelanggan dari detail booking online maupun walk-in. Tautan tersebut memberikan akses ke booking terkait; bagikan hanya kepada pelanggan itu.
- Hapus layanan/barber ditolak jika masih ada booking `confirmed` atau `serving`, termasuk booking lama yang statusnya belum ditangani. Data dihapus dari katalog aktif dengan riwayat tetap tersimpan. Hapus barber juga menghapus akses akun yang terkait.
- Hapus akun mengakhiri seluruh sesinya. Profil barber dan riwayat booking tetap tersimpan. Akun yang sedang dipakai dan Admin aktif terakhir tidak dapat dihapus.

## Konten publik dinamis

| Data | Letak pengaturan Admin |
| --- | --- |
| Nama, alamat, jam buka, tanggal tutup, Maps, Instagram, WhatsApp | Profil & jam buka |
| Logo, judul/deskripsi/foto pembuka, judul/deskripsi galeri, tagline footer | Tampilan publik |
| Nama, kategori, harga, durasi 15–360 menit, deskripsi, status layanan | Layanan |
| Nama, keahlian, jam kerja, istirahat, libur, penugasan layanan | Barber |
| Foto, judul, caption, teks alternatif, urutan, draft/publikasi | Galeri |
| Pertanyaan dan jawaban, urutan, status tampil | FAQ |
| Rating dan ulasan dari pelanggan terverifikasi | Ulasan, hanya baca |

Galeri menampilkan foto dalam slider yang bisa digeser atau dikendalikan dengan tombol. Foto belum tersedia sampai Admin mengunggah dan memublikasikan foto asli. Tidak ada foto pelanggan atau rating buatan. Teks “DI KURSI CUKUR” dan “01 / GADING SERPONG” sudah dihapus dari pembuka.

Kontak awal dikosongkan agar tidak mengarah ke akun yang keliru. Masukkan WhatsApp dan Instagram milik usaha. Untuk peta, salin HTML melalui Google Maps → Bagikan → Sematkan peta, lalu tempel pada kolom sematan. Hanya URL sematan resmi Google Maps yang diterima; HTML tidak dijalankan. Tombol buka Maps menggunakan tautan yang diisi Admin, atau pencarian berdasarkan alamat jika belum diisi.

Unggahan menerima JPG/PNG/WebP maksimal 8 MB dan 24 megapiksel, disimpan ulang sebagai JPEG maksimal sisi panjang 2.200 piksel. Orientasi EXIF foto JPEG diterapkan sebelum metadata gambar dibuang, agar foto ponsel tidak menyamping. Gambar draft hanya dapat dibuka Admin; publik mendapat gambar yang dipakai sebagai logo/foto pembuka atau galeri aktif dengan izin. File yang sudah tidak dipakai tetap berada di penyimpanan privat; pembersihan otomatis belum tersedia.

Ulasan tampil setelah dikirim pelanggan dari tautan privat. Admin tidak dapat menulis ulang rating pelanggan. Ulasan tidak menampilkan nomor WhatsApp, kode booking, atau token. Tidak ada pengiriman tautan otomatis melalui WhatsApp.

Sistem memakai kunci pada satu baris usaha untuk mengurutkan perubahan kalender, ditambah exclusion constraint PostgreSQL sebagai perlindungan benturan slot. Desain ini ditujukan untuk satu lokasi, bukan throughput multi-cabang dalam skala besar.

## Akses dan data

Admin mengelola seluruh operasional. Barber hanya dapat membaca booking yang ditugaskan kepadanya dan mengubah status pelayanannya. Nomor pelanggan dan riwayat internal tidak dikirim dalam respons akun barber. Tidak ada pemilih peran pada login atau role yang dipercaya dari browser.

Selama sesi staf masih aktif, membuka `/` atau menghapus path `/admin` mengarahkan Admin ke `/admin/agenda` dan Barber ke `/admin/jadwal`. Beranda pelanggan dapat dibuka setelah logout atau dari profil browser lain. Tautan logo di sidebar staf menuju dashboard perannya.

Sesi menggunakan cookie HttpOnly, SameSite Lax, serta batas 12 jam di server. Semua perubahan staf memerlukan token CSRF. Menonaktifkan akun, menonaktifkan profil barber, reset kata sandi, atau ganti kata sandi mengakhiri sesi terkait. Hash kata sandi memakai Argon2id.

Pada Compose, akun `gilir_app` dipakai saat website berjalan dan tidak mendapat hak membuat skema. Akun pemilik database digunakan hanya oleh container `tools` untuk migrasi dan pembuatan Admin. Database tidak membuka port ke host pada konfigurasi utama.

Tautan privat memakai fragment URL dan dikirim ke API melalui header Authorization. Data booking dan sesi tidak disimpan di localStorage. Formulir menyimpan isian sementara di memori tab. Setelah tab ditutup atau dimuat ulang, isian yang belum terkirim tidak dipulihkan.

## Pengujian

Tes integrasi memakai database terpisah, tanpa mengakses data operasional.

```powershell
docker compose -f compose.test.yaml up --build --abort-on-container-exit --exit-code-from test
docker compose -f compose.test.yaml down
```

Untuk menguji antarmuka dengan Node 24, jalankan `npm ci` lalu `npm test` dari folder `frontend`. Sebanyak 24 tes antarmuka mencakup navigasi staf, kedaluwarsa sesi, aktivasi tombol sesuai waktu, formulir ulasan, pembaruan tiket, konten dinamis, penghapusan akun, dialog, dan unggahan multipart. Tes browser dan HTTPS/restore tambahan dijelaskan pada [VALIDASI.md](docs/VALIDASI.md).

Perintah tes database harus berakhir dengan exit code 0. Tes menguji benturan slot bersamaan, idempotensi, snapshot harga dan durasi, perubahan jadwal yang gagal, pembatasan peran, CSRF, status layanan, dan izin barber.

## Backup dan pemulihan

Dari folder proyek pada PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\backup.ps1
```

Salinan berformat PostgreSQL custom dump dan folder foto `gilir-TANGGAL-JAM-uploads` akan berada di folder `backups`. Simpan keduanya di lokasi terpisah bersama `.env` dan versi kode. Batasi akses karena dump berisi data pelanggan dan hash akun. Untuk pasangan backup yang konsisten, jadwalkan saat Admin berhenti mengunggah atau mengubah publikasi foto.

Di server Linux dengan konfigurasi produksi, gunakan `bash scripts/backup.sh --production`. Skrip menghasilkan satu folder berisi dump, foto, dan SHA256SUMS, lalu menandainya lengkap setelah seluruh salinan berhasil. Folder `.partial` menandakan backup gagal dan tidak boleh dianggap lengkap.

Uji pemulihan ke database baru, bukan langsung ke database operasional. Ganti nama file contoh berikut dengan hasil backup yang benar.

```powershell
docker compose cp .\backups\gilir-20261111-120000.dump db:/tmp/restore-check.dump
docker compose exec db createdb -U gilir gilir_restore_check
docker compose exec db pg_restore -U gilir -d gilir_restore_check --exit-on-error /tmp/restore-check.dump
docker compose exec db psql -U gilir -d gilir_restore_check -c "SELECT count(*) FROM bookings;"
```

Periksa booking dan riwayat pada hasil pemulihan sebelum menetapkan prosedur produksi. Database `gilir_restore_check` berdiri sendiri dan tidak mengubah koneksi website. Jangan menggunakan `--clean` pada database operasional tanpa rencana pemulihan yang sudah disepakati.

Jika berpindah server, pulihkan folder foto ke `/app/uploads` pada volume `uploads` dan pertahankan kepemilikan UID/GID `10001:10001`. Memulihkan dump saja tidak mengembalikan file foto. Panduan pembaruan memuat pemeriksaan penyimpanan.

## Sebelum digunakan publik

Kode telah diuji untuk pemasangan ke staging. Deployment ke server dan domain client belum dilakukan, sehingga status produksi harus ditetapkan setelah pemeriksaan di server tujuan.

Gunakan server Linux yang dapat menjalankan container Go dan PostgreSQL. Paket menyediakan `compose.production.yaml` dan Caddy untuk HTTPS. Konfigurasi produksi menutup port app/database dari host dan menetapkan cookie Secure, origin domain, IP proxy yang dipercaya, batas memori, dan rotasi log. Proxy dibangun dari modul terpisah pada `deploy/proxy`, memakai Go yang sudah diperbarui dan dependensi yang dipatok. Ikuti [DEPLOY_PRODUKSI.md](docs/DEPLOY_PRODUKSI.md), termasuk persiapan DNS dan pemulihan data saat berpindah dari laptop.

Pembatasan percobaan masuk dan pembuatan booking berada di memori satu proses. Tanpa konfigurasi proxy, alamat koneksi langsung menjadi acuan. `TRUSTED_PROXY_CIDRS` mengizinkan pembacaan `X-Real-IP` hanya dari proxy tertentu; Caddy mengganti header tersebut dengan alamat koneksi pengunjung. Header yang dikirim pengunjung langsung dan rantai `X-Forwarded-For` tidak dipercaya. Konfigurasi ini mendukung satu app di belakang satu Caddy yang menerima koneksi langsung dari internet. Untuk beberapa app atau CDN/proxy tambahan, sesuaikan mekanisme IP dan pembatasan bersama sebelum rilis.

Sebelum rilis, lengkapi data usaha dan kontak bantuan, uji operasional bersama pemilik dan barber, pastikan backup bisa dipulihkan, serta tetapkan kebijakan penyimpanan data pelanggan. Tidak ada pengiriman WhatsApp otomatis pada versi ini. Nama dan nomor yang diisikan dipakai sebagai data booking internal.

Preview prototipe sebelumnya berjalan pada layanan berbeda. Paket Go/PostgreSQL ini tidak otomatis menggantikan backend atau data di preview tersebut.
