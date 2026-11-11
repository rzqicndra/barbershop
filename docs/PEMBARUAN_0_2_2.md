# Pembaruan 0.2.2

Versi ini memperbaiki pembuatan ID untuk booking pelanggan, walk-in, dan formulir layanan ketika dibuka lewat HTTP pada IP laptop. Browser pada alamat tersebut biasanya tidak menyediakan `crypto.randomUUID()`. Aplikasi sekarang menggunakan fallback UUIDv4 dengan `crypto.getRandomValues()`, yang tersedia pada HTTP lokal. Generator native tetap digunakan ketika tersedia.

Tidak ada perubahan skema database dari 0.2.1 ke 0.2.2. Jika versi sebelumnya lebih lama dari 0.2.1, jalankan [panduan 0.2.1](PEMBARUAN_0_2_1.md) terlebih dahulu.

1. Simpan `.env` dan buat backup database serta foto sesuai README sebelum memperbarui source.
2. Ekstrak ZIP terbaru ke folder sementara, lalu salin source ke folder proyek yang sudah berjalan. Pertahankan `.env`, folder `backups`, dan volume Docker proyek lama. Jangan membuat proyek Compose baru untuk menggantikan data lama.
3. Dari folder proyek yang sama, jalankan:

```powershell
docker compose -f compose.yaml up -d --build app
docker compose -f compose.yaml ps
```

Tidak perlu menjalankan setup, seed, atau membuat akun admin lagi untuk pembaruan dari 0.2.1. Buka ulang halaman agar browser mengambil bundle terbaru.

Untuk server produksi yang sudah memakai overlay, gunakan kedua file Compose dan build app beserta proxy:

```bash
docker compose -f compose.yaml -f compose.production.yaml build app proxy
docker compose -f compose.yaml -f compose.production.yaml up -d app proxy
```

Setelah pembaruan, ikuti [akses HP lokal](AKSES_HP_LOKAL.md) jika ingin mencoba lewat Wi-Fi. Jangan menjalankan `down -v`; perintah tersebut menghapus volume database dan foto.

Verifikasi 0.2.2: 26 pengujian frontend serta TypeScript/Vite build lulus. Smoke test Chromium dengan viewport ponsel dan origin HTTP non-loopback (API memakai fixture) berhasil mengirim booking, mempertahankan request key saat mencoba ulang, mencatat walk-in, serta menambah layanan tanpa `crypto.randomUUID()`. Koneksi Windows ke HP pada Wi-Fi pengguna tetap perlu dicoba langsung; pengujian ini tidak memverifikasi firewall atau router pengguna.
