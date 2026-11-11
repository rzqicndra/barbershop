import { StrictMode, useCallback, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource-variable/manrope'
import '@fontsource-variable/space-grotesk'
import './styles.css'
import type { Studio, User } from './data'
import { api, APIError, getSession, messageOf, setCSRF } from './api'
import { Home } from './public'
import { BookingTicket, BookingWizard } from './booking'
import { Agenda, Settings, StaffLogin, StaffShell } from './staff'
import { AccountPage } from './staff-settings'

function currentRoute() {
  const pathname = window.location.pathname.replace(/^\/+|\/+$/g, '')
  const hash = window.location.hash.slice(1) || 'beranda'
  const split = hash.indexOf('?')
  const hashPath = split < 0 ? hash : hash.slice(0, split)
  const legacy = hashPath === 'staf' || hashPath.startsWith('staf/')
  const staff = pathname === 'admin' || pathname.startsWith('admin/')
  return {
    path: staff ? pathname : legacy ? 'admin' + hashPath.slice(4) : hashPath,
    params: new URLSearchParams(split < 0 ? '' : hash.slice(split + 1)),
    legacy,
  }
}
function App() {
  const [studio, setStudio] = useState<Studio | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const [ready, setReady] = useState(false)
  const [error, setError] = useState('')
  const [route, setRoute] = useState(currentRoute)
  const [, setClock] = useState(0)
  const staffRoute = route.path === 'admin' || route.path.startsWith('admin/')
  const refresh = useCallback(async () => {
    setStudio(await api<Studio>(user?.role === 'admin' ? '/admin/catalog' : '/public'))
  }, [user?.role])
  const boot = useCallback(async () => {
    setError('')
    try {
      const catalog = await api<Studio>('/public')
      setStudio(catalog)
      try {
        setUser(await getSession())
      } catch (err) {
        if (!(err instanceof APIError && err.status === 401)) throw err
      }
      setReady(true)
    } catch (err) {
      setError(messageOf(err))
    }
  }, [])
  useEffect(() => {
    void boot()
  }, [boot])
  useEffect(() => {
    if (ready) refresh().catch((err) => setError(messageOf(err)))
  }, [ready, refresh])
  useEffect(() => {
    const change = () => setRoute(currentRoute())
    const end = () => {
      setUser(null)
      setCSRF('')
    }
    const timer = window.setInterval(() => setClock((value) => value + 1), 30_000)
    window.addEventListener('hashchange', change)
    window.addEventListener('popstate', change)
    window.addEventListener('gilir:session-ended', end)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('hashchange', change)
      window.removeEventListener('popstate', change)
      window.removeEventListener('gilir:session-ended', end)
    }
  }, [])
  function navigate(path: string, replace = false) {
    window.history[replace ? 'replaceState' : 'pushState'](null, '', path)
    setRoute(currentRoute())
  }
  useEffect(() => {
    if (!ready) return
    let destination = route.legacy ? '/' + route.path : ''
    if (staffRoute) {
      if (!user && route.path !== 'admin') destination = '/admin'
      if (
        user?.role === 'admin' &&
        !['admin/agenda', 'admin/pengaturan', 'admin/akun'].includes(route.path)
      )
        destination = '/admin/agenda'
      if (user?.role === 'barber' && !['admin/jadwal', 'admin/akun'].includes(route.path))
        destination = '/admin/jadwal'
    }
    if (destination) navigate(destination, true)
  }, [ready, route.path, route.legacy, staffRoute, user])
  useEffect(() => {
    requestAnimationFrame(() => {
      if (['layanan', 'barber', 'lokasi'].includes(route.path))
        document.getElementById(route.path)?.scrollIntoView({ behavior: 'smooth' })
      else window.scrollTo({ top: 0, behavior: 'instant' })
    })
    document.title = staffRoute
      ? 'Area staf | Gilir Barbershop'
      : route.path.startsWith('booking')
        ? 'Booking | Gilir Barbershop'
        : 'Gilir Barbershop | Pilih jam. Potong rapi.'
  }, [route.path, staffRoute, ready])
  async function signOut() {
    try {
      await api('/auth/logout', { method: 'POST' })
      setUser(null)
      setCSRF('')
      navigate('/admin', true)
    } catch (err) {
      setError(messageOf(err))
    }
  }
  if (!studio || !ready)
    return (
      <main className="narrow-wrap loading-page" id="konten">
        <h1>Gilir Barbershop</h1>
        <p role="status">{error || 'Menyiapkan jadwal…'}</p>
        {error && (
          <button className="button button-primary" onClick={() => void boot()}>
            Coba kembali
          </button>
        )}
      </main>
    )
  const data = { studio, refresh }
  let page
  if (route.path === 'booking')
    page = (
      <BookingWizard
        key={route.params.get('layanan') ?? ''}
        {...data}
        initialService={route.params.get('layanan') ?? ''}
      />
    )
  else if (route.path.startsWith('booking/'))
    page = (
      <BookingTicket
        key={route.path.slice(8)}
        {...data}
        token={route.path.slice(8)}
        created={route.params.get('baru') === '1'}
      />
    )
  else if (staffRoute) {
    if (!user)
      page = (
        <StaffLogin
          signIn={(value) => {
            setUser(value)
            navigate(value.role === 'admin' ? '/admin/agenda' : '/admin/jadwal')
          }}
        />
      )
    else
      page = (
        <StaffShell user={user} route={route.path} signOut={() => void signOut()}>
          {route.path === 'admin/akun' ? (
            <AccountPage
              user={user}
              onChanged={() => {
                setUser(null)
                setCSRF('')
                navigate('/admin', true)
              }}
            />
          ) : route.path === 'admin/pengaturan' && user.role === 'admin' ? (
            <Settings {...data} />
          ) : (
            <Agenda {...data} role={user.role} barberId={user.barberId ?? ''} />
          )}
        </StaffShell>
      )
  } else page = <Home studio={studio} />
  return (
    <>
      <a
        className="skip-link"
        href="#konten"
        onClick={(event) => {
          event.preventDefault()
          document.getElementById('konten')?.focus()
        }}
      >
        Lewati ke konten
      </a>
      {error && (
        <div className="notice notice-error global-error" role="alert">
          {error}
          <button className="text-link" onClick={() => setError('')}>
            Tutup
          </button>
        </div>
      )}
      {page}
    </>
  )
}
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
