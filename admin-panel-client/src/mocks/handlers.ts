import { delay, http, HttpResponse } from 'msw'
import { base, type ApiError } from '@/lib/api'
import type { AccountCreate, AccountPublic, Login, LoginSuccessResponse } from '@/lib/account'
import { isValidTime, type Time } from '@/lib/time'
import {
  PAGE_SIZES, POSSIBLE_STATUSES, SORT_FIELDS, defaultSortDirection,
  type AdminTrackingResponse, type GoodStatusName, type TrackingCreate, type TrackingDetails, type SortDirection, type SortField, type Status, type StatusCreate, type Tracking,
  type TrackingListResponse,
} from '@/lib/tracking'
import { accounts, sortStatuses, trackings, type MockAccount, type MockTracking } from './data'

// Mutable copies: changes live until the page reloads.
const db = {
  accounts: structuredClone(accounts),
  trackings: structuredClone(trackings),
}

const ACCEPTED_BUFFER_MS = 5 * 60_000
const LATENCY_MS = 300

// ---------------------------------------------------------------------------
// Session. A real server uses an HttpOnly cookie; the mock remembers the logged-in
// account id in localStorage so a page reload keeps you logged in.
// ---------------------------------------------------------------------------
const SESSION_KEY = 'mock-session-account-id'

function getSessionAccount(): MockAccount | null {
  let id: string | null = null
  try { id = localStorage.getItem(SESSION_KEY) } catch { /* storage unavailable */ }
  return db.accounts.find((a) => a.id === id) ?? null
}

function setSession(id: string | null) {
  try {
    if (id == null) localStorage.removeItem(SESSION_KEY)
    else localStorage.setItem(SESSION_KEY, id)
  } catch { /* storage unavailable */ }
}

function apiError(errorCode: number, errorMsg: string) {
  const body: ApiError = { errorCode, errorMsg }
  return HttpResponse.json(body, { status: errorCode })
}

const unauthorized = () => apiError(401, 'Not logged in or session expired')

// ---------------------------------------------------------------------------
// Tracking list: filtering, sorting, pagination per requirements/api-shape.md
// ---------------------------------------------------------------------------

// Case- and accent-insensitive. `đ` has no Unicode decomposition, so map it by hand.
function norm(s: string): string {
  return s.normalize('NFD').replace(/\p{Diacritic}/gu, '').replace(/đ/gi, 'd').toLowerCase()
}

function toTracking(m: MockTracking): Tracking {
  return { ...m.details, lastStatus: m.statuses[m.statuses.length - 1] }
}

type SortKey = SortField | 'lastStatusCreationTime'

function keyValue(t: Tracking, key: SortKey): string {
  switch (key) {
    case 'trackingNumber': return norm(t.trackingNumber)
    case 'pol': return norm(t.pol)
    case 'pod': return norm(t.pod)
    case 'lastStatusOccurringTime': return t.lastStatus.occurringTime
    case 'lastStatusCreationTime': return t.lastStatus.creationTime
    case 'creationTime': return t.creationTime
  }
}

// The `desc` ordering: sortBy DESC, then fixed tiebreakers; repeated keys are skipped.
function comparator(sortBy: SortField, direction: SortDirection) {
  const keys: [SortKey, 1 | -1][] = []
  const candidates: [SortKey, 1 | -1][] = [
    [sortBy, -1],
    ['lastStatusOccurringTime', -1],
    ['lastStatusCreationTime', -1],
    ['creationTime', -1],
    ['trackingNumber', 1],
  ]
  for (const c of candidates) {
    if (!keys.some(([k]) => k === c[0])) keys.push(c)
  }
  const flip = direction === 'asc' ? -1 : 1
  return (a: Tracking, b: Tracking) => {
    for (const [key, dir] of keys) {
      const x = keyValue(a, key)
      const y = keyValue(b, key)
      if (x !== y) return (x < y ? -1 : 1) * dir * flip
    }
    return 0
  }
}

type ParsedQuery = {
  trackingNumber: string, pol: string, pod: string,
  creationTimeMin: Time, creationTimeMax: Time,
  lastStatusOccurringTimeMin: Time, lastStatusOccurringTimeMax: Time,
  sortBy: SortField, sortDirection: SortDirection,
  page: number, pageSize: number,
}

