# Membuka Gilir dari HP tanpa deploy

Laptop menjalankan aplikasi dan database; HP membuka alamat IP laptop. Gunakan paket 0.2.2 atau lebih baru agar booking, walk-in, dan formulir layanan juga bekerja pada HTTP melalui IP lokal.

## 1. Hubungkan jaringan

Hubungkan laptop dan HP ke Wi-Fi rumah yang sama. Hindari guest Wi-Fi yang membatasi komunikasi antarperangkat. Laptop harus tetap menyala dan Docker Desktop berjalan.

Di PowerShell atau CMD **laptop Windows**, jalankan:

```powershell
ipconfig
```

Cari `IPv4 Address` pada adaptor Wi-Fi yang sedang tersambung. Contoh: `192.168.1.10`. Jangan memilih IP adaptor Docker, WSL, atau vEthernet. Jika laptop memakai kabel LAN, gunakan IPv4 adaptor Ethernet yang terhubung ke router Wi-Fi tersebut.

## 2. Atur konfigurasi proyek

Buka folder yang memuat `compose.yaml`. Contoh lokasi proyek:

```powershell
cd C:\Users\Rizqi\barbershop
notepad .env
```

Sesuaikan baris berikut, mengganti IP contoh dengan IPv4 laptop Anda:

```dotenv
BIND_ADDRESS=0.0.0.0
APP_PORT=8080
PUBLIC_ORIGIN=http://192.168.1.10:8080
COOKIE_SECURE=false
```

Edit baris yang sudah ada, jangan menambahkan duplikat. Pertahankan DB_PASSWORD, APP_DB_PASSWORD, TOKEN_KEY, dan nilai rahasia lainnya. Untuk pengujian ini gunakan Compose lokal, tanpa `compose.production.yaml`.

`BIND_ADDRESS` membuat port app dapat dijangkau perangkat lain. `PUBLIC_ORIGIN` harus sama dengan alamat yang dibuka dari HP agar login dan pengiriman booking diterima. `COOKIE_SECURE=false` digunakan karena pengujian lokal ini memakai HTTP; produksi tetap memakai HTTPS dan cookie Secure.

## 3. Jalankan konfigurasi baru

Dari folder proyek pada laptop:

```powershell
docker compose -f compose.yaml up -d --build app
docker compose -f compose.yaml ps
```

Tunggu app healthy. Jangan menjalankan setup atau membuat ulang database jika instalasi sebelumnya sudah berjalan. Buka alamat IP yang sama di browser laptop untuk memastikan aplikasi merespons.

## 4. Buka dari HP

Jika IPv4 laptop seperti contoh, buka:

- Halaman publik: `http://192.168.1.10:8080`
- Area staf: `http://192.168.1.10:8080/admin`

Gunakan `http`, bukan `https`, untuk konfigurasi lokal ini. `localhost` di HP menunjuk HP sendiri, sehingga bukan alamat laptop. Gunakan alamat IP tersebut juga ketika mencoba dari laptop; origin `localhost` tidak lagi menjadi origin yang dipilih pada langkah 2.

## 5. Jika HP belum dapat terhubung

Pada Windows, pastikan profil Wi-Fi rumah Anda adalah **Private**, melalui Settings → Network & internet → Wi-Fi → jaringan yang tersambung → Network profile type. Gunakan pengaturan ini pada jaringan pribadi yang Anda percayai.

Buka **PowerShell → Run as administrator**, lalu jalankan satu kali:

```powershell
New-NetFirewallRule -DisplayName "Gilir LAN 8080" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8080 -RemoteAddress LocalSubnet -Profile Private
```

Rule ini mengizinkan koneksi TCP 8080 dari subnet lokal pada profil Private. Tidak perlu mematikan Windows Firewall atau melakukan port forwarding router.

Jika masih gagal, periksa IP laptop, status container app, profil jaringan, VPN, atau isolasi perangkat pada Wi-Fi. Uji dengan Wi-Fi pribadi lain bila guest Wi-Fi memblokir komunikasi antarperangkat.

Jika muncul `ORIGIN_DENIED` saat login/booking, pastikan `PUBLIC_ORIGIN` identik dengan alamat browser, termasuk skema HTTP dan port, lalu jalankan ulang perintah Compose pada langkah 3.

## Setelah selesai

Jika IP laptop berubah, perbarui `PUBLIC_ORIGIN` dan jalankan ulang Compose. Untuk kembali membatasi akses hanya dari laptop, kembalikan BIND_ADDRESS ke `127.0.0.1` serta PUBLIC_ORIGIN ke `http://localhost:8080`, lalu jalankan ulang Compose.

Rule firewall percobaan dapat dihapus dari PowerShell administrator:

```powershell
Remove-NetFirewallRule -DisplayName "Gilir LAN 8080"
```

Cara ini untuk pengujian melalui jaringan lokal. Akses dari luar jaringan membutuhkan tunnel atau deployment tersendiri.

Referensi: [port publishing Docker](https://docs.docker.com/engine/network/port-publishing/), [rule Windows Firewall](https://learn.microsoft.com/en-us/powershell/module/netsecurity/new-netfirewallrule), [Web Crypto getRandomValues](https://developer.mozilla.org/en-US/docs/Web/API/Crypto/getRandomValues).
