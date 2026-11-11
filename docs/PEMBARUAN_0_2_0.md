# Pembaruan Gilir 0.2.0

Paket ini memperbarui proyek Go/PostgreSQL yang sudah berjalan. Ada migrasi database baru untuk penghapusan data, konten publik, galeri foto, dan ulasan pelanggan.

## Memasang di laptop yang sudah berisi Gilir

Gunakan folder lama `C:\Users\Rizqi\barbershop`. Nama proyek Compose tetap `gilir`, sehingga database tetap memakai `gilir_pgdata`. Foto unggahan memakai volume baru `gilir_uploads`.

1. Sebelum mengganti kode, jalankan skrip backup lama dari folder proyek. Simpan hasil dump, `.env`, dan salinan kode versi lama. Jika ada perubahan kode buatan sendiri, bandingkan sebelum mengganti.
2. Ekstrak ZIP pembaruan ke folder sementara. Dari folder `gilir-fullstack` hasil ekstrak, salin **backend, frontend, scripts, deploy, docs, Dockerfile, compose.yaml, compose.test.yaml, README.md, dan .dockerignore** ke folder lama. Ganti file kode ketika diminta. Jangan membuat folder `gilir-fullstack` bertingkat di dalam proyek lama.
3. Pertahankan `.env` yang sudah dipakai. Jangan menjalankan ulang pembuatan Admin, mengganti kunci token, atau menghapus volume database.
4. Buka CMD pada folder lama. Jalankan perintah berikut **satu per satu**. Tunggu prompt terminal kembali sebelum menjalankan baris selanjutnya.

```bat
cd /d C:\Users\Rizqi\barbershop
```

```bat
docker compose stop app
```

```bat
docker compose build app
```

```bat
docker compose run --rm --interactive=false tools migrate
```

Lanjutkan setelah muncul `Migrasi selesai.` dan prompt CMD kembali.

```bat
docker compose run --rm --interactive=false tools seed
```

```bat
docker compose up -d app
```

```bat
docker compose ps
```

Jika salah satu perintah gagal, periksa pesan kegagalan sebelum melanjutkan. Untuk log aplikasi, jalankan `docker compose logs --tail=80 app`.

Build menggunakan image lokal `gilir-local:0.2.0`. Migrasi mempertahankan akun, booking, jadwal, dan harga lama. Seed menambahkan layanan contoh yang belum ada dan tidak menimpa isian yang sudah diedit. Menjalankan migrate/seed berulang tidak menggandakan data. Jangan menggunakan `docker compose down -v` karena akan menghapus database dan foto unggahan.