function parseListQuery(params: URLSearchParams): ParsedQuery | string {
  const now = Date.now()
  const nowIso = new Date(now).toISOString()

  const time = (name: string, fallback: Time): Time | string => {
    const v = params.get(name)
    if (v == null) return fallback
    if (!isValidTime(v)) return `${name} is not a valid time`
    if (Date.parse(v) > now + ACCEPTED_BUFFER_MS) return `${name} is in the future`
    return v
  }
  const int = (name: string, fallback: number): number | string => {
    const v = params.get(name)
    if (v == null) return fallback
    return /^-?\d+$/.test(v) ? Number(v) : `${name} must be an integer`
  }

  const creationTimeMin = time('creationTimeMin', '1970-01-01T00:00:00.000Z')
  const creationTimeMax = time('creationTimeMax', nowIso)
  const lastStatusOccurringTimeMin = time('lastStatusOccurringTimeMin', '1970-01-01T00:00:00.000Z')
  const lastStatusOccurringTimeMax = time('lastStatusOccurringTimeMax', nowIso)
  for (const v of [creationTimeMin, creationTimeMax, lastStatusOccurringTimeMin, lastStatusOccurringTimeMax]) {
    if (!isValidTime(v)) return v
  }
  if (creationTimeMin > creationTimeMax) return 'creationTimeMin is after creationTimeMax'
  if (lastStatusOccurringTimeMin > lastStatusOccurringTimeMax) {
    return 'lastStatusOccurringTimeMin is after lastStatusOccurringTimeMax'
  }

  const sortByRaw = params.get('sortBy') ?? 'lastStatusOccurringTime'
  if (!(SORT_FIELDS as readonly string[]).includes(sortByRaw)) return `sortBy "${sortByRaw}" is not supported`
  const sortBy = sortByRaw as SortField

  const dirRaw = params.get('sortDirection') ?? defaultSortDirection(sortBy)
  if (dirRaw !== 'asc' && dirRaw !== 'desc') return 'sortDirection must be "asc" or "desc"'

  const page = int('page', 1)
  if (typeof page === 'string') return page
  if (page < 1) return 'page must be at least 1'

  const pageSize = int('pageSize', 20)
  if (typeof pageSize === 'string') return pageSize
  if (!(PAGE_SIZES as readonly number[]).includes(pageSize)) return `pageSize must be one of ${PAGE_SIZES.join(', ')}`

  return {
    trackingNumber: params.get('trackingNumber') ?? '',
    pol: params.get('pol') ?? '',
    pod: params.get('pod') ?? '',
    creationTimeMin, creationTimeMax, lastStatusOccurringTimeMin, lastStatusOccurringTimeMax,
    sortBy, sortDirection: dirRaw, page, pageSize,
  }
}

function listTrackings(q: ParsedQuery): TrackingListResponse | string {
  const matches = db.trackings.map(toTracking).filter((t) =>
    norm(t.trackingNumber).includes(norm(q.trackingNumber))
    && norm(t.pol).startsWith(norm(q.pol))
    && norm(t.pod).startsWith(norm(q.pod))
    && t.creationTime >= q.creationTimeMin && t.creationTime <= q.creationTimeMax
    && t.lastStatus.occurringTime >= q.lastStatusOccurringTimeMin
    && t.lastStatus.occurringTime <= q.lastStatusOccurringTimeMax,
  )
  matches.sort(comparator(q.sortBy, q.sortDirection))

  const pageCount = Math.ceil(matches.length / q.pageSize)
  if (pageCount < q.page && !(pageCount === 0 && q.page === 1)) {
    return `page ${q.page} is past the last page (${pageCount})`
  }
  const start = (q.page - 1) * q.pageSize
  return {
    trackings: matches.slice(start, start + q.pageSize),
    page: q.page,
    pageCount,
    pageSize: q.pageSize,
  }
}

// ---------------------------------------------------------------------------
// Status creation
// ---------------------------------------------------------------------------
function validateStatusCreate(body: unknown): StatusCreate | string {
  if (typeof body !== 'object' || body === null) return 'Body must be a JSON object'
  const b = body as Record<string, unknown>
  if (typeof b.location !== 'string' || b.location.trim() === '') return 'location must be a nonempty string'
  if (typeof b.occurringTime !== 'string' || !isValidTime(b.occurringTime)) return 'occurringTime is not a valid time'
  if (Date.parse(b.occurringTime) > Date.now() + ACCEPTED_BUFFER_MS) return 'occurringTime must be in the past'
  const common = { occurringTime: b.occurringTime, location: b.location.trim() }

  switch (b.statusType) {
    case 'good': {
      if (typeof b.status !== 'string' || !POSSIBLE_STATUSES.has(b.status)) return 'status is not a known good status'
      if (!Array.isArray(b.warnings) || !b.warnings.every((w) => typeof w === 'string' && w.trim() !== '')) {
        return 'warnings must be an array of nonempty strings'
      }
      return { ...common, statusType: 'good', status: b.status as GoodStatusName, warnings: b.warnings.map((w: string) => w.trim()) }
    }
    case 'bad':
      if (typeof b.status !== 'string' || b.status.trim() === '') return 'status must be a nonempty string'
      return { ...common, statusType: 'bad', status: b.status.trim() }
    default:
      return 'statusType must be "good" or "bad"'
  }
}

