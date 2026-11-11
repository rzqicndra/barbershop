import '@testing-library/jest-dom/vitest'
import { afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'

Object.defineProperty(window, 'scrollTo', { value: vi.fn(), writable: true })
window.requestAnimationFrame = (callback) => window.setTimeout(() => callback(performance.now()), 0)
window.cancelAnimationFrame = (id) => window.clearTimeout(id)
HTMLDialogElement.prototype.showModal = function () {
  this.setAttribute('open', '')
}
HTMLDialogElement.prototype.close = function () {
  this.removeAttribute('open')
}
afterEach(() => cleanup())
