import { afterEach, expect, it, vi } from 'vitest'
import { randomId } from './random-id'

afterEach(() => vi.unstubAllGlobals())

it('generates distinct UUIDv4 identifiers when an HTTP browser has no randomUUID', () => {
  const getRandomValues = globalThis.crypto.getRandomValues.bind(globalThis.crypto)
  vi.stubGlobal('crypto', { getRandomValues })

  const identifiers = Array.from({ length: 32 }, () => randomId())
  for (const identifier of identifiers) {
    expect(identifier).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
    )
  }
  expect(new Set(identifiers).size).toBe(identifiers.length)
})

it('uses the native UUID generator when the browser provides it', () => {
  const identifier = '4c5334d3-8984-41fa-b895-b534229d3f2a'
  const randomUUID = vi.fn(() => identifier)
  vi.stubGlobal('crypto', { randomUUID })

  expect(randomId()).toBe(identifier)
  expect(randomUUID).toHaveBeenCalledOnce()
})