Buka [area staf](http://localhost:8080/admin), lalu gunakan Ctrl+Shift+R jika browser masih menampilkan versi sebelumnya. Sesi staf aktif tetap diarahkan ke dashboard ketika membuka `/`. Untuk melihat beranda pelanggan, gunakan logout atau jendela incognito.

## Yang diatur setelah pembaruan

| Menu Pengaturan | Tindakan |
| --- | --- |
| Profil & jam buka | Masukkan nama/alamat, WhatsApp usaha, Instagram, tautan dan sematan Google Maps |
| Tampilan publik | Ubah logo, foto pembuka, judul, deskripsi, teks galeri, dan tagline |
| Layanan | Sesuaikan kategori, nama, harga, deskripsi, durasi, serta status aktif |
| Barber | Tentukan keahlian, jadwal, dan layanan yang ditugaskan |
| Galeri | Unggah foto pelanggan asli, isi judul/alt/caption, atur urutan, konfirmasi izin, lalu publikasikan |
| FAQ | Tambah, ubah, urutkan, sembunyikan, atau hapus pertanyaan umum |
| Ulasan | Lihat rating yang dikirim pelanggan dari kunjungan selesai |

Maps dan kontak belum diisi dengan akun atau titik lokasi contoh. Peta tersemat muncul setelah kolom sematan diisi. Di Google Maps, buka lokasi sebenarnya → Bagikan → Sematkan peta → Salin HTML. Tempel HTML tersebut pada kolom sematan; website hanya mengambil URL peta resmi. Nomor WhatsApp bisa diawali `08`, `628`, atau `+628`; Instagram bisa berupa `@username` atau URL resminya.

Caption “DI KURSI CUKUR” dan “01 / GADING SERPONG” sudah dihapus. Galeri menggunakan slider yang dapat digeser di ponsel dan tombol navigasi di desktop. Tidak ada autoplay.

## Layanan tambahan

| Layanan | Contoh harga | Contoh durasi | Status awal |
| --- | ---: | ---: | --- |
| Hair coloring | Rp250.000 | 150 menit | Nonaktif |
| Perming | Rp350.000 | 180 menit | Nonaktif |
| Keratin treatment | Rp300.000 | 150 menit | Nonaktif |
| Hair & scalp treatment | Rp100.000 | 60 menit | Nonaktif |

Harga dan durasi di atas adalah data awal yang perlu disesuaikan dengan usaha client. Pilih layanan tambahan pada profil barber yang memiliki keahlian tersebut, kemudian aktifkan layanannya. Ketersediaan jadwal memperhitungkan durasi penuh dan waktu istirahat. Layanan aktif yang belum ditugaskan ke barber mengarahkan pengunjung ke konsultasi kontak.

Admin bisa menambah layanan lain, misalnya bleaching atau styling, tanpa mengubah kode. Durasi layanan mendukung 15–360 menit. Perubahan harga/durasi tidak mengubah nilai booking yang sudah dikonfirmasi.

## Perilaku tombol Hapus

| Objek | Hasil dan batasannya |
| --- | --- |
| Akun staf | Hilang dari daftar, tidak dapat login, seluruh sesinya diakhiri. Profil barber dan riwayat booking dipertahankan |
| Akun yang sedang dipakai | Tidak dapat dihapus |
| Admin aktif terakhir | Dilindungi agar pengelolaan usaha tetap dapat diakses |
| Barber | Hilang dari katalog; akun yang terhubung juga dihapus aksesnya. Ditolak jika masih ada booking Dikonfirmasi atau Sedang dilayani |
| Layanan | Hilang dari katalog. Ditolak jika masih ada booking Dikonfirmasi atau Sedang dilayani |
| Foto galeri atau FAQ | Dihapus dari daftar konten. Foto yang tidak dipakai sebagai logo/foto pembuka/galeri publik menjadi privat |

Penghapusan katalog dan akun memakai penanda internal agar booking lama, harga, nama layanan/barber, dan riwayat tindakan tetap utuh. Ini bukan penghapusan semua data historis. Jika ada penolakan, buka kode booking yang ditampilkan di Agenda dan selesaikan, batalkan dengan alasan, atau catat tidak hadir sesuai keadaan sebenarnya. Booking lama yang masih berstatus Dikonfirmasi juga harus ditangani.

## Alur rating dan ulasan

1. Pelanggan booking dan menyimpan tautan privat kunjungannya.
2. Barber atau Admin mencatat Mulai layanan, lalu Tandai selesai setelah pelayanan benar-benar selesai.
3. Pelanggan membuka kembali tautan privat tersebut. Formulir bintang 1–5 tersedia hanya setelah status Selesai.
4. Pelanggan memilih nama tampilan publik dan menulis ulasan opsional, lalu mengirimkannya.
5. Rating dihitung pada ringkasan publik dan ulasan tampil sebagai kunjungan selesai terverifikasi.

Satu booking mendapat satu ulasan. Status Dikonfirmasi, Sedang dilayani, Dibatalkan, atau Tidak hadir tidak dapat mengirim ulasan, termasuk melalui API langsung. Pengiriman ulang isian yang sama mengembalikan ulasan awal; percobaan membuat ulasan kedua ditolak. Admin tidak dapat mengubah rating pelanggan.

Untuk **walk-in**, Admin membuka detail kunjungan dan menekan **Salin tautan pelanggan**. Bagikan tautan itu hanya kepada pelanggan terkait. Tautan memberi akses ke detail booking, dan berlaku sampai 30 hari setelah waktu selesai terjadwal. Tidak ada pesan WhatsApp yang dikirim otomatis.

## Foto dan backup

Foto menerima JPG, PNG, atau WebP maksimal 8 MB, dimensi maksimal 8.000 piksel per sisi dan 24 megapiksel. Server menyimpan ulang sebagai JPEG dengan sisi panjang maksimal 2.200 piksel. Unggahan pertama bersifat privat; publikasi galeri memerlukan konfirmasi izin pelanggan.

Foto unggahan tersimpan di volume terpisah agar tetap tersedia saat aplikasi dibangun ulang. Jalankan skrip backup terbaru dari folder proyek:

```bat
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\backup.ps1
```

Hasilnya mencakup dump database dan folder foto dengan timestamp yang sama. Backup saat Admin berhenti mengubah foto/publikasi, dan simpan juga `.env`. Memulihkan database saja tidak memulihkan file foto.

Untuk memeriksa bahwa foto sudah tersimpan setelah build ulang: unggah satu foto dari Admin, simpan sebagai galeri publik, buka dari incognito, bangun ulang app, lalu buka kembali foto yang sama. Jangan menghapus volume saat melakukan pemeriksaan ini.

## Pemeriksaan penerimaan di laptop

- Login Admin lama masih berhasil dan booking lama masih terlihat.
- Akun barber yang dihapus tidak dapat memakai sesi lama atau login kembali.
- Hapus layanan/barber dengan booking aktif ditolak dan menampilkan kode booking.
- Maps, Instagram, WhatsApp mengarah ke lokasi dan akun milik client.
- Foto draft hanya terlihat Admin; foto publik tampil pada slider sesuai urutan.
- Booking yang belum selesai tidak dapat mengirim ulasan.
- Booking online maupun walk-in yang selesai dapat mengirim satu ulasan melalui tautan privatnya.
- Konten yang diedit Admin tampil di beranda setelah dimuat ulang.

Build frontend, tes React, serta tes integrasi Go/PostgreSQL dijalankan pada lingkungan pengembangan. Docker Engine dan Windows PowerShell tidak tersedia di lingkungan tersebut; build Compose, volume foto, dan skrip backup perlu diverifikasi melalui langkah di laptop di atas. Rincian ada di [VALIDASI.md](VALIDASI.md).
