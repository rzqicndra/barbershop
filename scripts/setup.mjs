import { randomBytes } from 'node:crypto'
import { writeFile, access } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
const target = fileURLToPath(new URL('../.env', import.meta.url))
try {
  await access(target)
  console.log('.env sudah tersedia. Nilai lama dipertahankan.')
} catch (error) {
  if (error.code !== 'ENOENT') throw error
  await writeFile(target, [
    '# Rahasia lokal. Jangan commit atau kirim file ini.',
    'DB_PASSWORD=' + randomBytes(24).toString('hex'),
    'APP_DB_PASSWORD=' + randomBytes(24).toString('hex'),
    'TOKEN_KEY=' + randomBytes(32).toString('hex'),
    'PUBLIC_ORIGIN=http://localhost:8080',
    'COOKIE_SECURE=false',
    'BIND_ADDRESS=127.0.0.1',
    'APP_PORT=8080',
    '',
  ].join('\n'), { flag: 'wx', mode: 0o600 })
  console.log('.env berhasil dibuat. Lanjutkan panduan README.md.')
}
