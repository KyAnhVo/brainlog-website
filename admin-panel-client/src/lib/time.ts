// UTC, RFC 3339, millisecond precision, `Z` suffix: yyyy-mm-ddThh:mm:ss.xxxZ
type Time = string

const TIME_SHAPE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/

// Date.parse rolls out-of-range fields over (02-30 -> 03-02, 24:00 -> next day),
// so the round trip must reproduce the input exactly.
function isValidTime(s: string): boolean {
  if (!TIME_SHAPE.test(s)) return false
  const t = Date.parse(s)
  return !Number.isNaN(t) && new Date(t).toISOString() === s
}

// <input type="datetime-local"> works in the viewer's local time ("yyyy-mm-ddThh:mm").
function toLocalInput(t: Time | undefined): string {
  if (!t) return ''
  const d = new Date(t)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Inverse of toLocalInput. Empty or unparsable input gives `undefined`.
function fromLocalInput(s: string): Time | undefined {
  if (!s) return undefined
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? undefined : d.toISOString()
}

function formatTime(t: Time): string {
  return new Date(t).toLocaleString()
}

export { formatTime, fromLocalInput, isValidTime, toLocalInput }
export type { Time }
