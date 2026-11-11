# Catatan validasi versi 0.2.1

Diperiksa pada 11 November 2026. Pemeriksaan dilakukan dengan Node 24.22.1, Go 1.27.4, PostgreSQL 17.7, Chrome Headless Shell 157, dan proxy Caddy yang dibangun dari modul 2.11.9 dengan dependency patch pada modul terpisah.

## Hasil yang dijalankan

| Pemeriksaan | Hasil |
| --- | --- |
| TypeScript dan Vite production build | Lolos |
| 24 tes komponen React | Lolos |
| `go vet ./...` dan build binary Go | Lolos |
| Seluruh tes unit dan integrasi PostgreSQL dengan `go test -race -count=1 -v ./...` | Lolos |
| Tes terkait akun/penghapusan/otorisasi diulang setelah penguncian pembuatan akun diperbaiki | Lolos |
| Migrasi dari skema 001 yang berisi profil, akun, layanan yang diedit, dan booking lama | Lolos; nilai lama dipertahankan, nama barber diisi pada snapshot |
| Migrasi diulang dan seed setelah upgrade | Tidak menggandakan migrasi atau menimpa layanan lama |
| Akun database terbatas menulis ulasan, media, dan galeri setelah migrasi | Lolos memakai hak DML dari default privileges |
| Akun database terbatas membuat tabel | Ditolak |
| Hapus akun yang sedang dipakai | Ditolak |
| Hapus akun barber lalu memakai sesi lama atau login kembali | Ditolak |
| Buat akun pengganti dengan email dan profil barber yang sama | Identitas baru; sesi akun lama tetap ditolak |
| Hapus layanan/barber dengan booking Dikonfirmasi atau Sedang dilayani | Ditolak dengan `BOOKINGS_AFFECTED` |
| Hapus barber setelah booking selesai | Hilang dari katalog dan akses akun terkait dicabut |
| Baca booking lama setelah barber/layanan dihapus | Nama, harga, durasi, dan riwayat tetap tersedia |
| Seed setelah data katalog dihapus | Tidak mengembalikan data yang dihapus |
| Ulasan sebelum status Selesai | Ditolak API; insert langsung juga ditolak trigger database |
| Dua ulasan identik pada booking yang sama secara bersamaan | Satu dibuat, satu mendapat ulasan yang sama; tidak menggandakan rating |
| Ulasan kedua dengan isian berbeda, token tidak sah, rating tidak valid, atau tautan kedaluwarsa | Ditolak sesuai kode API |
| Walk-in mendapatkan tautan privat dan mengirim ulasan setelah selesai | Lolos |
| Ringkasan dan ulasan publik | Rating terhitung; nomor, token, dan ID booking tidak dikirim |
| Foto JPG valid, SVG, sesi tamu, dan akun Barber | JPG Admin diterima; SVG, tamu, dan Barber ditolak |
| Foto draft | 404 bagi publik, dapat dipratinjau Admin |
| Publikasi galeri tanpa izin pelanggan | Ditolak |
| Publikasi, sembunyikan, cabut izin, dan hapus foto galeri | Mengubah akses publik sesuai keadaan yang disimpan |
| Header foto | JPEG, no-store, dan nosniff |
| Simpan kontak dan sematan Maps | Instagram/WhatsApp dinormalisasi; URL sematan di luar Google ditolak |
| Tambah/hapus FAQ dan pembatasan Barber | Lolos |
| Formulir ulasan pada status belum selesai, sedang dilayani, batal, dan tidak hadir | Tidak tersedia untuk mengirim |
| Formulir ulasan selesai | Mengirim token privat, bintang, nama tampilan, dan komentar yang dipilih pelanggan |
| Ulasan sudah tersimpan atau dikirim tab lain | Tidak menawarkan ulasan kedua; detail dimuat ulang saat konflik |
| Konten publik dinamis pada tes React | Judul, branding, foto, Maps, Instagram, WhatsApp, FAQ, dan kategori berasal dari data |
| Caption pembuka yang diminta dihapus | Tidak dirender |
| Layanan aktif yang belum ditugaskan ke barber | Mengarah ke konsultasi kontak |
| Konfirmasi penghapusan ditolak karena booking aktif | Dialog tetap terbuka dengan pesan penolakan |
| Unggahan antarmuka | Memakai FormData multipart dan URL hasil server |
| Dua permintaan Mulai layanan untuk barber yang sama bersamaan | Satu berhasil, satu ditolak; database juga melarang dua serving |
| Proxy dengan IP tetap dan header X-Real-IP | Header dari peer tidak terpercaya diabaikan; pengunjung berbeda mempunyai limiter berbeda |
| Pekerjaan login/foto bersamaan melebihi kapasitas | 503 dengan Retry-After; endpoint kesehatan tetap merespons |
| Poll tiket pelanggan | Status Selesai membuka formulir ulasan tanpa membuka ulang tiket |
| Dialog pending dan dialog bertumpuk | Escape tidak menutup saat penghapusan pending; judul aksesibilitas unik |
| Orientasi JPEG ponsel | Dimensi keluaran mengikuti EXIF dan metadata dibuang |
| Chromium desktop 1440 px dan viewport mobile 390 px | Booking, status, ulasan, galeri, menu, dan tidak ada overflow horizontal diuji |
| Error runtime dan CSP pada alur browser akhir | Tidak ditemukan |
| HTTPS Caddy lokal → Go → PostgreSQL | Login Secure/HttpOnly dan operasi booking/foto/ulasan lolos |
| Pemalsuan IP melalui Caddy | Tidak melewati limiter; koneksi pengunjung lain tetap mendapat kuota sendiri |
| Dump/restore database dan salinan file foto | Booking/ulasan terbaca pada database lain; hash foto sama |
| Compose CLI produksi | Valid; app/database tidak memublikasikan port dan hanya proxy membuka 80/443 |
| govulncheck source app dan proxy | Tidak menemukan kerentanan pada fungsi yang dipakai; advisory modul OpenPGP yang tidak dipanggil tetap dilaporkan |
| npm audit | Tidak melaporkan advisory |

