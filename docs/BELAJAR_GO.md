# Mengenal Go lewat proyek Gilir

Mulai dengan menjalankan versi Docker sesuai README. Setelah itu, pasang Go 1.27.x dari [installer resmi](https://go.dev/doc/install) dan Node 24 LTS dari [Node.js](https://nodejs.org/en/download). Buka terminal baru setelah instalasi.

```powershell
go version
node --version
npm --version
```

Di VS Code, extension Go dari penerbit Go Team membantu membaca tipe, melompat ke fungsi, dan memformat kode.

## Istilah yang akan sering dipakai

| Istilah | Makna pada proyek ini |
| --- | --- |
| `go.mod` | Identitas modul dan versi dependensi backend |
| `go.sum` | Checksum dependensi yang didownload |
| `package main` | Paket yang dapat menjadi program executable |
| `func main()` | Titik awal ketika binary dijalankan |
| `internal/app` | Kode aplikasi yang hanya dapat diimpor dari dalam proyek backend |
| `struct` | Bentuk data, misalnya Service atau Booking |
| `context.Context` | Membawa pembatalan dan batas waktu pekerjaan, terutama permintaan HTTP dan query |
| `error` | Hasil kegagalan yang harus diperiksa oleh pemanggil fungsi |
| `pgxpool` | Sekumpulan koneksi PostgreSQL yang digunakan ulang |
| `transaction` | Beberapa perubahan database yang berhasil bersama atau dibatalkan bersama |
| `middleware` | Pemeriksaan yang berjalan sebelum endpoint, misalnya verifikasi sesi dan CSRF |

Go mengompilasi kode menjadi binary. Docker menjalankan binary tersebut untuk menerima permintaan API dan menyajikan hasil build frontend. React mengelola tampilan di browser, sedangkan Go memutuskan apakah suatu booking atau perubahan diizinkan.

Saat pelanggan menekan Konfirmasi, `frontend/src/booking.tsx` mengirim JSON melalui `api.ts`. Chi mencocokkan URL dengan endpoint `createPublicBooking` di `booking.go`. Backend memvalidasi input, mengambil kunci kalender, memeriksa harga dan slot terbaru, lalu menyimpan booking beserta riwayat dalam satu transaksi. Respons baru dikirim setelah commit berhasil.

## Pengembangan langsung dengan perubahan cepat

Langkah ini menggunakan database Compose yang sama dengan instalasi lokal. Hentikan container aplikasi agar port 8080 tersedia. Container database tetap berjalan.

```powershell
docker compose stop app
docker compose -f compose.yaml -f compose.dev.yaml up -d db
docker compose run --rm tools migrate
```

Buka terminal pertama dari folder utama proyek.

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\dev-backend.ps1
```

Skrip membaca rahasia dari `.env` lalu menjalankan `go run ./cmd/gilir serve`. Ini adalah proses development, bukan hot reload otomatis. Jika mengubah kode Go, hentikan dengan Ctrl+C lalu jalankan lagi.

Buka terminal kedua.

```powershell
cd frontend
npm ci
npm run dev
```

Buka [http://localhost:5173](http://localhost:5173). Vite meneruskan `/api` ke Go pada port 8080. Gunakan `localhost` secara konsisten agar origin dan cookie sesuai. Tampilan berubah otomatis saat file frontend disimpan.

Jangan menjalankan perintah `migrate` dengan akun database `gilir_app`, karena akun itu memang tidak mempunyai hak membuat tabel. Migrasi lokal dilakukan melalui container `tools` pada langkah sebelumnya.

## Perintah Go dasar

Jalankan dari folder `backend`.

```powershell
# Mengunduh dependensi sesuai go.mod
go mod download

# Memformat kode Go
go fmt ./...

# Pemeriksaan statis
go vet ./...

# Tes unit. Tes database dilewati jika TEST_DATABASE_URL belum diisi.
go test ./...

# Membuat binary Windows
go build -o bin/gilir.exe ./cmd/gilir
```

Untuk pengujian integrasi yang lengkap, gunakan perintah `compose.test.yaml` di README. Database uji harus memiliki nama berakhiran `_test`. Setiap tes membuat skema sementara yang dibersihkan setelah selesai.

## Contoh titik perubahan

- Mengubah warna atau spasi tampilan dilakukan di `frontend/src/styles.css`.
- Mengubah teks tampilan dilakukan di komponen TSX terkait.
- Menambah aturan jadwal dilakukan di `calendar.go`, kemudian tambahkan tes untuk batas aturan tersebut.
- Menambah kolom database dilakukan dengan file migrasi baru, misalnya `002_...sql`. Jangan mengubah migrasi yang sudah digunakan oleh database lain.
- Menambah endpoint dilakukan di `http.go`, disertai fungsi handler dan pemeriksaan hak akses yang sesuai.

Mulai dari satu perubahan kecil, jalankan ulang aplikasi, dan periksa hasilnya. Untuk aturan booking, jalankan tes integrasi sebelum perubahan dipakai oleh pemilik usaha.
