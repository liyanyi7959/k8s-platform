const DECIMAL_UNITS: Record<string, number> = {
  n: 1e-9,
  u: 1e-6,
  m: 1e-3,
  '': 1,
  k: 1e3,
  K: 1e3,
  M: 1e6,
  G: 1e9,
  T: 1e12,
  P: 1e15,
  E: 1e18
}

const BINARY_UNITS: Record<string, number> = {
  Ki: 1024,
  Mi: 1024 ** 2,
  Gi: 1024 ** 3,
  Ti: 1024 ** 4,
  Pi: 1024 ** 5,
  Ei: 1024 ** 6
}

const SI_VALUE_RE = /^([+-]?(?:\d+(?:\.\d+)?|\.\d+))(?:([a-zA-Z]{0,2}))?$/

export function parseSiValue(value: unknown): number {
  if (typeof value === 'number') return Number.isFinite(value) ? value : Number.NaN
  const raw = String(value ?? '').trim()
  if (!raw) return Number.NaN

  const match = raw.match(SI_VALUE_RE)
  if (!match) return Number.NaN

  const numeric = Number(match[1])
  if (!Number.isFinite(numeric)) return Number.NaN

  const unit = match[2] ?? ''
  if (unit in BINARY_UNITS) return numeric * BINARY_UNITS[unit]
  if (unit in DECIMAL_UNITS) return numeric * DECIMAL_UNITS[unit]
  return Number.NaN
}