let createdStatusSeq = 0

// ---------------------------------------------------------------------------
// Tracking creation (mirrors server/internal/api/trackingcreate.go)
// ---------------------------------------------------------------------------
const TRACKING_ALPHABET = '23456789ABCDEFGHJKLMNPQRSTUVWXYZ'

function newTrackingNumber(): string {
  const r = crypto.getRandomValues(new Uint8Array(8))
  const sym = Array.from(r, (b) => TRACKING_ALPHABET[b & 31]).join('')
  return `BLJ-${sym.slice(0, 4)}-${sym.slice(4)}`
}

const positive = (v: unknown) => typeof v === 'number' && Number.isFinite(v) && v > 0 && v <= 1e9

function validateTrackingCreate(body: unknown): TrackingCreate | string {
  if (typeof body !== 'object' || body === null) return 'Body must be a JSON object'
  const b = body as Record<string, unknown>
  for (const k of Object.keys(b)) {
    if (!['cont', 'seal', 'pol', 'pod', 'physicalDetails'].includes(k)) return `Unknown field ${k}`
  }
  const text = (v: unknown) => (typeof v === 'string' ? v.trim() : undefined)
  const pol = text(b.pol)
  const pod = text(b.pod)
  if (!pol) return 'pol must not be empty'
  if (!pod) return 'pod must not be empty'
  const out: TrackingCreate = { pol, pod }
  if (text(b.cont)) out.cont = text(b.cont)
  if (text(b.seal)) out.seal = text(b.seal)
  if (b.physicalDetails != null) {
    const p = b.physicalDetails as Record<string, unknown>
    const d = (p.dimensions ?? {}) as Record<string, unknown>
    if (!positive(p.weightKg)) return 'physicalDetails.weightKg must be a positive number'
    if (!positive(d.lengthCm) || !positive(d.widthCm) || !positive(d.heightCm)) {
      return 'physicalDetails.dimensions needs positive lengthCm, widthCm and heightCm'
    }
    if (p.pcsCount != null && !(Number.isInteger(p.pcsCount) && (p.pcsCount as number) >= 1)) {
      return 'physicalDetails.pcsCount must be a positive integer'
    }
    out.physicalDetails = {
      weightKg: p.weightKg as number,
      dimensions: { lengthCm: d.lengthCm as number, widthCm: d.widthCm as number, heightCm: d.heightCm as number },
      ...(p.pcsCount != null ? { pcsCount: p.pcsCount as number } : {}),
    }
  }
  return out
}

// ---------------------------------------------------------------------------
// Accounts
// ---------------------------------------------------------------------------
function validateAccountCreate(body: unknown): AccountCreate | string {
  if (typeof body !== 'object' || body === null) return 'Body must be a JSON object'
  const b = body as Record<string, unknown>
  if (typeof b.email !== 'string' || !/^[^\s@]+@[^\s@]+$/.test(b.email)) return 'email is not valid'
  if (typeof b.password !== 'string' || b.password.length < 8) return 'password must be at least 8 characters'
  if (typeof b.name !== 'string' || b.name.trim() === '') return 'name must be nonempty'
  if (b.role !== 'admin' && b.role !== 'user') return 'role must be "admin" or "user"'
  return { email: b.email, password: b.password, name: b.name.trim(), role: b.role }
}

// Good-enough UUIDv7 shape for a mock: 48-bit ms timestamp, version 7, variant 10.
function uuidv7(): string {
  const hex = Date.now().toString(16).padStart(12, '0')
  const rand = crypto.getRandomValues(new Uint8Array(10))
  const r = Array.from(rand, (x) => x.toString(16).padStart(2, '0')).join('')
  const variant = ((rand[2] & 0x3f) | 0x80).toString(16)
  return `${hex.slice(0, 8)}-${hex.slice(8)}-7${r.slice(0, 3)}-${variant}${r.slice(6, 8)}-${r.slice(8, 20)}`
}

