# Deploy Gilir ke server Linux

Target konfigurasi: satu lokasi barbershop, satu instance app Go, PostgreSQL, dan satu proxy Caddy dengan koneksi internet langsung. Kode sudah diuji untuk staging; server tujuan belum disiapkan dalam sesi pengerjaan ini.

## Persiapan

- Server Linux dengan Docker Engine dan Docker Compose 2.24.4 atau lebih baru. Verifikasi memakai `docker version` dan `docker compose version`.
- Domain milik usaha dengan DNS A/AAAA menuju server. Isi AAAA hanya jika IPv6 benar-benar berfungsi.
- Port 80 dan 443 dapat dijangkau publik. Batasi akses SSH sesuai pengelolaan server. App dan PostgreSQL tidak dipublikasikan oleh konfigurasi produksi.
- Ruang disk untuk database, foto, build, dan backup. Batas memori contoh adalah 768 MiB untuk app, 768 MiB untuk database, dan 192 MiB untuk proxy; sediakan juga kapasitas OS dan proses build. Sesuaikan setelah mengukur beban sebenarnya.

Gunakan DNS langsung terlebih dahulu. Cloudflare proxy/CDN, load balancer tambahan, atau beberapa instance app membutuhkan penyesuaian IP pengunjung dan limiter bersama. Jangan mengisi TRUSTED_PROXY_CIDRS dengan seluruh internet.

## Konfigurasi

Salin source ke folder server, misalnya `/opt/gilir`. Untuk instalasi baru, buat `.env` menggunakan `node scripts/setup.mjs` atau buat di laptop melalui skrip setup yang disediakan lalu pindahkan dengan akses terbatas. Jangan menimpa `.env` yang sudah terhubung ke database lama.

Tambahkan nilai berikut pada `.env` menggunakan domain dan email yang sebenarnya:

```dotenv
DOMAIN=booking.domain-usaha-anda.com
ACME_EMAIL=email-pengelola@domain-usaha-anda.com
PROXY_SUBNET=172.30.50.0/24
PROXY_IPV4=172.30.50.10
```

DOMAIN hanya hostname, tanpa `https://` atau path. Jika subnet tersebut bentrok dengan jaringan server, ubah PROXY_SUBNET dan PROXY_IPV4 secara bersamaan. Batasi izin `.env`, misalnya `chmod 600 .env`, dan jangan memublikasikannya.

Konfigurasi gabungan menetapkan PUBLIC_ORIGIN ke `https://DOMAIN`, COOKIE_SECURE=true, dan APP_ENV=production. App hanya mempercayai X-Real-IP dari alamat Caddy yang ditetapkan. Caddy mengganti header tersebut dengan IP koneksi pengunjung. Port app dihapus dari host; database tetap di jaringan internal. Log dirotasi dan file foto/sertifikat memakai volume persisten.

Periksa konfigurasi tanpa mencetak rahasia:

```bash
docker compose -f compose.yaml -f compose.production.yaml config --quiet
```

## Instalasi baru dengan database kosong

Jalankan satu per satu. Proxy dibangun memakai Go 1.27.4 dan dependensi yang dipatok pada `deploy/proxy`, sehingga tidak memakai binary proxy lama dari image dasar.

```bash
docker compose -f compose.yaml -f compose.production.yaml build app proxy
docker compose -f compose.yaml -f compose.production.yaml up -d db
docker compose -f compose.yaml -f compose.production.yaml run --rm --interactive=false tools migrate
docker compose -f compose.yaml -f compose.production.yaml run --rm --interactive=false tools seed
docker compose -f compose.yaml -f compose.production.yaml run --rm tools admin:create
docker compose -f compose.yaml -f compose.production.yaml up -d app proxy
docker compose -f compose.yaml -f compose.production.yaml ps
```

`admin:create` interaktif; tunggu selesai sebelum memasukkan perintah selanjutnya. Tidak ada akun bawaan. Jangan menjalankan langkah membuat akun ini untuk mengganti database yang sudah mempunyai Admin.

Caddy meminta dan memperbarui sertifikat domain saat DNS, port, dan penyimpanan sertifikat sudah benar. Periksa log proxy dan endpoint HTTPS. Jangan mengabaikan kesalahan sertifikat pada pemeriksaan domain publik.

```bash
docker compose -f compose.yaml -f compose.production.yaml logs --tail=80 app proxy
curl --fail https://DOMAIN-ANDA/api/v1/health
docker compose -f compose.yaml -f compose.production.yaml exec -T app gilir healthcheck
```

