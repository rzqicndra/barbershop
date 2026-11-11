#!/usr/bin/env bash
set -euo pipefail
umask 077
task_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$task_root"
task_compose=(docker compose)
if [[ "${1:-}" == "--production" ]]; then
    task_compose+=(-f compose.yaml -f compose.production.yaml)
elif [[ $# -gt 0 ]]; then
    echo 'Gunakan: bash scripts/backup.sh [--production]' >&2
    exit 2
fi
task_stamp=$(date -u +%Y%m%d-%H%M%S)
task_partial="$task_root/backups/.partial-gilir-$task_stamp-$$"
task_final="$task_root/backups/gilir-$task_stamp-$$"
mkdir -p "$task_partial/uploads"
trap 'echo "Backup belum lengkap; periksa folder .partial di backups sebelum mencoba lagi." >&2' ERR
# Take the database snapshot first. Uploaded files are immutable and are not
# physically removed when content is hidden, so copying them afterwards includes
# every file referenced by this snapshot.
"${task_compose[@]}" exec -T db pg_dump -U gilir -d gilir -Fc > "$task_partial/database.dump"
test -s "$task_partial/database.dump"
"${task_compose[@]}" cp app:/app/uploads/. "$task_partial/uploads/"
(
    cd "$task_partial"
    find database.dump uploads -type f -print0 | sort -z | xargs -0 sha256sum > SHA256SUMS
    sha256sum -c SHA256SUMS > /dev/null
)
mv -- "$task_partial" "$task_final"
trap - ERR
echo "Backup lengkap: $task_final"
echo 'Simpan juga .env secara terpisah dan salin backup ke tempat lain dengan akses terbatas.'
