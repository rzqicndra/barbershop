let offset = 0
export function syncClock(value: string | null) {
  if (!value) return
  const parsed = Date.parse(value)
  if (Number.isFinite(parsed)) offset = parsed - Date.now()
}
export const serverNow = () => Date.now() + offset
