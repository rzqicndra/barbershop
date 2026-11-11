# Pemeriksaan kesiapan deploy Gilir 0.2.1

Tanggal: 11 November 2026. Sumber pemeriksaan adalah paket 0.2.0 yang sebelumnya dikirim, lalu kode revisi 0.2.1. Pemeriksaan tidak mengakses database laptop atau server operasional client.

**Keputusan: layak dipasang ke staging. Rilis publik menunggu verifikasi server tujuan dan penerimaan pemilik/barber.**

## Temuan dan perbaikan

| Temuan | Dampak | Perbaikan |
| --- | --- | --- |
| Dua booking dengan rentang rencana berbeda dapat sama-sama Sedang dilayani oleh satu barber | Pelayanan nyata tercatat ganda saat kunjungan sebelumnya terlambat | Pemeriksaan transaksi dan unique index satu serving per barber |
| Limiter memakai IP koneksi langsung saja | Pengunjung di belakang reverse proxy berbagi batas percobaan | CIDR proxy eksplisit; X-Real-IP hanya dipercaya dari proxy yang ditetapkan |
| Pemeriksaan login dan pemrosesan foto tidak membatasi pekerjaan bersamaan | Beban memori dapat melonjak | Dua pekerjaan bersamaan per jenis; respons 503 dengan Retry-After saat penuh |
| Scan menemukan advisory Go pada standar HTTP versi 1.27.3 | Versi compiler lama membawa fungsi yang dilaporkan rentan | Go 1.27.4 pada app dan build proxy |
| Dependensi bawaan Caddy memuat versi x/net dan compress yang dilaporkan rentan | Proxy perlu pembaruan selain app | Modul proxy dipatok ke x/net 0.62.0 dan compress 1.20.2, lalu dibangun dan dipindai ulang |
| Tiket pelanggan yang tetap terbuka tidak memeriksa perubahan status secara berkala | Formulir ulasan bisa tetap terkunci sampai tab dimuat ulang/aktif kembali | Poll 30 detik saat terlihat, pembaruan saat aktif, dan pengabaian respons lama |
| Font kecil disematkan sebagai data URL oleh build | Browser memblokirnya karena font-src self | Font selalu diterbitkan sebagai file lokal; kebijakan keamanan tetap ketat |
| Dialog memakai ID judul yang sama dan Escape dapat menutup dialog hapus saat pending | Penamaan aksesibilitas keliru dan konteks simpan hilang | ID unik dan pembatalan native dikendalikan aplikasi |
| Pemrosesan foto membuang EXIF tanpa menerapkan orientasi | JPEG ponsel dapat menyamping | Orientasi diterapkan sebelum resize dan penghapusan metadata |
| Paket lokal belum membawa konfigurasi server yang konkret | Pemasangan HTTPS, healthcheck, volume, log, dan backup bergantung pada langkah manual yang belum tersusun | Compose produksi, Caddy, CLI healthcheck, skrip backup Linux, dan panduan pemasangan |

Penghapusan aman, pembatasan peran, harga/durasi snapshot, transaksi booking, CSRF, tautan privat, dan satu ulasan per kunjungan selesai dipertahankan dan diuji ulang.

## Bukti pengujian

- TypeScript/build frontend dan 24 tes React lulus.
- Tes Go/PostgreSQL dengan race detector lulus, termasuk percobaan memulai dua layanan bersamaan.
- Browser Chromium menguji booking → tiket privat → layanan selesai → ulasan, unggahan/publikasi galeri, cookie Secure/HttpOnly, navigasi staf, dan layout 1440/390 px. Tidak ada error runtime atau pelanggaran CSP pada alur akhir.
- Proxy HTTPS lokal diuji dengan app yang berjalan: dua alamat koneksi pengunjung memperoleh batas login terpisah, dan header IP palsu tidak mengganti identitasnya.
- Dump dipulihkan ke database lain; booking, rating, dan nama barber terbaca. File foto hasil pemulihan mempunyai hash yang sama dengan sumber.
- Konfigurasi gabungan produksi diperiksa dengan Docker Compose CLI. Port app/database tidak dipublikasikan; hanya proxy membuka port 80/443.
- Pemindaian govulncheck pada app dan proxy setelah perbaikan tidak menemukan kerentanan pada fungsi yang digunakan. Masih ada advisory modul OpenPGP yang tidak diimpor/dipanggil oleh aplikasi. Ini bukan klaim bahwa seluruh dependency tree bebas advisory.
- npm audit setelah penambahan alat uji tidak melaporkan advisory.

Rincian perintah dan batas verifikasi ada pada [VALIDASI.md](VALIDASI.md).

## Yang belum dapat dinyatakan selesai

| Bagian | Mengapa masih diperlukan |
| --- | --- |
| Docker Engine pada host target | Lingkungan pengerjaan hanya mempunyai Compose CLI; build dan proses container/volume harus dijalankan di host sebenarnya |
| DNS dan sertifikat domain publik | Domain/server client belum diberikan; HTTPS yang diuji memakai CA lokal khusus pengujian |
| Backup operasional di server tujuan | Pengujian pemulihan memakai data sintetis; lokasi backup terpisah, jadwal, dan retensi belum ditetapkan |
| Isi usaha | Kontak, alamat, foto, harga, durasi, dan penugasan barber perlu dikonfirmasi pemilik; data awal bukan persetujuan operasional |
| Ponsel pemilik/barber dan kondisi jaringan nyata | Viewport browser bukan pengganti uji perangkat fisik |
| Beban produksi dan pemindaian image OS | Belum ada ukuran trafik, hasil uji beban representatif, atau image container yang dapat dipindai di lingkungan ini |

Tidak ada deployment eksternal yang dijalankan pada pemeriksaan ini. Paket revisi dan panduan tersedia untuk pemasangan staging yang dapat ditinjau.