Tes lama tetap mencakup benturan dua booking pada slot yang sama, exclusion constraint PostgreSQL, idempotensi, snapshot harga/durasi, reschedule yang gagal, batas waktu pelanggan, izin barber, CSRF, Origin, cakupan agenda tiap barber, dan perubahan status.

Tes navigasi mencakup Admin/Barber yang membuka root, sesi berakhir, logout, serta endpoint API yang tetap mengembalikan JSON. Tes waktu antarmuka memeriksa tombol Mulai layanan sebelum dan ketika jam booking tercapai.

## Cara mengulang

Dari folder proyek pada laptop dengan Docker:

```powershell
docker compose -f compose.test.yaml up --build --abort-on-container-exit --exit-code-from test
docker compose -f compose.test.yaml down
```

Tes integrasi hanya menerima database dengan nama berakhiran `_test`, lalu membuat dan menghapus skema acak di database itu. Tidak menggunakan database operasional.

Untuk frontend, jalankan `npm ci`, `npm test`, dan `npm run build` dari folder `frontend`. Pemeriksaan Go dapat dijalankan dengan `go vet ./...` dan `go test -race -count=1 ./...` dari `backend`, dengan `TEST_DATABASE_URL` yang menunjuk database pengujian. Tanpa variabel tersebut tes integrasi akan dilewati.

Tes penerimaan browser bersifat opsional dan terpisah dari tes biasa. Pasang Chromium/Chrome Headless Shell lokal, lakukan npm ci dan build frontend, lalu jalankan dari backend dengan database *_test:

```bash
E2E_BROWSER=1 CHROMIUM_EXECUTABLE=/path/ke/chrome-headless-shell TEST_DATABASE_URL='postgres://USER:PASSWORD@127.0.0.1:5432/gilir_test?sslmode=disable' go test -race -count=1 -run '^TestBrowserAcceptance$' -v ./internal/app
```

Harness membuat schema fixture, memakai HTTPS lokal bersertifikat test, dan menghapus fixture setelah selesai. Route pengatur waktu hanya berada di berkas _test.go, tidak dikompilasi ke binary produksi. Uji browser mempercayai sertifikat lokal khusus test; tidak membuktikan sertifikat domain publik sudah benar.

Tes HTTPS proxy dan pemulihan memerlukan binary proxy yang sudah dibangun dari `deploy/proxy` dan direktori alat PostgreSQL. Jalankan dari backend:

```bash
E2E_PROXY=/path/ke/caddy PG_TOOLS_DIR=/path/ke/postgresql/bin TEST_DATABASE_URL='postgres://USER:PASSWORD@127.0.0.1:5432/gilir_test?sslmode=disable' go test -race -count=1 -run '^TestDeploymentRoundTrip$' -v ./internal/app
```

Tes ini membuat database pemulihan acak dengan akhiran _test dan menghapusnya setelah pemeriksaan. Role database pengujian memerlukan hak membuat database serta extension btree_gist. Gunakan lingkungan pengujian terisolasi.

## Batas verifikasi

Docker Engine tidak tersedia di lingkungan pengerjaan. Image Docker, pemasangan volume `uploads`, pembatasan container, dan Compose versi 0.2.1 belum dijalankan sebagai container di sini. Build frontend/binary Go/proxy serta PostgreSQL dijalankan langsung. Konfigurasi gabungan Compose produksi diuji menggunakan CLI 5.7.1 dan Caddyfile diuji memakai binary proxy.

PowerShell tidak tersedia, sehingga skrip backup database dan foto belum dieksekusi pada Windows. Skrip Linux lulus pemeriksaan sintaks, tetapi perintah Docker di dalamnya belum dieksekusi karena Engine tidak tersedia. Operasi dump/restore dan pencocokan file foto diuji langsung melalui alat PostgreSQL pada fixture HTTPS. Jalankan [PEMBARUAN_0_2_1.md](PEMBARUAN_0_2_1.md) dan verifikasi persistensi volume serta backup di host tujuan.

Browser pada revisi ini menguji data sintetis pada fixture terisolasi. Pemeriksaan viewport bukan uji perangkat fisik. Tautan Maps/Instagram/WhatsApp, titik lokasi usaha sebenarnya, foto pelanggan asli, dan kebiasaan operasional tetap perlu diperiksa bersama client pada perangkat yang akan dipakai.

Galeri dan kontak awal tidak diisi data pelanggan atau akun sosial buatan. Empat layanan tambahan dibuat nonaktif dengan harga/durasi contoh untuk dikonfigurasi pemilik. Belum ada pembersihan otomatis file unggahan yang tidak dipakai, moderasi ulasan, atau pesan WhatsApp otomatis.

Deployment domain publik, HTTPS produksi, pengujian beban, peninjauan keamanan independen, dan uji penerimaan pemilik/barber belum dilakukan. Paket ini adalah pembaruan proyek lokal yang telah diuji pada lingkup di atas.
