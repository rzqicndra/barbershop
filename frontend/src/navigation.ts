import type { Role, User } from './data'

export function staffHome(role: Role) {
  return role === 'admin' ? '/admin/agenda' : '/admin/jadwal'
}

export function navigationDestination(path: string, user: User | null, legacy = false) {
  const staff = path === 'admin' || path.startsWith('admin/')
  if (user) {
    const allowed =
      user.role === 'admin'
        ? ['admin/agenda', 'admin/pengaturan', 'admin/akun']
        : ['admin/jadwal', 'admin/akun']
    if (!staff || !allowed.includes(path)) return staffHome(user.role)
  } else if (staff && path !== 'admin') return '/admin'
  return legacy ? '/' + path : ''
}
