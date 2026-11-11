# Pembaruan 0.1.1

Revisi ini menambahkan pengalihan berdasarkan sesi, memperjelas tombol pelayanan, dan menerangkan tugas operasional setiap peran. Struktur database tidak berubah.

## Memasang ke proyek yang sudah berjalan

Gunakan folder proyek lama `C:\Users\Rizqi\barbershop` agar nama proyek Compose dan volume database tetap sama.

1. Ekstrak ZIP revisi ke folder sementara.
2. Dari folder `gilir-fullstack` hasil ekstrak, salin folder `backend`, folder `frontend`, dan file `compose.yaml` ke `C:\Users\Rizqi\barbershop`. Ganti file kode lama ketika diminta. Jika kamu sudah mengubah kode sendiri, bandingkan perubahannya sebelum mengganti.
3. Pertahankan `.env` yang sudah dipakai dan volume database `gilir_pgdata`. ZIP tidak membawa `.env` atau data database.
4. Buka CMD atau PowerShell di `C:\Users\Rizqi\barbershop`.
5. Jalankan setiap perintah berikut secara terpisah. Tunggu terminal kembali siap sebelum melanjutkan.

```powershell
docker compose build app
docker compose up -d --force-recreate app
docker compose ps
```

Pada daftar container, `app` harus berjalan dan memetakan port 8080. Revisi menggunakan image lokal `gilir-local:0.1.1`. Build pertama untuk revisi ini menggunakan ulang layer Docker yang masih cocok.

Tidak perlu menjalankan `seed`, `admin:create`, atau menghapus volume. Akun, sesi yang masih berlaku, data katalog, dan booking memakai database lama. Jangan menggunakan `docker compose down -v` karena opsi tersebut menghapus volume data.

Buka ulang [login staf](http://localhost:8080/admin). Jika browser masih menampilkan antarmuka lama, gunakan Ctrl+Shift+R. Jika aplikasi terus memulai ulang, periksa `docker compose logs --tail=80 app`.

## Pengalihan sesi

| Kondisi | Hasil |
| --- | --- |
| Admin sudah login lalu membuka `/` | Kembali ke `/admin/agenda` |
| Barber sudah login lalu membuka `/` | Kembali ke `/admin/jadwal` |
| Barber membuka pengaturan Admin | Kembali ke jadwalnya |
| Sesi berakhir lalu membuka halaman staf | Menuju `/admin` untuk login |
| Pengunjung tanpa sesi staf membuka `/` | Beranda pelanggan |

Pemeriksaan berlaku pada middleware Go dan navigasi frontend. Untuk melihat halaman pelanggan saat masih memakai akun staf, logout atau gunakan profil browser lain. Logo di sidebar staf menuju dashboard sesuai perannya.

## Mengapa Admin bisa mengubah jadwal?

Admin menangani perubahan atas permintaan pelanggan, barber berhalangan, atau kebutuhan operasional lain. Perubahan hanya berlaku untuk booking terkonfirmasi yang belum dimulai. Alasan wajib diisi dan dicatat bersama identitas pengubah pada riwayat booking.

Jadwal pengganti tetap diperiksa terhadap jam kerja, istirahat, libur, dan booking lain. Jika waktu baru bentrok, jadwal lama tetap berlaku. Harga dan durasi booking yang sudah dikonfirmasi tetap dipakai.

Akun Barber tidak mendapat endpoint untuk menjadwal ulang atau membatalkan booking.

## Mengapa Mulai layanan sebelumnya tidak muncul?

Versi awal menyembunyikan tombol sebelum jam mulai booking. Untuk jadwal 12.30 WIB, tombol belum tersedia jika waktu server masih 11.47 WIB.

Pada revisi ini, tombol tetap terlihat dengan keadaan nonaktif, disertai tanggal, jam mulai, dan waktu server. Tombol aktif otomatis saat jam booking tercapai. Backend tetap memeriksa waktu dan version booking ketika tombol ditekan.

Alur pelayanan:

1. Buka detail kunjungan melalui tombol panah pada baris pelanggan.
2. Tekan Mulai layanan pada atau setelah waktu terjadwal.
3. Status menjadi Sedang dilayani.
4. Setelah pelayanan benar-benar selesai, tekan Tandai selesai.
5. Jika pelanggan tidak datang, Tandai tidak hadir tersedia setelah rentang waktu booking berakhir.

## Fungsi akun Barber

| Peran | Cakupan dan tindakan |
| --- | --- |
| Admin | Seluruh agenda, walk-in, pengaturan usaha, akun staf, serta penjadwalan ulang dan pembatalan dengan alasan |
| Barber | Jadwal yang ditugaskan kepadanya, detail layanan, mulai layanan, selesai layanan, dan pencatatan tidak hadir |

Jika tiga booking seluruhnya ditugaskan ke Raka, ketiganya akan tampak pada agenda Admin dan jadwal Raka. Akun barber lain tidak dapat membaca atau mengubah booking tersebut. Nomor pelanggan dan riwayat internal tidak dikirim pada respons akun Barber.

Status yang dicatat Barber tersimpan dalam database yang sama dan dapat dibaca Admin. Agenda memperbarui data setiap 30 detik dan ketika jendela kembali aktif. Ini bukan pemisahan atau penyalinan data booking per dashboard.

## Verifikasi

Sembilan tes antarmuka, tes integrasi Go/PostgreSQL dengan race detector, pemeriksaan Go, dan build frontend sudah lolos. Tes mencakup pengalihan berdasarkan peran, sesi berakhir, aktivasi tombol tepat pada waktu jadwal, serta perubahan status oleh Barber yang tercermin di detail Admin. Docker Compose revisi ini perlu dijalankan di laptop untuk verifikasi akhir.
