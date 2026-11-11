# Pembaruan Gilir 0.2.1

Revisi hasil pemeriksaan kesiapan deploy. Versi ini memperbarui Go dan dependensi proxy, membatasi satu layanan berjalan per barber, memperbaiki pembaruan tiket pelanggan, orientasi foto ponsel, serta menyediakan konfigurasi produksi.

## Update proyek lama di Windows

Gunakan folder lama `C:\Users\Rizqi\barbershop` dan pertahankan `.env`, `gilir_pgdata`, serta `gilir_uploads`. Tidak perlu membuat ulang akun Admin.

1. Jalankan skrip backup lama dan simpan hasilnya bersama salinan kode serta `.env`.
2. Sebelum menghentikan versi lama, periksa kunjungan ganda yang masih Sedang dilayani:

```bat
docker compose exec -T db psql -U gilir -d gilir -c "SELECT barber_id,string_agg(code,', ') AS kunjungan FROM bookings WHERE status='serving' GROUP BY barber_id HAVING count(*)>1;"
```

Jika ada hasil, buka Agenda pada versi lama dan catat selesai pada kunjungan yang memang sudah selesai. Migrasi tidak akan mengubah riwayat itu secara otomatis. Lanjutkan setelah query tidak menghasilkan baris.

3. Ekstrak ZIP ke folder sementara. Salin `backend`, `frontend`, `deploy`, `scripts`, `docs`, `Dockerfile`, `compose.yaml`, `compose.production.yaml`, `compose.test.yaml`, `.dockerignore`, dan `README.md` dari folder `gilir-fullstack` hasil ekstrak ke folder proyek lama. Bandingkan jika ada perubahan kode sendiri.
4. Jalankan setiap perintah berikut secara terpisah dari CMD. Tunggu prompt kembali sebelum melanjutkan.

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

```bat
docker compose run --rm --interactive=false tools seed
```

```bat
docker compose up -d app
```

```bat
docker compose ps
```

Image menggunakan `gilir-local:0.2.1`. Status app akan menjadi healthy setelah endpoint HTTP dan database dapat diakses. Jika gagal, periksa `docker compose logs --tail=80 app`. Jangan melanjutkan saat migrasi gagal dan jangan menghapus volume.

Lakukan Ctrl+Shift+R pada browser. Beranda pelanggan tetap memerlukan logout atau incognito saat sesi staf masih aktif. Pengaturan galeri, kontak, dan layanan tambahan tetap mengikuti [panduan 0.2.0](PEMBARUAN_0_2_0.md).

## Perubahan perilaku

| Bagian | Hasil |
| --- | --- |
| Mulai layanan | Ditolak jika barber yang sama masih melayani kunjungan lain, termasuk kunjungan yang melewati waktu selesai rencana |
| Tiket pelanggan | Memeriksa pembaruan setiap 30 detik pada tab yang terlihat dan saat tab kembali aktif |
| Rating | Formulir tersedia setelah status Selesai terbaca; nama dan ulasan yang dikirim tampil publik |
| Foto ponsel | Orientasi EXIF JPEG diterapkan sebelum metadata dihapus |
| Font | Dimuat sebagai file lokal agar sesuai kebijakan keamanan halaman |
| Dialog hapus | Escape tidak menutup dialog saat penghapusan sedang disimpan |
| Login/foto | Pemrosesan bersamaan dibatasi; keadaan penuh memberi petunjuk mencoba kembali |
| Server publik | Tersedia konfigurasi HTTPS, proxy terpercaya, healthcheck, batas memori, dan rotasi log |

Migrasi baru menambah indeks yang membatasi satu baris `serving` per barber. Setelah migrasi, jangan menjalankan image versi lama pada skema baru. Jika perlu rollback, gunakan backup yang sesuai pada lingkungan pemulihan dan periksa hasilnya terlebih dahulu.

## Menuju server publik

Gunakan [DEPLOY_PRODUKSI.md](DEPLOY_PRODUKSI.md). Konfigurasi produksi tidak dipakai untuk membuka localhost biasa. Domain, DNS, sertifikat publik, backup di server tujuan, dan uji pemilik/barber masih harus diperiksa sebelum menerima pelanggan.
