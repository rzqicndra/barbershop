import { serverNow } from './clock'

export type Service = {
  id: string
  name: string
  description: string
  price: number
  duration: number
  active: boolean
  version: number
}
export type Barber = {
  id: string
  name: string
  specialty: string
  serviceIds: string[]
  start: number
  end: number
  breakStart: number
  breakEnd: number
  daysOff: number[]
  active: boolean
  version: number
}
export type TimeOff = {
  id: string
  barberId: string
  date: string
  start: number
  end: number
  reason: string
}
export type BookingStatus = 'confirmed' | 'serving' | 'completed' | 'cancelled' | 'no-show'
export type Role = 'admin' | 'barber'
export type SlotMode = 'online' | 'walk-in' | 'admin'
export type History = { action: string; by: string; at: number; detail?: string }
export type Booking = {
  id: string
  token?: string
  code: string
  date: string
  start: number
  serviceId: string
  serviceName: string
  duration: number
  price: number
  barberId: string
  customer: string
  phone?: string
  source: 'online' | 'walk-in'
  status: BookingStatus
  version: number
  history?: History[]
}
export type User = {
  id: string
  email: string
  name: string
  role: Role
  barberId?: string
  active: boolean
}
export type Studio = {
  profile: {
    name: string
    location: string
    open: number
    close: number
    closedDates: string[]
    version: number
  }
  services: Service[]
  barbers: Barber[]
  timeOff: TimeOff[]
}
export type StudioProps = { studio: Studio; refresh: () => Promise<void> }
export const ZONE = 'Asia/Jakarta'
export const statuses: Record<BookingStatus, string> = {
  confirmed: 'Dikonfirmasi',
  serving: 'Sedang dilayani',
  completed: 'Selesai',
  cancelled: 'Dibatalkan',
  'no-show': 'Tidak hadir',
}
const dayFormat = new Intl.DateTimeFormat('en-CA', {
  timeZone: ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
})
const minuteFormat = new Intl.DateTimeFormat('en-GB', {
  timeZone: ZONE,
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})
export const dateFormat = (date: string, options: Intl.DateTimeFormatOptions) =>
  new Intl.DateTimeFormat('id-ID', { timeZone: ZONE, ...options }).format(
    new Date(date + 'T12:00:00+07:00'),
  )
export const today = (now = serverNow()) => dayFormat.format(now)
export const minutesNow = (now = serverNow()) => parseTime(minuteFormat.format(now))
export const addDays = (date: string, days: number) =>
  today(Date.parse(date + 'T12:00:00+07:00') + days * 86_400_000)
export const longDate = (date: string) =>
  dateFormat(date, { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
export const shortDate = (date: string) => dateFormat(date, { day: 'numeric', month: 'long' })
export const timeInput = (minutes: number) =>
  [Math.floor(minutes / 60), minutes % 60].map((n) => String(n).padStart(2, '0')).join(':')
export const time = (minutes: number) => timeInput(minutes).replace(':', '.')
export const money = (value: number) => 'Rp' + new Intl.NumberFormat('id-ID').format(value)
export const timestamp = (date: string, start: number) =>
  Date.parse(date + 'T00:00:00+07:00') + start * 60_000
export function parseTime(value: string) {
  const [hour, minute] = value.split(':').map(Number)
  return hour * 60 + minute
}
export function suitableBarbers(studio: Studio, serviceId: string) {
  return studio.barbers.filter((b) => b.active && b.serviceIds.includes(serviceId))
}
export function validPhone(value: string) {
  return /^(?:\+62|62|0)8\d{8,11}$/.test(value.replace(/[\s()-]/g, ''))
}
export function canManage(booking: Booking, actor: 'customer' | 'admin', now = serverNow()) {
  const remaining = timestamp(booking.date, booking.start) - now
  return (
    booking.status === 'confirmed' &&
    (actor === 'customer' ? remaining >= 2 * 60 * 60_000 : remaining > 0)
  )
}
export function statusActions(booking: Booking, now = serverNow()): BookingStatus[] {
  if (booking.status === 'serving') return ['completed']
  if (booking.status !== 'confirmed') return []
  const start = timestamp(booking.date, booking.start)
  if (now < start) return []
  return now >= start + booking.duration * 60_000 ? ['serving', 'no-show'] : ['serving']
}