export const handlers = [
  http.post(`${base}/admin/login`, async ({ request }) => {
    await delay(LATENCY_MS)
    const body = await request.json().catch(() => null) as Partial<Login> | null
    const account = db.accounts.find((a) =>
      a.email.toLowerCase() === String(body?.email ?? '').toLowerCase() && a.password === body?.password)
    if (!account) return apiError(401, 'Invalid email or password')
    setSession(account.id)
    const res: LoginSuccessResponse = { name: account.name, account_created: account.account_create_time, role: account.role }
    return HttpResponse.json(res)
  }),

  http.post(`${base}/admin/logout`, async () => {
    await delay(LATENCY_MS)
    setSession(null)
    return HttpResponse.json({})
  }),

  http.get(`${base}/admin/me`, async () => {
    await delay(LATENCY_MS)
    const me = getSessionAccount()
    if (!me) return unauthorized()
    const res: LoginSuccessResponse = { name: me.name, account_created: me.account_create_time, role: me.role }
    return HttpResponse.json(res)
  }),

  http.get(`${base}/admin/tracking`, async ({ request }) => {
    await delay(LATENCY_MS)
    if (!getSessionAccount()) return unauthorized()
    const q = parseListQuery(new URL(request.url).searchParams)
    if (typeof q === 'string') return apiError(400, q)
    const res = listTrackings(q)
    if (typeof res === 'string') return apiError(400, res)
    return HttpResponse.json(res)
  }),

  http.post(`${base}/admin/tracking`, async ({ request }) => {
    await delay(LATENCY_MS)
    const me = getSessionAccount()
    if (!me) return unauthorized()
    const input = validateTrackingCreate(await request.json().catch(() => null))
    if (typeof input === 'string') return apiError(400, input)
    const now = new Date().toISOString()
    const details: TrackingDetails = { ...input, trackingNumber: newTrackingNumber(), creatorId: me.id, creationTime: now }
    const first: Status = {
      id: `status-new-${++createdStatusSeq}`, statusType: 'good', status: 'Tracking created', warnings: [],
      occurringTime: now, location: input.pol, creatorId: me.id, creationTime: now,
    }
    db.trackings.push({ details, statuses: [first] })
    const res: AdminTrackingResponse = { details, statuses: [first] }
    return HttpResponse.json(res)
  }),

  http.get(`${base}/admin/tracking/:trackingNumber`, async ({ params }) => {
    await delay(LATENCY_MS)
    if (!getSessionAccount()) return unauthorized()
    const found = db.trackings.find((t) => t.details.trackingNumber === params.trackingNumber)
    if (!found) return apiError(404, `No tracking found with number ${String(params.trackingNumber)}`)
    const res: AdminTrackingResponse = { details: found.details, statuses: found.statuses }
    return HttpResponse.json(res)
  }),

  http.post(`${base}/admin/tracking/:trackingNumber/status`, async ({ params, request }) => {
    await delay(LATENCY_MS)
    const me = getSessionAccount()
    if (!me) return unauthorized()
    const found = db.trackings.find((t) => t.details.trackingNumber === params.trackingNumber)
    if (!found) return apiError(404, `No tracking found with number ${String(params.trackingNumber)}`)
    const input = validateStatusCreate(await request.json().catch(() => null))
    if (typeof input === 'string') return apiError(400, input)
    const status: Status = {
      ...input,
      id: `status-new-${++createdStatusSeq}`,
      creatorId: me.id,
      creationTime: new Date().toISOString(),
    }
    found.statuses = sortStatuses([...found.statuses, status])
    return HttpResponse.json(status)
  }),

  http.get(`${base}/admin/account/:id`, async ({ params }) => {
    await delay(LATENCY_MS)
    if (!getSessionAccount()) return unauthorized()
    const account = db.accounts.find((a) => a.id === params.id)
    if (!account) return apiError(404, 'Account not found')
    const res: AccountPublic = { id: account.id, name: account.name }
    return HttpResponse.json(res)
  }),

  http.post(`${base}/admin/createAccount`, async ({ request }) => {
    await delay(LATENCY_MS)
    const me = getSessionAccount()
    if (!me || me.role !== 'admin') return unauthorized()
    const input = validateAccountCreate(await request.json().catch(() => null))
    if (typeof input === 'string') return apiError(400, input)
    if (db.accounts.some((a) => a.email.toLowerCase() === input.email.toLowerCase())) {
      return apiError(409, 'An account with this email already exists')
    }
    // Lets you see the 500 path in the UI.
    if (input.email.toLowerCase().startsWith('error500@')) return apiError(500, 'Internal server error')
    const account: MockAccount = { ...input, id: uuidv7(), account_create_time: new Date().toISOString() }
    db.accounts.push(account)
    const res: AccountPublic = { id: account.id, name: account.name }
    return HttpResponse.json(res)
  }),

  http.post(`${base}/admin/deleteAccount`, async ({ request }) => {
    await delay(LATENCY_MS)
    const me = getSessionAccount()
    if (!me || me.role !== 'admin') return unauthorized()
    const body = await request.json().catch(() => null) as Record<string, unknown> | null
    const index = typeof body?.email === 'string'
      ? db.accounts.findIndex((a) => a.email.toLowerCase() === String(body.email).toLowerCase())
      : typeof body?.id === 'string'
        ? db.accounts.findIndex((a) => a.id === body.id)
        : -2
    if (index === -2) return apiError(400, 'Provide either an email or an id')
    if (index === -1) return apiError(404, 'Account not found')
    const [removed] = db.accounts.splice(index, 1)
    // All sessions of a deleted account are deleted too.
    if (removed.id === me.id) setSession(null)
    return HttpResponse.json({})
  }),
]