Ganti DOMAIN-ANDA dengan hostname sebenarnya. Hasil health harus `{"status":"ok"}`. Dari jaringan luar, port app 8080 dan database 5432 tidak boleh terbuka.

## Memindahkan data dari laptop

Jangan memakai alur database kosong untuk menimpa data yang akan dipindahkan. Siapkan backup database dan foto dari laptop, serta pertahankan TOKEN_KEY agar tautan pelanggan lama tetap berlaku.

1. Pulihkan dump ke database server yang masih terisolasi dari pelanggan.
2. Pulihkan foto ke volume uploads pada `/app/uploads`, dengan kepemilikan UID/GID 10001:10001.
3. Pastikan akun pemilik database dan `gilir_app` tersedia di server. Dump tidak membawa role global PostgreSQL. Sesuaikan password database server melalui prosedur pengelolaannya; mengganti `.env` saja tidak mengganti password role yang sudah ada.
4. Terapkan migrasi versi ini menggunakan akun pemilik database.
5. Periksa akun, booking lama, rating, foto, dan tautan pelanggan sebelum mengganti DNS atau menerima transaksi baru.

Jika kredensial server berbeda dari laptop, pertahankan TOKEN_KEY lama tetapi gunakan DATABASE_URL/password server yang benar. Uji pada salinan data terlebih dahulu. Proses pemindahan sebenarnya perlu mengikuti keadaan server yang dipilih; jangan menjalankan pg_restore dengan --clean pada database aktif tanpa rencana pemulihan.

## Backup di server

```bash
bash scripts/backup.sh --production
```

Folder hasil lengkap memuat `database.dump`, `uploads/`, dan `SHA256SUMS`. Pemeriksaan checksum:

```bash
cd backups/NAMA-FOLDER-BACKUP
sha256sum -c SHA256SUMS
```

Simpan `.env` secara terpisah dan salin backup ke lokasi lain dengan akses terbatas. Tidak ada jadwal atau layanan backup eksternal yang dibuat otomatis oleh paket ini. Tentukan jadwal operasional dan retensi bersama pemilik.

Pemeriksaan pemulihan menggunakan **database baru** dengan nama berbeda, lalu menghitung booking/ulasan dan membuka foto hasil pemulihan. Pengujian pengembangan telah memulihkan dump dan foto ke database terpisah; lakukan pengujian yang sama terhadap backup dari server tujuan. Checksum tidak membuktikan bahwa seluruh pemulihan aplikasi sudah berhasil.

## Keputusan rilis

Rilis publik setelah semua hasil berikut teramati di server tujuan:

| Pemeriksaan | Hasil yang diperlukan |
| --- | --- |
| Container app | Healthy, tidak berulang restart |
| HTTPS | Sertifikat domain sah dan HTTP dialihkan ke HTTPS |
| Akun | Admin dan tiap Barber mendapat cakupan yang benar |
| Booking | Dua pelanggan tidak dapat mengambil slot yang sama; walk-in memakai kalender yang sama |
| Pelayanan | Satu barber hanya mempunyai satu layanan berjalan |
| Ulasan | Ditolak sebelum selesai dan diterima sekali setelah selesai |
| Foto | Draft privat, publikasi sesuai pilihan, tetap tersedia setelah recreate app |
| Backup | Dump dan foto berhasil dipulihkan ke lingkungan terpisah |
| Konten | Harga, durasi, barber, alamat, Maps, Instagram, WhatsApp, dan foto telah diperiksa pemilik |
| Perangkat | Pemilik dan Barber menyelesaikan alur dari ponsel yang akan dipakai |

Untuk foto AI, gunakan judul/caption yang menjelaskan sebagai inspirasi atau visual AI. Gunakan dokumentasi nyata jika menampilkan hasil pelanggan.

Fitur yang belum termasuk tetap sama: pembayaran online, pesan WhatsApp otomatis, multi-cabang, moderasi ulasan, dan pembersihan file unggahan otomatis. Kalender mempertahankan durasi rencana dan belum menggeser booking berikutnya saat pelayanan terlambat.

Booking tamu langsung berstatus Dikonfirmasi ketika slot tersedia. Nomor WhatsApp diperiksa formatnya dan belum diverifikasi melalui OTP. Pastikan pemilik memahami alur ini saat uji penerimaan.

Referensi konfigurasi: [Docker Compose produksi](https://docs.docker.com/compose/how-tos/production/), [penggabungan Compose](https://docs.docker.com/reference/compose-file/merge/), [Caddy HTTPS](https://caddyserver.com/docs/automatic-https), dan [header proxy Caddy](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy).
